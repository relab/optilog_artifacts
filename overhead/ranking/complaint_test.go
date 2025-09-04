package ranking

import (
	"fmt"
	"sort"
	"testing"

	"github.com/relab/hotstuff"
)

func TestAddComplaint(t *testing.T) {
	complaintCache := New()
	complaint := hotstuff.Complaint{
		Complainee:  1,
		Complainant: 2,
	}
	complaint1 := hotstuff.Complaint{
		Complainee:  1,
		Complainant: 2,
	}
	complaint2 := hotstuff.Complaint{
		Complainee:  1,
		Complainant: 2,
	}
	complaintCache.AddComplaint(&complaint)
	complaints := complaintCache.GetPendingComplaints()
	if len(complaints) != 1 {
		t.Errorf("GetPendingComplaints failed to fetch the complaints")
	}
	if complaints[0].ID != 1 {
		t.Errorf("Complaint ID is not set properly")
	}
	complaintCache.AddComplaint(&complaint1)
	complaintCache.AddComplaint(&complaint2)
	complaints = complaintCache.GetPendingComplaints()
	if len(complaints) != 3 {
		t.Errorf("GetPendingComplaints failed to fetch the complaints")
	}
	sort.Slice(complaints, func(i, j int) bool {
		return complaints[i].ID < complaints[j].ID
	})
	if complaints[1].ID != 2 {
		t.Errorf("Complaint ID is not set properly")
	}
	if complaints[2].ID != 3 {
		t.Errorf("Complaint ID is not set properly expected 3 got %d", complaints[2].ID)
	}

	complaint3 := hotstuff.Complaint{
		Complainee:  2,
		Complainant: 1,
	}
	complaintCache.AddComplaint(&complaint3)
	complaints = complaintCache.GetPendingComplaints()
	if len(complaints) != 4 {
		t.Errorf("GetPendingComplaints failed to fetch the complaints")
	}
	sort.Slice(complaints, func(i, j int) bool {
		return complaints[i].ID < complaints[j].ID
	})
	if complaints[1].ID != 1 {
		t.Errorf("Complaint ID is not set properly")
	}
}

func TestCommitComplaints(t *testing.T) {
	complaintCache := New()
	complaint := hotstuff.Complaint{
		Complainee:    1,
		Complainant:   2,
		ComplaintType: hotstuff.InvalidComplaint,
	}
	complaint1 := hotstuff.Complaint{
		Complainee:    1,
		Complainant:   2,
		ComplaintType: hotstuff.InvalidComplaint,
	}
	complaint2 := hotstuff.Complaint{
		Complainee:    1,
		Complainant:   2,
		ComplaintType: hotstuff.InvalidComplaint,
	}
	complaintCache.AddComplaint(&complaint)
	complaintCache.AddComplaint(&complaint1)
	complaints := complaintCache.GetPendingComplaints()

	complaintCache.CommitComplaints(complaints)
	score := complaintCache.GetScore()
	if score[hotstuff.ID(2)] != -2 {
		t.Errorf("GetScore: is not correct expected -2 got %d\n", score[hotstuff.ID(2)])
	}
	testComplaints := complaintCache.GetPendingComplaints()
	if len(testComplaints) != 0 {
		t.Errorf("GetPendingComplaints: returned complaints not correct")
	}

	complaint3 := hotstuff.Complaint{
		Complainee:    1,
		Complainant:   2,
		ComplaintType: hotstuff.InvalidComplaint,
	}
	complaintCache.AddComplaint(&complaint3)
	complaintCache.AddComplaint(&complaint2)
	complaints = complaintCache.GetPendingComplaints()
	if len(complaints) != 2 {
		t.Errorf("GetPendingComplaints failed to fetch the complaints")
	}
	sort.Slice(complaints, func(i, j int) bool {
		return complaints[i].ID < complaints[j].ID
	})
	if complaints[1].ID != 4 {
		t.Errorf("Complaint ID is not set properly")
	}

}

