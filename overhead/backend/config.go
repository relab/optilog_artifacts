// Package backend implements the networking backend for hotstuff using the Gorums framework.
package backend

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"
	"unsafe"

	"github.com/relab/hotstuff/eventloop"
	"github.com/relab/hotstuff/logging"
	"github.com/relab/hotstuff/modules"
	"github.com/relab/hotstuff/synchronizer"

	"github.com/relab/gorums"
	"github.com/relab/hotstuff"
	"github.com/relab/hotstuff/internal/proto/hotstuffpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

// Replica provides methods used by hotstuff to send messages to replicas.
type Replica struct {
	eventLoop    *eventloop.EventLoop
	node         *hotstuffpb.Node
	id           hotstuff.ID
	pubKey       hotstuff.PublicKey
	md           map[string]string
	active       bool
	locationInfo map[hotstuff.ID]string
}

// ID returns the replica's ID.
func (r *Replica) ID() hotstuff.ID {
	return r.id
}

// PublicKey returns the replica's public key.
func (r *Replica) PublicKey() hotstuff.PublicKey {
	return r.pubKey
}

// Vote sends the partial certificate to the other replica.
func (r *Replica) Vote(cert hotstuff.PartialCert) {
	if r.node == nil {
		return
	}
	ctx, cancel := synchronizer.TimeoutContext(r.eventLoop.Context(), r.eventLoop)
	defer cancel()
	pCert := hotstuffpb.PartialCertToProto(cert)
	r.node.Vote(ctx, pCert)
}

// SetActive sets the status of the replica
func (r *Replica) SetActive(active bool) {
	r.active = active
}

// NewView sends the quorum certificate to the other replica.
func (r *Replica) NewView(msg hotstuff.SyncInfo) {
	if r.node == nil {
		return
	}
	ctx, cancel := synchronizer.TimeoutContext(r.eventLoop.Context(), r.eventLoop)
	defer cancel()
	r.node.NewView(ctx, hotstuffpb.SyncInfoToProto(msg))
}

// Metadata returns the gRPC metadata from this replica's connection.
func (r *Replica) Metadata() map[string]string {
	return r.md
}

func (r *Replica) Active() bool {
	return r.active
}

// Config holds information about the current configuration of replicas that participate in the protocol,
// and some information about the local replica. It also provides methods to send messages to the other replicas.
type Config struct {
	synchronizer    modules.Synchronizer
	opts            []gorums.ManagerOption
	connected       bool
	mgr             *hotstuffpb.Manager
	isActiveReplica bool
	ranking         modules.Ranking
	subConfig
}

type subConfig struct {
	eventLoop            *eventloop.EventLoop
	logger               logging.Logger
	opts                 *modules.Options
	cfg                  *hotstuffpb.Configuration
	replicas             map[hotstuff.ID]modules.Replica
	passiveConfiguration *hotstuffpb.Configuration
	quorumMap            map[hotstuff.View]int
	locationInfo         map[hotstuff.ID]string
	mgr                  *hotstuffpb.Manager
}

// InitModule initializes the configuration.
func (cfg *Config) InitModule(mods *modules.Core) {
	mods.Get(
		&cfg.eventLoop,
		&cfg.logger,
		&cfg.subConfig.opts,
		&cfg.synchronizer,
	)
	mods.TryGet(&cfg.ranking)
	// We delay processing `replicaConnected` events until after the configurations `connected` event has occurred.
	cfg.eventLoop.RegisterHandler(replicaConnected{}, func(event any) {
		if !cfg.connected {
			cfg.eventLoop.DelayUntil(ConnectedEvent{}, event)
			return
		}
		cfg.replicaConnected(event.(replicaConnected))
	})

	cfg.eventLoop.RegisterHandler(hotstuff.ReconfigurationMsg{}, func(event any) {
		reconfigurationMsg := event.(hotstuff.ReconfigurationMsg)
		cfg.handleReconfigurationEvent(reconfigurationMsg)
	})
	cfg.eventLoop.RegisterHandler(hotstuff.CheckLatencyVector{}, func(event any) {
		checkLatencyVector := event.(hotstuff.CheckLatencyVector)
		cfg.handleLatencyVector(checkLatencyVector)
	})

}

