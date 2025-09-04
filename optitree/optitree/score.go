package main

import (
	"slices"
)

// qcLatency returns the latency to obtain a quorum certificate (QC) from the give tree.
func (l Latencies) qcLatency(quorumSize, branchFactor int, all []node) Latency {
	root, nodes := all[0], all[1:]

	// Top-down dissemination of votes:
	// Disseminate from root to internal nodes:
	for i := range branchFactor {
		nodes[i].disseminated = root.disseminated + l.pathLatency(root.id, nodes[i].id)
		// Disseminate from internal nodes to all leaves
		for j := range branchFactor {
			leafIndex := branchFactor*i + j + branchFactor
			if leafIndex >= len(nodes) {
				break
			}
			nodes[leafIndex].disseminated = nodes[i].disseminated + l.pathLatency(nodes[i].id, nodes[leafIndex].id)
			nodes[leafIndex].aggregated = nodes[leafIndex].disseminated // Return instantly the vote
			nodes[leafIndex].delivered = nodes[leafIndex].aggregated + l.pathLatency(nodes[leafIndex].id, nodes[i].id)
		}
	}
	// Dissemination finished!
	// Aggregation starts!
	for i := range branchFactor {
		nodes[i].aggregated = nodes[i].disseminated
		for j := range branchFactor {
			leafIndex := branchFactor*i + j + branchFactor
			if leafIndex >= len(nodes) {
				break
			}
			nodes[i].aggregated = max(nodes[i].aggregated, nodes[leafIndex].delivered)
			nodes[i].votes += nodes[leafIndex].votes
		}
		nodes[i].delivered = nodes[i].aggregated + l.pathLatency(nodes[i].id, root.id)
	}
	// Sort ascending arrival times of votes at leader
	internalNodes := nodes[:branchFactor]
	orderByLatency(branchFactor, internalNodes, nodes)

	// Collect QC latency
	for _, internal := range internalNodes {
		if root.votes >= quorumSize {
			return root.aggregated
		}
		root.aggregated = max(root.aggregated, internal.delivered)
		root.votes += internal.votes
	}
	return root.aggregated
}

// orderByLatency reorders the internal nodes and the leaves according
// to the latency of the internal nodes.
// Note that if we only care about the QC latency we can ignore the leaves.
func orderByLatency(bf int, internalNodes, nodes []node) {
	original := slices.Clone(internalNodes)
	slices.SortFunc(internalNodes, func(i, j node) int {
		return int(i.delivered - j.delivered)
	})
	if slices.Equal(original, internalNodes) {
		return
	}
	// We don't support reordering leaves for partial trees.
	if len(nodes)+1 == TreeSize(bf) {
		// Reorder leaves according to the reordering of the internal nodes.
		// This is only necessary for emitting the correct latency optimal tree;
		// the QC latency is not affected by the order of the leaves.
		// We could potentially optimize this by only reordering the leaves
		// once for the final optimal tree instead of doing it for every tree.
		src := slices.Clone(nodes)
		for from := range original {
			to := slices.Index(internalNodes, original[from])
			fs, fe := leafRange(from, bf)
			ts, te := leafRange(to, bf)
			copy(nodes[ts:te], src[fs:fe])
		}
	}
}

func leafRange(i, bf int) (int, int) {
	s := bf*i + bf
	return s, s + bf
}
