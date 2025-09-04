// Package kauri contains the implementation of the kauri protocol
package kauri

import (
	"context"
	"encoding/binary"
	"errors"
	"math/rand"
	"reflect"
	"sort"
	"time"

	"github.com/relab/gorums"
	"github.com/relab/hotstuff"
	"github.com/relab/hotstuff/backend"
	"github.com/relab/hotstuff/eventloop"
	"github.com/relab/hotstuff/internal/proto/hotstuffpb"
	"github.com/relab/hotstuff/internal/proto/kauripb"
	"github.com/relab/hotstuff/logging"
	"github.com/relab/hotstuff/modules"
)

func init() {
	modules.RegisterModule("kauri", New)
}

// TreeType determines the type of tree to be built, for future extension
type TreeType int

// NoFaultTree is a tree without any faulty replicas in the tree, which is currently supported.
// TreeWithFaults is a tree with faulty replicas, not yet implemented.
const (
	NoFaultTree    TreeType = 1
	TreeWithFaults TreeType = 2
)

// Kauri structure contains the modules for kauri protocol implementation.
type Kauri struct {
	configuration          *backend.Config
	server                 *backend.Server
	blockChain             modules.BlockChain
	crypto                 modules.Crypto
	eventLoop              *eventloop.EventLoop
	logger                 logging.Logger
	opts                   *modules.Options
	synchronizer           modules.Synchronizer
	leaderRotation         modules.LeaderRotation
	tree                   TreeConfiguration
	initDone               bool
	aggregatedContribution hotstuff.QuorumSignature
	blockHash              hotstuff.Hash
	currentView            hotstuff.View
	senders                []hotstuff.ID
	nodes                  map[hotstuff.ID]*kauripb.Node
	isAggregationSent      bool
	partitionNumber        int
	ranking                modules.Ranking
	partitions             map[int][]hotstuff.ID
	faultNumber            int
	changeTree             bool
	tickerId               int
	isOptiLog              bool
}

// New initializes the kauri structure
func New() modules.Kauri {
	return &Kauri{nodes: make(map[hotstuff.ID]*kauripb.Node), partitions: make(map[int][]hotstuff.ID), faultNumber: -1}
}

// InitModule initializes the Handel module.
func (k *Kauri) InitModule(mods *modules.Core) {
	mods.Get(
		&k.configuration,
		&k.server,
		&k.blockChain,
		&k.crypto,
		&k.eventLoop,
		&k.logger,
		&k.opts,
		&k.leaderRotation,
		&k.synchronizer,
	)

	mods.TryGet(&k.ranking)
	k.opts.SetShouldUseKauri()
	k.eventLoop.RegisterObserver(backend.ConnectedEvent{}, func(_ any) {
		k.postInit()
	})
	k.eventLoop.RegisterHandler(ContributionRecvEvent{}, func(event any) {
		k.OnContributionRecv(event.(ContributionRecvEvent))
	})
	k.isOptiLog = true //toggle this for kauri
	// Uncomment this block to enable the tree change event
	// k.tickerId = k.eventLoop.AddTicker(time.Duration(10*time.Second), k.tick)
	// k.eventLoop.RegisterHandler(ChangeTreeEvent{}, func(event any) {
	// 	k.setChangeTree(event.(ChangeTreeEvent))
	// })
}

func (k *Kauri) setChangeTree(_ ChangeTreeEvent) {
	k.changeTree = true
}

func (k *Kauri) tick(tickTime time.Time) any {
	if k.faultNumber < len(hotstuff.FaultyNodes) {
		k.faultNumber++
	} else {
		k.eventLoop.RemoveTicker(k.tickerId)
	}
	return ChangeTreeEvent{}
}

func (k *Kauri) postInit() {
	k.logger.Info("Handel: Initializing")
	kauripb.RegisterKauriServer(k.server.GetGorumsServer(), serviceImpl{k})
	k.initializeConfiguration()
}