// createSubConfiguration creates a new active and passive subconfigurations
func (cfg *Config) createSubConfiguration(activeIDs []hotstuff.ID) (sub *subConfig, err error) {

	nodeIDs := make([]uint32, 0)
	for _, id := range activeIDs {
		if id != cfg.subConfig.opts.ID() {
			nodeIDs = append(nodeIDs, uint32(id))
		}
		cfg.replicas[id].SetActive(true)
	}
	passiveNodeIDs := make([]uint32, 0)
	for id := range cfg.replicas {
		found := false
		for _, activeID := range activeIDs {
			if activeID == id {
				found = true
			}
		}
		if !found {
			if id != cfg.subConfig.opts.ID() {
				passiveNodeIDs = append(passiveNodeIDs, uint32(id))
			}
			cfg.replicas[id].SetActive(false)
		}
	}
	newCfg, err := cfg.mgr.NewConfiguration(qspec{}, gorums.WithNodeIDs(nodeIDs))

	if err != nil {
		cfg.logger.Warn("error in handling reconfiguration event", err)
		return nil, err
	}
	var passiveCfg *hotstuffpb.Configuration
	if len(passiveNodeIDs) > 0 {
		passiveCfg, err = cfg.mgr.NewConfiguration(qspec{}, gorums.WithNodeIDs(passiveNodeIDs))
		if err != nil {
			cfg.logger.Warn("error in handling reconfiguration event", err)
			return nil, err
		}
	}
	return &subConfig{
		replicas:             cfg.replicas,
		eventLoop:            cfg.eventLoop,
		logger:               cfg.logger,
		opts:                 cfg.subConfig.opts,
		cfg:                  newCfg,
		passiveConfiguration: passiveCfg,
		quorumMap:            cfg.quorumMap,
		mgr:                  cfg.mgr,
		locationInfo:         cfg.locationInfo,
	}, nil
}

func (cfg *Config) handleLatencyVector(latencyVector hotstuff.CheckLatencyVector) {
	//cfg.logger.Info("handling the latency vector")
	sender := latencyVector.Proposer
	senderLocation := cfg.locationInfo[sender]
	for _, latency := range latencyVector.LatencyVector {
		//temp := latency
		id := latency >> 24
		latencyValue := latency << 8 >> 8
		location := cfg.locationInfo[hotstuff.ID(id)]
		//cfg.logger.Info("id is ", id, sender, location)
		originalLatency := latencies[senderLocation][location].Microseconds()
		//cfg.logger.Info("Expected latency, got latency ", originalLatency, latencyValue)
		if originalLatency*3 < int64(latencyValue) && cfg.ranking != nil {
			cfg.ranking.AddComplaint(&hotstuff.Complaint{
				Complainee:    hotstuff.ID(id),
				Complainant:   sender,
				ComplaintType: hotstuff.Suspicion,
			})
		}
	}
}

// handleReconfigurationEvent handles the reconfiguration request.
func (cfg *Config) handleReconfigurationEvent(reconfigurationMsg hotstuff.ReconfigurationMsg) {
	cfg.logger.Info("handling the configuration update event")
	cfg.quorumMap[reconfigurationMsg.View] = reconfigurationMsg.QuorumSize
	myId := cfg.subConfig.opts.ID()
	isActive := false
	for _, id := range reconfigurationMsg.ActiveReplicas {
		if id == myId {
			isActive = true
		}
	}
	subConfig, err := cfg.createSubConfiguration(reconfigurationMsg.ActiveReplicas)
	if err != nil {
		// Unable to create the configuration, so no change in the failure case.
		cfg.logger.Info("Unable to create configuration on the reconfiguration req", err)
		return
	}
	cfg.subConfig = *subConfig
	if isActive && !cfg.isActiveReplica {
		cfg.synchronizer.Resume(reconfigurationMsg.QuorumCertificate)
		cfg.isActiveReplica = true
	} else if !isActive && cfg.isActiveReplica {
		cfg.synchronizer.Pause(reconfigurationMsg.QuorumCertificate)
		cfg.isActiveReplica = false
	} else {
		cfg.synchronizer.AdvanceView(hotstuff.NewSyncInfo().WithQC(reconfigurationMsg.QuorumCertificate),
			true)
		cfg.isActiveReplica = true
	}
}

