package kauri

import (
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"
)

// When running benchmarks, this can be set to true to avoid printing during the benchmark.
var benchmarking = false

var (
	regions []string
	cities  []string
)

type Latency int32

func (l Latency) String() string {
	// convert from microseconds to milliseconds
	return time.Duration(l * 1000).String()
}

type Latencies [][]Latency

func NewLatencies(size int) Latencies {
	latencies := make(Latencies, size)
	for i := range latencies {
		latencies[i] = make([]Latency, size)
	}
	return latencies
}

func NewRand(size int) Latencies {
	const maxLatency = 1000 // 1s
	r := rand.New(rand.NewPCG(1, 2))
	latencies := make(Latencies, size)
	for i := range latencies {
		latencies[i] = make([]Latency, size)
		for j := range latencies[i] {
			if i == j {
				latencies[i][j] = 0
				continue
			}
			latencies[i][j] = Latency(r.Int32N(maxLatency))
		}
	}
	return latencies
}

// duration returns the latency between two cities in the latencies matrix as a time.Duration.
func (l Latencies) duration(from, to int) time.Duration {
	return time.Duration(l[from][to] * 1000)
}

// pathLatency returns the latency between two cities in the latencies matrix.
// The unit of the latency is microseconds.
func (l Latencies) pathLatency(from, to int) Latency {
	return l[from][to]
}

// treeLatency calculates the total latency from the root, through one intermediate level, to the leaf nodes.
func (l Latencies) treeLatency(root, branchFactor int, tree []int) (total Latency) {
	// latency from root to each intermediate node
	for i := range branchFactor {
		total += l.pathLatency(root, tree[i])

		// latency from intermediate node to leaves
		for j := range branchFactor {
			leafIndex := branchFactor*i + j + branchFactor
			total += l.pathLatency(tree[i], tree[leafIndex])
		}
	}
	return total
}

// func (l Latencies) PrintNodes(nodes []node, bf int) {
// 	if benchmarking {
// 		return
// 	}
// 	tot := time.Duration(0)
// 	root, nodes := nodes[0], nodes[1:]
// 	tw := tabwriter.NewWriter(os.Stdout, 2, 8, 2, ' ', 0)
// 	fmt.Fprintf(tw, "%2d (%s)\n", root.id, cityName(root.id))
// 	for i := range bf {
// 		lat := l.duration(root.id, nodes[i].id)
// 		tot += lat
// 		fmt.Fprintf(tw, " |-- %2d (%s)\tlat: %s\tvotes: %d\tdisseminated: %d\taggregated: %d\tdelivered: %d\n",
// 			nodes[i].id, cityName(nodes[i].id), lat, nodes[i].votes, nodes[i].disseminated, nodes[i].aggregated, nodes[i].delivered)
// 		for j := range bf {
// 			leafIndex := bf*i + j + bf
// 			lat = l.duration(nodes[i].id, nodes[leafIndex].id)
// 			tot += lat
// 			fmt.Fprintf(tw, "      |-- %2d (%s)\tlat: %s\tvotes: %d\tdisseminated: %d\taggregated: %d\tdelivered: %d\n",
// 				nodes[leafIndex].id, cityName(nodes[leafIndex].id), lat, nodes[leafIndex].votes, nodes[leafIndex].disseminated, nodes[leafIndex].aggregated, nodes[leafIndex].delivered)
// 		}
// 	}
// 	tw.Flush()
// 	fmt.Printf("Total latency: %s\n", tot)
// }

func (l Latencies) Print(tree []int, bf int) {
	tot := time.Duration(0)
	root, tree := tree[0], tree[1:]
	tw := tabwriter.NewWriter(os.Stdout, 2, 8, 2, ' ', 0)
	fmt.Fprintf(tw, "%2d (%s)\n", root, cityName(root))
	for i := range bf {
		lat := l.duration(root, tree[i])
		tot += lat
		fmt.Fprintf(tw, " |-- %2d (%s)\tlat: %s\n", tree[i], cityName(tree[i]), lat)
		for j := range bf {
			leafIndex := bf*i + j + bf
			lat = l.duration(tree[i], tree[leafIndex])
			tot += lat
			fmt.Fprintf(tw, "      |-- %2d (%s)\tlat: %s\n", tree[leafIndex], cityName(tree[leafIndex]), lat)
		}
	}
	tw.Flush()
	fmt.Printf("Total latency: %s\n", tot)
}

func cityName(index int) string {
	return cities[index]
}

func loadLatencies(csvFile string) (Latencies, error) {
	b, err := os.ReadFile(csvFile)
	if err != nil {
		return nil, err
	}
	return parseLatencies(string(b))
}

func parseLatencies(csvData string) (Latencies, error) {
	regionLine := strings.Split(csvData, "\n")
	latencies := NewLatencies(len(regionLine))
	for from, row := range regionLine {
		if row == "" {
			break
		}
		if row[0] == ',' {
			continue
		}
		cols := strings.Split(row, ",")
		region := cols[0]
		regions = append(regions, region)
		if strings.Contains(region, "(") {
			cities = append(cities, city(region))
		} else {
			cities = append(cities, region)
		}
		for to, col := range cols[1:] {
			if col == "" {
				break
			}
			value, err := strconv.ParseFloat(col, 64)
			if err != nil {
				return nil, err
			}
			value *= 1000                                    // multiply to keep three decimal places
			value /= 2                                       // divide by 2 to convert RTT latency to one-way
			latencies[from][to] = Latency(math.Round(value)) // microseconds
		}
	}
	return latencies, nil
}

func city(region string) string {
	return region[strings.Index(region, "(")+1 : strings.Index(region, ")")]
}
