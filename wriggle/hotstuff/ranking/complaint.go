package ranking

import (
	"container/list"
	"sort"

	"github.com/relab/hotstuff"
	"github.com/relab/hotstuff/backend"
	"github.com/relab/hotstuff/eventloop"
	"github.com/relab/hotstuff/logging"
	"github.com/relab/hotstuff/modules"
)

const SUSPICIONFACTOR = 2

func init() {
	modules.RegisterModule("complaintcache", New)
}

func New() modules.Ranking {
	return &ComplaintCache{
		score:                 make(map[hotstuff.ID]int),
		suspicionMatrix:       make(map[hotstuff.ID]map[hotstuff.ID]int),
		complaintCache:        list.New(),
		alreadyVoted:          make(map[hotstuff.ID]map[hotstuff.ID]uint64),
		serialNumForComplaint: make(map[hotstuff.ID]uint64),
		faultyNodes:           make([]hotstuff.ID, 0),
		leaderScore:           make(map[hotstuff.ID]float64),
	}
}

type ComplaintCache struct {
	consensus             modules.Consensus
	crypto                modules.Crypto
	eventLoop             *eventloop.EventLoop
	configuration         *backend.Config
	opts                  *modules.Options
	logger                logging.Logger
	score                 map[hotstuff.ID]int
	suspicionMatrix       map[hotstuff.ID]map[hotstuff.ID]int
	alreadyVoted          map[hotstuff.ID]map[hotstuff.ID]uint64
	serialNumForComplaint map[hotstuff.ID]uint64
	leaderScore           map[hotstuff.ID]float64
	faultyNodes           []hotstuff.ID
	complaintCache        *list.List
	id                    hotstuff.ID
	configLength          int
}

// TODO initialize the score matrix
func (cc *ComplaintCache) InitModule(mods *modules.Core) {
	mods.Get(
		&cc.consensus,
		&cc.crypto,
		&cc.configuration,
		&cc.eventLoop,
		&cc.logger,
		&cc.opts,
	)
	cc.eventLoop.RegisterObserver(backend.ConnectedEvent{}, func(_ any) {
		cc.postInit()
	})
	cc.id = cc.opts.ID()

}

func (cc *ComplaintCache) postInit() {
	replicas := cc.configuration.ActiveReplicas()
	cc.configLength = len(replicas)
	for id := range replicas {
		cc.score[id] = 100
		cc.suspicionMatrix[id] = make(map[hotstuff.ID]int)
		cc.alreadyVoted[id] = make(map[hotstuff.ID]uint64)
		cc.serialNumForComplaint[id] = 0
	}
}

func (cc *ComplaintCache) AddComplaint(complaint *hotstuff.Complaint) {
	//Since the complaint is raised by the same node,
	//we may not have to verify the complaint.
	if value, ok := cc.serialNumForComplaint[complaint.Complainant]; ok {
		complaint.ID = value + 1
		cc.serialNumForComplaint[complaint.Complainant] = value + 1
	} else {
		cc.serialNumForComplaint[complaint.Complainant] = 1
		complaint.ID = 1
	}
	cc.complaintCache.PushBack(complaint)
	//cc.logger.Info("Added complaint ", complaint.ComplaintType, complaint.Complainee)
}

func (cc *ComplaintCache) GetPendingComplaints() []*hotstuff.Complaint {

	pendingComplaintsLen := cc.complaintCache.Len()
	pendingComplaints := make([]*hotstuff.Complaint, 0, pendingComplaintsLen)
	for ele := cc.complaintCache.Front(); ele != nil; ele = ele.Next() {
		pendingComplaints = append(pendingComplaints, ele.Value.(*hotstuff.Complaint))
	}
	return pendingComplaints
}

func (cc *ComplaintCache) CommitComplaints(complaints []*hotstuff.Complaint) {
	if len(complaints) > 0 {
		if complaints[0].Complainee == cc.id {
			var next *list.Element
			// remove the complaints about to be committed from pending complaints.
			for _, complaint := range complaints {
				for e := cc.complaintCache.Front(); e != nil; e = next {
					next = e.Next()
					comp := e.Value.(*hotstuff.Complaint)
					if complaint.ID == comp.ID {
						cc.complaintCache.Remove(e)
						break
					}
				}
			}
		}
	}
	for _, complaint := range complaints {
		if value, ok := cc.alreadyVoted[complaint.Complainee]; ok {
			voted, isPreset := value[complaint.Complainant]
			if isPreset {
				if voted >= complaint.ID {
					continue
				}
			}
		} else {
			cc.alreadyVoted[complaint.Complainee] = make(map[hotstuff.ID]uint64)
		}
		cc.alreadyVoted[complaint.Complainee][complaint.Complainant] = complaint.ID
		if complaint.ComplaintType == hotstuff.Suspicion {
			if _, ok := cc.suspicionMatrix[complaint.Complainee]; !ok {
				cc.suspicionMatrix[complaint.Complainee] = make(map[hotstuff.ID]int)
			}
			cc.suspicionMatrix[complaint.Complainee][complaint.Complainant] += 1
		} else {
			penalty := hotstuff.Penalities[complaint.ComplaintType]
			cc.score[complaint.Complainant] -= penalty
			cc.faultyNodes = append(cc.faultyNodes, complaint.Complainant)
		}
	}

	//cc.logger.Info("Score after commit is", cc.suspicionMatrix)
}