func (cfg *Config) GetLatency(sender hotstuff.ID, receiver hotstuff.ID) time.Duration {
	return latencies[cfg.locationInfo[sender]][cfg.locationInfo[receiver]]
}

func (cfg *subConfig) GetLatency(sender hotstuff.ID, receiver hotstuff.ID) time.Duration {
	return latencies[cfg.locationInfo[sender]][cfg.locationInfo[receiver]]
}

// NewConfig creates a new configuration.
func NewConfig(creds credentials.TransportCredentials, locationInfo map[hotstuff.ID]string, opts ...gorums.ManagerOption) *Config {
	if creds == nil {
		creds = insecure.NewCredentials()
	}
	grpcOpts := []grpc.DialOption{
		grpc.WithBlock(),
		grpc.WithReturnConnectionError(),
		grpc.WithTransportCredentials(creds),
	}
	opts = append(opts, gorums.WithGrpcDialOptions(grpcOpts...))

	// initialization will be finished by InitModule
	cfg := &Config{
		subConfig: subConfig{
			replicas:     make(map[hotstuff.ID]modules.Replica),
			quorumMap:    make(map[hotstuff.View]int),
			locationInfo: locationInfo,
		},
		opts:            opts,
		isActiveReplica: true,
	}
	return cfg
}

func (cfg *Config) replicaConnected(c replicaConnected) {
	info, peerok := peer.FromContext(c.ctx)
	md, mdok := metadata.FromIncomingContext(c.ctx)
	if !peerok || !mdok {
		return
	}

	id, err := GetPeerIDFromContext(c.ctx, cfg)
	if err != nil {
		cfg.logger.Warnf("Failed to get id for %v: %v", info.Addr, err)
		return
	}

	replica, ok := cfg.replicas[id]
	if !ok {
		cfg.logger.Warnf("Replica with id %d was not found", id)
		return
	}

	replica.(*Replica).md = readMetadata(md)

	cfg.logger.Debugf("Replica %d connected from address %v", id, info.Addr)
}

const keyPrefix = "hotstuff-"

func mapToMetadata(m map[string]string) metadata.MD {
	md := metadata.New(nil)
	for k, v := range m {
		md.Set(keyPrefix+k, v)
	}
	return md
}

func readMetadata(md metadata.MD) map[string]string {
	m := make(map[string]string)
	for k, values := range md {
		if _, key, ok := strings.Cut(k, keyPrefix); ok {
			m[key] = values[0]
		}
	}
	return m
}

// GetRawConfiguration returns the underlying gorums RawConfiguration.
func (cfg *Config) GetRawConfiguration() gorums.RawConfiguration {
	return cfg.cfg.RawConfiguration
}

// ReplicaInfo holds information about a replica.
type ReplicaInfo struct {
	ID      hotstuff.ID
	Address string
	PubKey  hotstuff.PublicKey
}

// Connect opens connections to the replicas in the configuration.
func (cfg *Config) Connect(replicas []ReplicaInfo) (err error) {
	opts := cfg.opts
	cfg.opts = nil // options are not needed beyond this point, so we delete them.

	md := mapToMetadata(cfg.subConfig.opts.ConnectionMetadata())

	// embed own ID to allow other replicas to identify messages from this replica
	md.Set("id", fmt.Sprintf("%d", cfg.subConfig.opts.ID()))

	opts = append(opts, gorums.WithMetadata(md))

	cfg.mgr = hotstuffpb.NewManager(opts...)

	// set up an ID mapping to give to gorums
	idMapping := make(map[string]uint32, len(replicas))
	for _, replica := range replicas {
		// also initialize Replica structures
		cfg.replicas[replica.ID] = &Replica{
			eventLoop:    cfg.eventLoop,
			id:           replica.ID,
			pubKey:       replica.PubKey,
			md:           make(map[string]string),
			active:       true,
			locationInfo: cfg.locationInfo,
		}
		// we do not want to connect to ourself
		if replica.ID != cfg.subConfig.opts.ID() {
			idMapping[replica.Address] = uint32(replica.ID)
		}
	}

	// this will connect to the replicas
	cfg.cfg, err = cfg.mgr.NewConfiguration(qspec{}, gorums.WithNodeMap(idMapping))
	if err != nil {
		return fmt.Errorf("failed to create configuration: %w", err)
	}

	// now we need to update the "node" field of each replica we connected to
	for _, node := range cfg.cfg.Nodes() {
		// the node ID should correspond with the replica ID
		// because we already configured an ID mapping for gorums to use.
		id := hotstuff.ID(node.ID())
		replica := cfg.replicas[id].(*Replica)
		replica.node = node
	}

	cfg.connected = true

	// this event is sent so that any delayed `replicaConnected` events can be processed.
	cfg.eventLoop.AddEvent(ConnectedEvent{})

	return nil
}