func TestCommitComplaintsWithInvalid(t *testing.T) {
	complaintCache := New()
	complaint := hotstuff.Complaint{
		Complainee:    1,
		Complainant:   2,
		ComplaintType: hotstuff.InvalidComplaint,
	}
	complaint1 := hotstuff.Complaint{
		Complainee:    1,
		Complainant:   2,
		ComplaintType: hotstuff.InvalidComplaint,
	}
	complaint2 := hotstuff.Complaint{
		Complainee:    1,
		Complainant:   2,
		ComplaintType: hotstuff.InvalidComplaint,
		ID:            1,
	}
	complaintCache.AddComplaint(&complaint)
	complaintCache.AddComplaint(&complaint1)
	complaints := complaintCache.GetPendingComplaints()
	complaints = append(complaints, &complaint2)
	complaintCache.CommitComplaints(complaints)
	score := complaintCache.GetScore()
	if score[hotstuff.ID(2)] != -2 {
		t.Errorf("GetScore: is not correct expected -2 got %d\n", score[hotstuff.ID(2)])
	}
}

func TestCommitComplaintsWithSuspicion(t *testing.T) {
	complaintCache := New()
	complaint := hotstuff.Complaint{
		Complainee:    1,
		Complainant:   2,
		ComplaintType: hotstuff.Suspicion,
	}
	complaint1 := hotstuff.Complaint{
		Complainee:    1,
		Complainant:   2,
		ComplaintType: hotstuff.Suspicion,
	}
	complaint2 := hotstuff.Complaint{
		Complainee:    1,
		Complainant:   2,
		ComplaintType: hotstuff.Suspicion,
		ID:            1,
	}
	complaintCache.AddComplaint(&complaint)
	complaintCache.AddComplaint(&complaint1)
	complaints := complaintCache.GetPendingComplaints()
	complaints = append(complaints, &complaint2)
	complaintCache.CommitComplaints(complaints)
	score := complaintCache.GetScore()
	if len(score) != 0 {
		t.Errorf("GetScore: set invalid scores")
	}
	susMatrix := complaintCache.GetSuspicionMatrix()
	if susMatrix[hotstuff.ID(1)][hotstuff.ID(2)] != 2 {
		t.Errorf("GetSuspicionMatrix: returned invalid scores")
	}
}

func TestCommitOtherComplaints(t *testing.T) {
	complaintCache := New()
	complaint := hotstuff.Complaint{
		Complainee:    2,
		Complainant:   1,
		ComplaintType: hotstuff.InvalidComplaint,
	}
	complaint1 := hotstuff.Complaint{
		Complainee:    2,
		Complainant:   4,
		ComplaintType: hotstuff.InvalidComplaint,
	}
	complaint2 := hotstuff.Complaint{
		Complainee:    2,
		Complainant:   3,
		ComplaintType: hotstuff.InvalidComplaint,
	}
	complaints := make([]*hotstuff.Complaint, 0)
	complaints = append(complaints, &complaint)
	complaints = append(complaints, &complaint1)
	complaints = append(complaints, &complaint2)
	complaintCache.CommitComplaints(complaints)
	score := complaintCache.GetScore()
	if score[hotstuff.ID(1)] != -1 {
		t.Errorf("GetScore: returned invalid scores")
	}
	if score[hotstuff.ID(2)] != 0 {
		t.Errorf("GetScore: returned invalid scores")
	}
	if score[hotstuff.ID(3)] != -1 {
		t.Errorf("GetScore: returned invalid scores")
	}
	if score[hotstuff.ID(4)] != -1 {
		t.Errorf("GetScore: returned invalid scores")
	}
}

func TestTopN(t *testing.T) {
	complaintCache := New()
	complaint := hotstuff.Complaint{
		Complainee:    2,
		Complainant:   1,
		ComplaintType: hotstuff.InvalidComplaint,
		ID:            1,
	}
	complaint1 := hotstuff.Complaint{
		Complainee:    2,
		Complainant:   4,
		ComplaintType: hotstuff.InvalidComplaint,
		ID:            2,
	}
	complaint2 := hotstuff.Complaint{
		Complainee:    2,
		Complainant:   3,
		ComplaintType: hotstuff.InvalidComplaint,
		ID:            3,
	}
	complaint3 := hotstuff.Complaint{
		Complainee:    2,
		Complainant:   3,
		ComplaintType: hotstuff.InvalidComplaint,
		ID:            4,
	}
	complaint4 := hotstuff.Complaint{
		Complainee:    2,
		Complainant:   4,
		ComplaintType: hotstuff.InvalidComplaint,
		ID:            5,
	}
	complaint5 := hotstuff.Complaint{
		Complainee:    2,
		Complainant:   4,
		ComplaintType: hotstuff.InvalidComplaint,
		ID:            6,
	}
	complaints := make([]*hotstuff.Complaint, 0)
	complaints = append(complaints, &complaint)
	complaints = append(complaints, &complaint1)
	complaints = append(complaints, &complaint2)
	complaints = append(complaints, &complaint3)
	complaints = append(complaints, &complaint4)
	complaints = append(complaints, &complaint5)
	complaintCache.CommitComplaints(complaints)
	topN, ok := complaintCache.GetTopN(2)
	fmt.Printf("topN are %v\n", topN)
	if !ok {
		t.Errorf("GetTopN: returned failure")
	}
	if len(topN) != 2 {
		t.Errorf("GetTopN: returned invalid length of the score")
	}
	sort.SliceStable(topN, func(i, j int) bool {
		return topN[i] < topN[j]
	})
	if topN[0] != hotstuff.ID(3) {
		t.Errorf("GetTopN: invalid nodes expected 3 got %d\n", topN[0])
	}
	if topN[1] != hotstuff.ID(4) {
		t.Errorf("GetTopN: invalid nodes expected 4 got %d\n", topN[1])
	}
	_, ok = complaintCache.GetTopN(0)
	if ok {
		t.Errorf("GetTopN: returned success in failure case")
	}
	_, ok = complaintCache.GetTopN(4)
	if ok {
		t.Errorf("GetTopN: returned success in failure case")
	}
}