func (cc *ComplaintCache) VerifyComplaints(complaints []*hotstuff.Complaint) bool {
	for _, complaint := range complaints {
		if !cc.VerifyComplaint(complaint) {
			return false
		}
	}
	return true
}

func (cc *ComplaintCache) VerifyComplaint(complaint *hotstuff.Complaint) bool {

	switch complaint.ComplaintType {
	case hotstuff.InvalidProposal:
		// proof, ok := complaint.Proof.(hotstuff.ProposeMsg)
		// if !ok {
		// 	return false
		// }
		// return !cc.consensus.VerifyProposal(proof)
		return true
	case hotstuff.InvalidQuorumCert:
		proof, ok := complaint.Proof.(hotstuff.QuorumCert)
		if !ok {
			return false
		}
		return !cc.crypto.VerifyQuorumCert(proof)
	case hotstuff.InvalidVote:
		proof, ok := complaint.Proof.(hotstuff.PartialCert)
		if !ok {
			return false
		}
		return !cc.crypto.VerifyPartialCert(proof)
	case hotstuff.InvalidComplaint:
		proof, ok := complaint.Proof.(hotstuff.Complaint)
		if !ok {
			return false
		}
		return cc.VerifyComplaint(&proof)
	case hotstuff.Suspicion:
		return true
	default:
		return false
	}
}

func (cc *ComplaintCache) GetTopN(n int) ([]hotstuff.ID, bool) {

	IDS := make([]hotstuff.ID, 0)
	if n > len(cc.score) || n <= 0 {
		return IDS, false
	}
	for id := range cc.score {
		IDS = append(IDS, id)
	}
	sort.SliceStable(IDS, func(i, j int) bool {
		return cc.score[IDS[i]] < cc.score[IDS[j]]
	})
	return IDS[:n], true
}
func (cc *ComplaintCache) GetRobustInternalNodes(nodeCount int) []hotstuff.ID {
	ids := make(map[hotstuff.ID]bool, 0)
	suspicionScore := make(map[hotstuff.ID]int)
	for _, suspicionMap := range cc.suspicionMatrix {
		for id1, score := range suspicionMap {
			suspicionScore[id1] += score
			if suspicionScore[id1]/cc.configLength >= SUSPICIONFACTOR {
				ids[id1] = false
			} else {
				ids[id1] = true
			}
		}
	}
	trustedNodes := make([]hotstuff.ID, 0)
	count := 0
	for id, value := range ids {
		if value {
			trustedNodes = append(trustedNodes, id)
			count++
		}
		if count == nodeCount {
			break
		}
	}
	return trustedNodes
}

func (cc *ComplaintCache) GetScore() map[hotstuff.ID]int {
	return cc.score
}

func (cc *ComplaintCache) GetSuspicionMatrix() map[hotstuff.ID]map[hotstuff.ID]int {
	return cc.suspicionMatrix
}

func (cc *ComplaintCache) GetLeaderScore(id hotstuff.ID) float64 {
	return cc.leaderScore[id]
}

func (cc *ComplaintCache) GetSuspectedNodes() map[hotstuff.ID]int {
	cc.logger.Info("suspicion nodes are", cc.suspicionMatrix)
	suspectedNodes := make(map[hotstuff.ID]int)
	for _, suspicions := range cc.suspicionMatrix {
		for suspectedNode, count := range suspicions {
			prevCount, ok := suspectedNodes[suspectedNode]
			if !ok {
				suspectedNodes[suspectedNode] = count
			} else {
				suspectedNodes[suspectedNode] = prevCount + count
			}
		}
	}
	return suspectedNodes
}

func (cc *ComplaintCache) GetFaultyNodes() []hotstuff.ID {
	return cc.faultyNodes
}