// Replicas returns all of the replicas in the configuration.
func (cfg *subConfig) Replicas() map[hotstuff.ID]modules.Replica {
	return cfg.replicas
}

// Replica returns a replica if it is present in the configuration.
func (cfg *subConfig) Replica(id hotstuff.ID) (replica modules.Replica, ok bool) {
	replica, ok = cfg.replicas[id]
	return
}

// SubConfig returns a subconfiguration containing the replicas specified in the ids slice.
func (cfg *Config) SubConfig(ids []hotstuff.ID) (sub modules.Configuration, err error) {
	replicas := make(map[hotstuff.ID]modules.Replica)
	nids := make([]uint32, len(ids))
	for i, id := range ids {
		nids[i] = uint32(id)
		replicas[id] = cfg.replicas[id]
	}
	newCfg, err := cfg.mgr.NewConfiguration(gorums.WithNodeIDs(nids))
	if err != nil {
		return nil, err
	}
	return &subConfig{
		eventLoop:    cfg.eventLoop,
		logger:       cfg.logger,
		opts:         cfg.subConfig.opts,
		cfg:          newCfg,
		replicas:     replicas,
		locationInfo: cfg.locationInfo,
	}, nil
}

// Replicas returns all of the replicas in the configuration.
func (cfg *subConfig) ActiveReplicas() map[hotstuff.ID]modules.Replica {
	activeReplicas := make(map[hotstuff.ID]modules.Replica)
	for id, replica := range cfg.replicas {
		if replica.Active() {
			activeReplicas[id] = replica
		}
	}
	return activeReplicas
}

func (cfg *subConfig) Reconfiguration(reconfigurationMsg hotstuff.ReconfigurationMsg) {
	ctx, cancel := synchronizer.TimeoutContext(cfg.eventLoop.Context(), cfg.eventLoop)
	defer cancel()
	protoMsg := hotstuffpb.ReconfigurationToProto(reconfigurationMsg)
	if cfg.passiveConfiguration != nil {
		cfg.passiveConfiguration.ReconfigurationRequest(ctx,
			protoMsg)
	}
	cfg.cfg.ReconfigurationRequest(ctx, protoMsg)
}

func (cfg *subConfig) SubConfig(_ []hotstuff.ID) (_ modules.Configuration, err error) {
	return nil, errors.New("not supported")
}

// Len returns the number of replicas in the configuration.
func (cfg *subConfig) Len() int {
	return len(cfg.replicas)
}

// QuorumSize returns the size of a quorum
func (cfg *subConfig) QuorumSize(view hotstuff.View) int {
	qs, ok := cfg.quorumMap[view]
	if !ok {
		qs = hotstuff.QuorumSize(len(cfg.ActiveReplicas()))
	}
	return qs
}

// Propose sends the block to all replicas in the configuration
func (cfg *subConfig) Propose(proposal hotstuff.ProposeMsg) {
	if cfg.cfg == nil {
		return
	}
	ctx, cancel := synchronizer.TimeoutContext(cfg.eventLoop.Context(), cfg.eventLoop)
	defer cancel()
	protoProposal := hotstuffpb.ProposalToProto(proposal)
	cfg.cfg.Propose(
		ctx,
		protoProposal,
	)
	cfg.eventLoop.AddEvent(hotstuff.BlockBytesEvent{NumberBytes: int(unsafe.Sizeof(protoProposal) / 1024)})
}