func TestGetRobustInternalNodes(t *testing.T) {
	complaintCache := New()
	complaint := hotstuff.Complaint{
		Complainee:    2,
		Complainant:   1,
		ComplaintType: hotstuff.Suspicion,
		ID:            1,
	}
	complaint1 := hotstuff.Complaint{
		Complainee:    2,
		Complainant:   4,
		ComplaintType: hotstuff.Suspicion,
		ID:            2,
	}
	complaint2 := hotstuff.Complaint{
		Complainee:    2,
		Complainant:   3,
		ComplaintType: hotstuff.Suspicion,
		ID:            3,
	}
	complaint3 := hotstuff.Complaint{
		Complainee:    2,
		Complainant:   3,
		ComplaintType: hotstuff.Suspicion,
		ID:            4,
	}
	complaint4 := hotstuff.Complaint{
		Complainee:    2,
		Complainant:   4,
		ComplaintType: hotstuff.Suspicion,
		ID:            5,
	}
	complaint5 := hotstuff.Complaint{
		Complainee:    2,
		Complainant:   5,
		ComplaintType: hotstuff.Suspicion,
		ID:            6,
	}
	complaint6 := hotstuff.Complaint{
		Complainee:    2,
		Complainant:   6,
		ComplaintType: hotstuff.Suspicion,
		ID:            7,
	}
	complaint7 := hotstuff.Complaint{
		Complainee:    2,
		Complainant:   7,
		ComplaintType: hotstuff.Suspicion,
		ID:            8,
	}
	complaint8 := hotstuff.Complaint{
		Complainee:    2,
		Complainant:   4,
		ComplaintType: hotstuff.Suspicion,
		ID:            9,
	}
	complaints := make([]*hotstuff.Complaint, 0)
	complaints = append(complaints, &complaint)
	complaints = append(complaints, &complaint1)
	complaints = append(complaints, &complaint2)
	complaints = append(complaints, &complaint3)
	complaints = append(complaints, &complaint4)
	complaints = append(complaints, &complaint5)
	complaints = append(complaints, &complaint6)
	complaints = append(complaints, &complaint7)
	complaints = append(complaints, &complaint8)
	complaintCache.CommitComplaints(complaints)
	topN := complaintCache.GetRobustInternalNodes(4)
	fmt.Printf("GetRobustInternalNodes are %v\n", topN)
	if len(topN) != 4 {
		t.Errorf("GetTopN: returned invalid length of the score")
	}
	sort.SliceStable(topN, func(i, j int) bool {
		return topN[i] < topN[j]
	})
	if topN[0] != hotstuff.ID(1) {
		t.Errorf("GetTopN: invalid nodes expected 3 got %d\n", topN[0])
	}
	if topN[1] != hotstuff.ID(5) {
		t.Errorf("GetTopN: invalid nodes expected 4 got %d\n", topN[1])
	}
	if topN[2] != hotstuff.ID(6) {
		t.Errorf("GetTopN: invalid nodes expected 4 got %d\n", topN[1])
	}
	if topN[3] != hotstuff.ID(7) {
		t.Errorf("GetTopN: invalid nodes expected 4 got %d\n", topN[1])
	}
}