func (k *Kauri) initializeConfiguration() {
	kauriCfg := kauripb.ConfigurationFromRaw(k.configuration.GetRawConfiguration(), nil)
	for _, n := range kauriCfg.Nodes() {
		k.nodes[hotstuff.ID(n.ID())] = n
	}
	// count :=1 // Number of fault nodes
	// suspicions := k.ranking.GetSuspectedNodes()
	// k.logger.Info("suspicions are ", suspicions)
	// ids := make([]hotstuff.ID, 0, len(suspicions))
	// for id := range suspicions {
	// 	ids = append(ids, id)
	// }
	// sort.SliceStable(ids, func(i, j int) bool {
	// 	return suspicions[ids[i]] > suspicions[ids[j]]
	// })

	// removeNodes := make([]hotstuff.ID, 0)
	// for index, id := range ids {
	// 	if index <= count {
	// 		removeNodes = append(removeNodes, id)
	// 	} else {
	// 		break
	// 	}
	// }

	k.tree = CreateTree(k.configuration.Len(), k.opts.ID())

	// pIDs := make(map[hotstuff.ID]int)
	// for id := range k.configuration.ActiveReplicas() {
	// 	pIDs[id] = index
	// 	index++
	// }
	idMappings := make(map[hotstuff.ID]int)
	for i := 0; i < k.configuration.Len(); i++ {
		idMappings[hotstuff.ID(i+1)] = i
	}
	k.tree.InitializeWithPIDs(idMappings)
	k.initDone = true
	k.senders = make([]hotstuff.ID, 0)
	k.partitions = k.makePartitions()
	k.partitionNumber = 1
	k.logger.Info("partitions are ", k.partitions)
}