func (cfg *subConfig) Update(block hotstuff.Block) {
	if cfg.passiveConfiguration == nil {
		return
	}
	ctx, cancel := synchronizer.TimeoutContext(cfg.eventLoop.Context(), cfg.eventLoop)
	defer cancel()
	cfg.passiveConfiguration.Update(ctx,
		hotstuffpb.UpdateToProto(hotstuff.Update{Block: &block,
			QuorumSize: hotstuff.QuorumSize(len((cfg.ActiveReplicas())))}),
	)
}

// Timeout sends the timeout message to all replicas.
func (cfg *subConfig) Timeout(msg hotstuff.TimeoutMsg) {
	if cfg.cfg == nil {
		return
	}

	// will wait until the second timeout before cancelling
	ctx, cancel := synchronizer.TimeoutContext(cfg.eventLoop.Context(), cfg.eventLoop)
	defer cancel()

	cfg.cfg.Timeout(
		ctx,
		hotstuffpb.TimeoutMsgToProto(msg),
	)
}

// Fetch requests a block from all the replicas in the configuration
func (cfg *subConfig) Fetch(ctx context.Context, hash hotstuff.Hash) (*hotstuff.Block, bool) {
	var allConfig *hotstuffpb.Configuration
	var err error
	if cfg.passiveConfiguration != nil {
		allConfig, err = cfg.mgr.NewConfiguration(qspec{}, cfg.cfg.And(cfg.passiveConfiguration))
		if err != nil {
			cfg.logger.Info("Error in fetch ", err)
			return nil, false
		}
	} else {
		allConfig = cfg.cfg
	}
	protoBlock, err := allConfig.Fetch(ctx, &hotstuffpb.BlockHash{Hash: hash[:]})
	if err != nil {
		qcErr, ok := err.(gorums.QuorumCallError)
		// filter out context errors
		if !ok || (qcErr.Reason != context.Canceled.Error() && qcErr.Reason != context.DeadlineExceeded.Error()) {
			cfg.logger.Infof("Failed to fetch block: %v", err)
		}
		return nil, false
	}
	return hotstuffpb.BlockFromProto(protoBlock), true
}

// Close closes all connections made by this configuration.
func (cfg *Config) Close() {
	cfg.mgr.Close()
}

func (sub *subConfig) GetCommittees(size int, isPhase1 bool) map[int][]hotstuff.ID {
	committees := make(map[int][]hotstuff.ID)
	isIdTaken := make(map[hotstuff.ID]bool)
	for id := range sub.locationInfo {
		isIdTaken[id] = false
	}
	sub.logger.Info("location info", sub.locationInfo)
	numofcommittees := len(sub.locationInfo) / size
	formed := 0
	if isPhase1 {
		for formed < numofcommittees {
			tempCommittee := make([]hotstuff.ID, 0)
			var location string
			for id, tempLocation := range sub.locationInfo {
				if !isIdTaken[id] {
					location = tempLocation
					break
				}
			}
			for len(tempCommittee) < size {
				for id, tempLocation := range sub.locationInfo {
					if !isIdTaken[id] && tempLocation == location {
						tempCommittee = append(tempCommittee, id)
						isIdTaken[id] = true
					}
					if len(tempCommittee) == size {
						break
					}
				}
				location = findNearestLocation(location, sub.locationInfo)
			}
			committees[formed+1] = tempCommittee
			formed += 1
		}
	} else {
		rnd := rand.New(rand.NewSource(1))
		for formed < numofcommittees {
			tempCommittee := make([]hotstuff.ID, 0)
			for len(tempCommittee) < size {
				value := hotstuff.ID((rnd.Int() % len(sub.locationInfo)) + 1)
				if !isIdTaken[value] {
					isIdTaken[value] = true
					tempCommittee = append(tempCommittee, value)
				}
			}
			committees[formed+1] = tempCommittee
			formed += 1
		}
		committees = sortCommittees(committees, sub.locationInfo)
	}
	return committees
}

func sortCommittees(committees map[int][]hotstuff.ID, locationInfo map[hotstuff.ID]string) map[int][]hotstuff.ID {
	retCommittees := make(map[int][]hotstuff.ID)
	committeeLatencies := make(map[int]int64)
	for index, committee := range committees {
		committeeLatencies[index] = committeeLatency(committee, locationInfo)
	}
	committeeIDs := make([]int, 0)
	for id := range committees {
		committeeIDs = append(committeeIDs, id)
	}
	sort.SliceStable(committeeIDs, func(i, j int) bool {
		return committeeLatencies[committeeIDs[i]] < committeeLatencies[committeeIDs[j]]
	})
	for index, id := range committeeIDs {
		retCommittees[index] = committees[id]
	}
	return retCommittees
}

func committeeLatency(committee []hotstuff.ID, locationInfo map[hotstuff.ID]string) int64 {
	var maxLatency int64
	var tempLatency int64
	for _, id := range committee {
		for _, id1 := range committee {
			tempLatency += int64(latencies[locationInfo[id]][locationInfo[id1]])
		}
	}
	if tempLatency > maxLatency {
		maxLatency = tempLatency
	}
	return maxLatency
}

func GetCommittees(size int, isPhase1 bool, locationInfo map[hotstuff.ID]string) map[int][]hotstuff.ID {
	committees := make(map[int][]hotstuff.ID)
	isIdTaken := make(map[hotstuff.ID]bool)
	for id := range locationInfo {
		isIdTaken[id] = false
	}
	numofcommittees := len(locationInfo) / size
	formed := 0
	if isPhase1 {
		for formed < numofcommittees {
			tempCommittee := make([]hotstuff.ID, 0)
			var location string
			for id, tempLocation := range locationInfo {
				if !isIdTaken[id] {
					location = tempLocation
					break
				}
			}
			for len(tempCommittee) < size {
				found := false
				for id, tempLocation := range locationInfo {
					if !isIdTaken[id] && tempLocation == location {
						tempCommittee = append(tempCommittee, id)
						isIdTaken[id] = true
						found = true
					}
					if len(tempCommittee) == size {
						break
					}
				}
				if !found {
					for id, tempLocation := range locationInfo {
						if !isIdTaken[id] {
							location = tempLocation
							break
						}
					}
				} else {
					location = findNearestLocation(location, locationInfo)
				}
			}
			committees[formed] = tempCommittee
			formed += 1
		}
	} else {
		rnd := rand.New(rand.NewSource(1))
		for formed < numofcommittees {
			tempCommittee := make([]hotstuff.ID, 0)
			for len(tempCommittee) < size {
				value := hotstuff.ID((rnd.Int() % len(locationInfo)) + 1)
				if !isIdTaken[value] {
					isIdTaken[value] = true
					tempCommittee = append(tempCommittee, value)
				}
			}
			committees[formed] = tempCommittee
			formed += 1
		}
		//committees = sortCommittees(committees, locationInfo)
	}

	return committees
}

func (sub *subConfig) FindNearestReplica(id hotstuff.ID, replicas []hotstuff.ID) hotstuff.ID {
	var nearestReplica hotstuff.ID
	minLatency := int64(1 << 62)
	for _, replica := range replicas {
		if replica != id {
			latency := sub.GetLatency(id, replica)
			if latency < time.Duration(minLatency) {
				minLatency = int64(latency)
				nearestReplica = replica
			}
		}
	}
	return nearestReplica
}

func findNearestLocation(location string, locationInfo map[hotstuff.ID]string) string {
	latencyVector := latencies[location]
	var nearLocation string
	maxDuration := time.Duration(1 * time.Second)
	for tempLocation, duration := range latencyVector {
		if duration < maxDuration && tempLocation != location && isPresent(tempLocation, locationInfo) {
			nearLocation = tempLocation
			maxDuration = duration
		}
	}
	return nearLocation
}

func isPresent(location string, locationInfo map[hotstuff.ID]string) bool {
	for _, loc := range locationInfo {
		if location == loc {
			return true
		}
	}
	return false
}

var _ modules.Configuration = (*Config)(nil)

type qspec struct{}

// FetchQF is the quorum function for the Fetch quorum call method.
// It simply returns true if one of the replies matches the requested block.
func (q qspec) FetchQF(in *hotstuffpb.BlockHash, replies map[uint32]*hotstuffpb.Block) (*hotstuffpb.Block, bool) {
	var h hotstuff.Hash
	copy(h[:], in.GetHash())
	for _, b := range replies {
		block := hotstuffpb.BlockFromProto(b)
		if h == block.Hash() {
			return b, true
		}
	}
	return nil, false
}

// ConnectedEvent is sent when the configuration has connected to the other replicas.
type ConnectedEvent struct{}