func (k *Kauri) makePartitions() map[int][]hotstuff.ID {
	internalNodesNumber := MaxChild + 1
	if k.isOptiLog {
		return k.configuration.GetCommittees(internalNodesNumber, true)
	}
	partitionsNumber := k.configuration.Len() / internalNodesNumber
	partitions := make(map[int][]hotstuff.ID)
	totalNodes := k.configuration.Len()
	ids := make([]hotstuff.ID, 0, totalNodes)
	for id := range k.configuration.Replicas() {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	seed := k.opts.SharedRandomSeed()
	//Shuffle the list of IDs using the shared random seed + the first 8 bytes of the hash.
	rnd := rand.New(rand.NewSource(seed))
	rnd.Shuffle(len(ids), reflect.Swapper(ids))
	for i := 0; i < partitionsNumber; i++ {
		tempHostSet := make([]hotstuff.ID, 0, internalNodesNumber)
		for j := 0; j < internalNodesNumber; j++ {
			tempHostSet = append(tempHostSet, ids[i*internalNodesNumber+j])
		}
		partitions[i+1] = tempHostSet
	}
	return partitions
}

// Begin starts dissemination of proposal and aggregation of votes.
func (k *Kauri) Begin(pc hotstuff.PartialCert, p hotstuff.ProposeMsg) {
	if !k.initDone {
		k.eventLoop.DelayUntil(backend.ConnectedEvent{}, func() { k.Begin(pc, p) })
		return
	}
	k.reset()
	k.blockHash = pc.BlockHash()
	k.currentView = p.Block.View()
	k.aggregatedContribution = pc.Signature()
	if k.currentView == 0 {
		ids := k.randomizeIDS(k.blockHash, k.leaderRotation.GetLeader(k.currentView))
		k.tree.InitializeWithPIDs(ids)
	}
	if k.currentView == 1 || k.changeTree {
		ids := k.randomizeIDS(k.blockHash, k.leaderRotation.GetLeader(k.currentView))
		if k.isOptiLog {
			ids = k.assignLeafNodes(k.partitions[k.partitionNumber], k.leaderRotation.GetLeader(k.currentView))
			//ids = k.moveFaultsToLeaf(ids)
		} else {
			ids = k.makeArrayWithPartitions(k.partitions[k.partitionNumber],
				k.leaderRotation.GetLeader(k.currentView))
		}
		k.partitionNumber++
		k.tree.InitializeWithPIDs(ids)
		k.changeTree = false
		k.logger.Info("******************Tree changed***********")
	}
	isFaulty := false
	for _, id := range hotstuff.FaultyNodes {
		if id == k.opts.ID() {
			isFaulty = true
		}
	}
	if isFaulty && k.opts.ID() <= hotstuff.ID(k.faultNumber) {
		return
		// ticker := time.NewTicker(time.Duration(100 * time.Millisecond))
		// <-ticker.C
		// ticker.Stop()
	}
	k.SendProposalToChildren(p)
	waitTime := time.Duration(uint64(k.tree.GetHeight() * 30 * int(time.Millisecond)))
	go k.aggregateAndSend(waitTime, k.currentView)
}

func (k *Kauri) reset() {
	k.aggregatedContribution = nil
	k.senders = make([]hotstuff.ID, 0)
	k.isAggregationSent = false
}

func (k *Kauri) aggregateAndSend(t time.Duration, view hotstuff.View) {
	ticker := time.NewTicker(t)
	<-ticker.C
	ticker.Stop()
	if k.currentView != view {
		return
	}
	if !k.isAggregationSent {
		k.SendContributionToParent()
	}
}

// SendProposalToChildren sends the proposal to the children
func (k *Kauri) SendProposalToChildren(p hotstuff.ProposeMsg) {
	children := k.tree.GetChildren()
	if len(children) != 0 {
		config, err := k.configuration.SubConfig(k.tree.GetChildren())
		if err != nil {
			k.logger.Error("Unable to send the proposal to children", err)
			return
		}
		k.logger.Debug("sending proposal to children ", k.tree.GetChildren())
		p.Block.SetTime(time.Now())
		config.Propose(p)
	} else {
		k.SendContributionToParent()
		k.isAggregationSent = true
	}
}

// OnContributionRecv is invoked upon receiving the vote for aggregation.
func (k *Kauri) OnContributionRecv(event ContributionRecvEvent) {
	if k.currentView != hotstuff.View(event.Contribution.View) {
		return
	}
	contribution := event.Contribution
	k.logger.Debug("processing the contribution from ", contribution.ID)
	currentSignature := hotstuffpb.QuorumSignatureFromProto(contribution.Signature)
	_, err := k.mergeWithContribution(currentSignature)
	if err != nil {
		k.logger.Debug("Unable to merge the contribution from ", contribution.ID)
		return
	}
	k.senders = append(k.senders, hotstuff.ID(contribution.ID))

	if _, ok := isSubSet(k.tree.GetSubTreeNodes(), k.senders); ok {
		k.SendContributionToParent()
		k.isAggregationSent = true
	}
}

// SendContributionToParent sends contribution to the parent node.
func (k *Kauri) SendContributionToParent() {
	isFaulty := false
	for _, id := range hotstuff.FaultyNodes {
		if id == k.opts.ID() {
			isFaulty = true
		}
	}
	if isFaulty {
		ticker := time.NewTicker(time.Duration(100 * time.Millisecond))
		<-ticker.C
		ticker.Stop()
	}
	parent, ok := k.tree.GetParent()
	children := k.tree.GetChildren()
	if len(children) != 0 {
		remaining, ok := isSubSet(children, k.senders)
		if !ok {
			for _, id := range remaining {
				k.ranking.AddComplaint(&hotstuff.Complaint{
					Complainee:       k.opts.ID(),
					Complainant:      id,
					IsProofAvailable: false,
					ComplaintType:    hotstuff.Suspicion,
				})
			}
		}
	}
	if ok {
		node, isPresent := k.nodes[parent]
		if isPresent {
			node.SendContribution(context.Background(), &kauripb.Contribution{
				ID:        uint32(k.opts.ID()),
				Signature: hotstuffpb.QuorumSignatureToProto(k.aggregatedContribution),
				View:      uint64(k.currentView),
			})
		}
	}
}

type serviceImpl struct {
	k *Kauri
}

func (i serviceImpl) SendContribution(ctx gorums.ServerCtx, request *kauripb.Contribution) {
	i.k.eventLoop.AddEvent(ContributionRecvEvent{Contribution: request})
}

// ContributionRecvEvent is raised when a contribution is received.
type ContributionRecvEvent struct {
	Contribution *kauripb.Contribution
}

func (k *Kauri) canMergeContributions(a, b hotstuff.QuorumSignature) bool {
	canMerge := true
	if a == nil || b == nil {
		k.logger.Info("one of it is nil")
		return false
	}
	a.Participants().RangeWhile(func(i hotstuff.ID) bool {
		b.Participants().RangeWhile(func(j hotstuff.ID) bool {
			// cannot merge a and b if they both contain a contribution from the same ID.
			if i == j {
				canMerge = false
			}
			return canMerge
		})
		return canMerge
	})
	return canMerge
}

func (k *Kauri) verifyContribution(signature hotstuff.QuorumSignature, hash hotstuff.Hash) bool {
	verified := false
	block, ok := k.blockChain.Get(hash)
	if !ok {
		k.logger.Info("failed to fetch the block ", hash)
		return verified
	}
	verified = k.crypto.Verify(signature, block.ToBytes())
	return verified
}

func (k *Kauri) mergeWithContribution(currentSignature hotstuff.QuorumSignature) (bool, error) {
	isVerified := k.verifyContribution(currentSignature, k.blockHash)
	if !isVerified {
		k.logger.Info("Contribution verification failed for view ", k.currentView,
			"from participants", currentSignature.Participants(), " block hash ", k.blockHash)
		return false, errors.New("unable to verify the contribution")
	}
	if k.aggregatedContribution == nil {
		k.aggregatedContribution = currentSignature
		return false, nil
	}

	if k.canMergeContributions(currentSignature, k.aggregatedContribution) {
		new, err := k.crypto.Combine(currentSignature, k.aggregatedContribution)
		if err == nil {
			k.aggregatedContribution = new
			if new.Participants().Len() >= k.configuration.QuorumSize(k.currentView) {
				k.logger.Debug("Aggregated Complete QC and sending the event")
				k.eventLoop.AddEvent(hotstuff.NewViewMsg{
					SyncInfo: hotstuff.NewSyncInfo().WithQC(hotstuff.NewQuorumCert(
						k.aggregatedContribution,
						k.currentView,
						k.blockHash, make([]uint32, 0),
					)),
				})
				return true, nil
			}
		} else {
			k.logger.Info("Failed to combine signatures: %v", err)
			return false, errors.New("unable to combine signature")
		}
	} else {
		k.logger.Debug("Failed to merge signatures due to overlap of signatures.")
		return false, errors.New("unable to merge signature")
	}
	return false, nil
}

// sends A-B
func isSubSet(a, b []hotstuff.ID) ([]hotstuff.ID, bool) {
	c := hotstuff.NewIDSet()
	remaining := make([]hotstuff.ID, 0)
	ret := true
	for _, id := range b {
		c.Add(id)
	}
	for _, id := range a {
		if !c.Contains(id) {
			ret = false
			remaining = append(remaining, id)
		}
	}
	return remaining, ret
}

func (k *Kauri) randomizeIDS(hash hotstuff.Hash, leaderID hotstuff.ID) map[hotstuff.ID]int {
	//assign leader to the root of the tree.

	totalNodes := k.configuration.Len()
	ids := make([]hotstuff.ID, 0, totalNodes)
	for id := range k.configuration.Replicas() {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	seed := k.opts.SharedRandomSeed() + int64(binary.LittleEndian.Uint64(hash[:]))
	//Shuffle the list of IDs using the shared random seed + the first 8 bytes of the hash.
	rnd := rand.New(rand.NewSource(seed))
	rnd.Shuffle(len(ids), reflect.Swapper(ids))
	lIndex := 0
	for index, id := range ids {
		if id == leaderID {
			lIndex = index
		}
	}
	currentRoot := ids[0]
	ids[0] = ids[lIndex]
	ids[lIndex] = currentRoot
	posMapping := make(map[hotstuff.ID]int)
	for index, ID := range ids {
		posMapping[ID] = index
	}
	k.logger.Info("original ids", posMapping)
	return posMapping
}

func (k *Kauri) assignLeafNodes(ids []hotstuff.ID, leaderID hotstuff.ID) map[hotstuff.ID]int {
	treePos := make(map[hotstuff.ID]int)
	pos := 0
	for _, id := range ids {
		treePos[id] = pos
		pos++
	}
	leafNodes := k.leafNodes(ids)
	for index, id := range ids {
		if index == 0 {
			continue
		}
		for i := 0; i < MaxChild; i++ {
			leafNode := k.configuration.FindNearestReplica(id, leafNodes)
			treePos[leafNode] = pos
			pos++
			for j, temp := range leafNodes {
				if temp == leafNode {
					leafNodes = append(leafNodes[:j], leafNodes[j+1:]...)
					break
				}
			}
		}
	}
	return correctLeaderPos(leaderID, treePos)
}

func (k *Kauri) moveFaultsToLeaf(posMappings map[hotstuff.ID]int) map[hotstuff.ID]int {
	totalNodesLength := len(posMappings) - 1
	for _, id := range hotstuff.FaultyNodes {
		faultyNode := id
		faultyNodePos := 0
		lastPos := totalNodesLength
		lastID := hotstuff.ID(0)
		for id, pos := range posMappings {
			if faultyNode == id {
				faultyNodePos = pos
			}
			if pos == lastPos {
				lastID = id
			}
		}
		posMappings[faultyNode] = lastPos
		posMappings[lastID] = faultyNodePos
		totalNodesLength--
		k.logger.Info("Fault node identified is ", faultyNode)
	}
	k.logger.Info("posmappings ", posMappings)
	return posMappings
}

func (k *Kauri) leafNodes(ids []hotstuff.ID) []hotstuff.ID {
	otherNodes := make([]hotstuff.ID, 0)
	for id := range k.configuration.Replicas() {
		isFound := false
		for _, temp := range ids {
			if temp == id {
				isFound = true
				break
			}
		}
		if !isFound {
			otherNodes = append(otherNodes, id)
		}
	}
	k.logger.Info("Other nodes", otherNodes)
	sort.Slice(otherNodes, func(i, j int) bool { return otherNodes[i] < otherNodes[j] })
	return otherNodes
}

func (k *Kauri) makeArrayWithPartitions(ids []hotstuff.ID, leaderId hotstuff.ID) map[hotstuff.ID]int {
	ret := make(map[hotstuff.ID]int)
	otherNodes := k.leafNodes(ids)
	for index, id := range ids {
		ret[id] = index
	}
	temp := len(ids)
	for _, id := range otherNodes {
		ret[id] = temp
		temp++
	}
	return correctLeaderPos(leaderId, ret)
}

func correctLeaderPos(leaderID hotstuff.ID, ids map[hotstuff.ID]int) map[hotstuff.ID]int {
	lIndex := ids[leaderID]
	currentRoot := hotstuff.ID(0)
	for id, index := range ids {
		if index == 0 {
			currentRoot = id
			break
		}
	}
	ids[leaderID] = 0
	ids[currentRoot] = lIndex
	return ids
}

type ChangeTreeEvent struct {
}
