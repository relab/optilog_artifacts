package bftsmart.optilog.monitors;

import bftsmart.optilog.sensors.SuspicionMeasurement;
import bftsmart.reconfiguration.ServerViewController;
import org.jgrapht.Graph;
import org.jgrapht.alg.clique.BronKerboschCliqueFinder;
import org.jgrapht.alg.matching.GreedyMaximumCardinalityMatching;
import org.jgrapht.generate.ComplementGraphGenerator;
import org.jgrapht.graph.DefaultWeightedEdge;
import org.jgrapht.graph.DirectedWeightedMultigraph;
import org.jgrapht.graph.SimpleWeightedGraph;
import org.jgrapht.alg.clique.DegeneracyBronKerboschCliqueFinder;
import org.jgrapht.Graphs;


//import org.jgrapht.alg.clique.;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;


import java.util.*;
import java.util.concurrent.TimeUnit;
import java.util.stream.Collectors;
import java.math.BigInteger;

public class SuspicionGraph {

    private Graph<Integer, DefaultWeightedEdge> suspicionGraph = new DirectedWeightedMultigraph<>(DefaultWeightedEdge.class);
    private final Logger logger = LoggerFactory.getLogger(this.getClass());
    private ServerViewController controller;

    public SuspicionGraph(ServerViewController controller) {
        this.controller = controller;
        // Add processes to the graph
        for (int i=0; i < controller.getCurrentViewN(); i++) {
            suspicionGraph.addVertex(i);
        }
    }

    private synchronized void addSuspicion(int reporter, int suspect) {

        if (controller.getStaticConf().getProcessId() == 1) {
            logger.info("OptiLog > SuspicionGraph > addSuspicion called reporter {} , suspect {} ", reporter, suspect);
        }
        if (reporter == suspect) {
            return; // graph does not allow self-loops
        }

        suspicionGraph.addVertex(suspect);
        suspicionGraph.addVertex(reporter);

        DefaultWeightedEdge edge = suspicionGraph.getEdge(reporter, suspect);
        if (edge == null) {
            DefaultWeightedEdge suspicion = suspicionGraph.addEdge(reporter, suspect);
            suspicionGraph.setEdgeWeight(suspicion, 1.0); // add a suspicion
        } else {
            suspicionGraph.setEdgeWeight(edge, suspicionGraph.getEdgeWeight(edge) + 1.0); // Increase suspicion weight by 1
        }
    }

    public synchronized void populate(List<SuspicionMeasurement> filteredSuspicions) {
        for (SuspicionMeasurement s: filteredSuspicions) {
            addSuspicion(s.getReporter(), s.getSuspect());
        }
    }

    public synchronized void removeSuspicions(double strength) {
        Set<DefaultWeightedEdge> edges = suspicionGraph.edgeSet();
        Set<DefaultWeightedEdge> toRemove = new LinkedHashSet<>();
        for (DefaultWeightedEdge edge : edges) {
            suspicionGraph.setEdgeWeight(edge, suspicionGraph.getEdgeWeight(edge) - strength); // Decrease suspicion strength
            if (suspicionGraph.getEdgeWeight(edge) <= 0) {
                toRemove.add(edge); // Remove suspicion
            }
        }
        if (!toRemove.isEmpty()) {
            suspicionGraph.removeAllEdges(toRemove);
        }
    }

    public synchronized void clearSuspicions() {
        suspicionGraph = new DirectedWeightedMultigraph<>(DefaultWeightedEdge.class);
    }

    public static Set<Integer> maxIndependentSet(Graph<Integer, DefaultWeightedEdge>  sGraph, boolean exhaustive) {
        // Step 1: Compute the undirected graph equivalent (turns directed edges into undirected)
        Graph<Integer, DefaultWeightedEdge> undirectedGraph = convertToUndirected(sGraph);

        // Step 2: Compute the complementary Graph G' (contains an edge e where is no edge in G and vice versa)
        //     Create an empty weighted graph for the complement
        Graph<Integer, DefaultWeightedEdge> complement = new SimpleWeightedGraph<>(DefaultWeightedEdge.class);
        //      Use ComplementGraphGenerator to populate it
        ComplementGraphGenerator<Integer, DefaultWeightedEdge> generator = new ComplementGraphGenerator<>(undirectedGraph);
        generator.generateGraph(complement);

        Iterator<Set<Integer>> maxIter;

        // Step 3: Run the Clique Finder on the complement graph (this is equivalent of finding the independent set):
        if (exhaustive) {
            BronKerboschCliqueFinder<Integer, DefaultWeightedEdge> cliqueFinder =
                    new BronKerboschCliqueFinder<>(complement);
            maxIter = cliqueFinder.maximumIterator();
        } else {    // heuristically find independent set
            DegeneracyBronKerboschCliqueFinder<Integer, DefaultWeightedEdge> cliqueFinder =
                    new DegeneracyBronKerboschCliqueFinder<>(complement, 100000, TimeUnit.MILLISECONDS);
            maxIter = cliqueFinder.maximumIterator();
        }

        Set<Integer> maxIndependentSet = maxIter.hasNext() ? maxIter.next() : Collections.emptySet();

        return maxIndependentSet;
    }


    public synchronized Set<Integer> candidateSet() {

        //Set<Integer> maxIndependentSet = maxIndependentSet(this.suspicionGraph, false);
        Set<Integer> maxIndependentSet = greedyIndependentSet(this.suspicionGraph);

        // Output the final result
        logger.info(">>> OptiLog: SuspicionGraph: Vertices in maximum independent set: " + maxIndependentSet);

        // Return max Independent set if sufficiently large
        if (maxIndependentSet.size() >= controller.getCurrentViewF() + 1) {
            return maxIndependentSet;
        }
        // If not large enough use an heuristic to craft a candidate set
        // (this might be the case during network disruptions)
        logger.warn(">>> OptiLog: SuspicionGraph: Using heuristic to craft candidate set");

        return heuristicCandidateSet();
    }

    // Find all triangles in the graph
    private Set<Set<Integer>> findTriangles(Graph<Integer, DefaultWeightedEdge> graph) {
        Set<Set<Integer>> triangles = new LinkedHashSet<>();

        for (Integer node : graph.vertexSet()) {
            for (Integer neighbor1 : graph.vertexSet()) {
                if (!graph.containsEdge(node, neighbor1) && !graph.containsEdge(neighbor1, node)) continue;
                for (Integer neighbor2 : graph.vertexSet()) {
                    if (node.equals(neighbor1) || node.equals(neighbor2) || neighbor1.equals(neighbor2)) continue;
                    if (graph.containsEdge(neighbor1, neighbor2) && graph.containsEdge(neighbor2, node)) {
                        Set<Integer> triangle = new TreeSet<>(Arrays.asList(node, neighbor1, neighbor2));
                        triangles.add(triangle);
                    }
                }
            }
        }
        return triangles;
    }

    // Find vertices that are:
    // 1. In a triangle
    // 2. Where the triangle has an edge in the matching
    // 3. Not adjacent to any edge in the matching
    private Set<Integer> findVerticesMeetingConditions(Graph<Integer, DefaultWeightedEdge> graph,
                                                       Set<Set<Integer>> triangles,
                                                       Set<DefaultWeightedEdge> matchingEdges) {
        Set<Integer> matchedVertices = new TreeSet<>();
        for (DefaultWeightedEdge edge : matchingEdges) {
            matchedVertices.add(graph.getEdgeSource(edge));
            matchedVertices.add(graph.getEdgeTarget(edge));
        }

        Set<Integer> resultSet = new TreeSet<>();
        for (Set<Integer> triangle : triangles) {
            boolean hasMatchingEdge = false;
            for (DefaultWeightedEdge edge : matchingEdges) {
                Integer u = graph.getEdgeSource(edge);
                Integer v = graph.getEdgeTarget(edge);
                if (triangle.contains(u) && triangle.contains(v)) {
                    hasMatchingEdge = true;
                    break;
                }
            }
            if (hasMatchingEdge) {
                for (Integer vertex : triangle) {
                    if (!matchedVertices.contains(vertex)) {
                        resultSet.add(vertex);
                    }
                }
            }
        }
        return resultSet;
    }

    // Make sure exclusionList will not grow beyond the size of (n-f-1) so f+1 candidates are always available
    private Set<Integer> reduceExclusionList(Set<Integer> suspects) {

        LinkedList<Integer> sortedList = new LinkedList<>(suspects);
        // Sort by degree (in-degree), ascending order
        sortedList.sort((a, b) -> Integer.compare(suspicionGraph.inDegreeOf(a), suspicionGraph.inDegreeOf(b)));
        while (sortedList.size() > controller.getCurrentViewN() - controller.getCurrentViewF() - 1) {
            sortedList.removeFirst();
        }
        suspects = new TreeSet<>(sortedList);
        return suspects;
    }

    public synchronized void printGraphAscii() {
        System.out.println("Suspicion Graph (ASCII Representation):");

        // Store formatted edges
        List<String> edgeRepresentations = new ArrayList<>();

        for (DefaultWeightedEdge edge : suspicionGraph.edgeSet()) {
            int source = suspicionGraph.getEdgeSource(edge);
            int target = suspicionGraph.getEdgeTarget(edge);
            double weight = suspicionGraph.getEdgeWeight(edge);

            // Format the edge visually
            edgeRepresentations.add(String.format("%d --(%.2f)--> %d", source, weight, target));
        }

        // Print all edges
        if (edgeRepresentations.isEmpty()) {
            System.out.println("[Graph is empty]");
        } else {
            edgeRepresentations.forEach(System.out::println);
        }
    }

    private static Graph<Integer, DefaultWeightedEdge> convertToUndirected(Graph<Integer, DefaultWeightedEdge> directedGraph) {
        // Create an undirected graph
        Graph<Integer, DefaultWeightedEdge> undirectedGraph = new SimpleWeightedGraph<>(DefaultWeightedEdge.class);

        // Add all vertices
        for (Integer vertex : directedGraph.vertexSet()) {
            undirectedGraph.addVertex(vertex);
        }

        // Find bidirectional edges
        Set<DefaultWeightedEdge> toKeep = new LinkedHashSet<>();

        for (DefaultWeightedEdge edge : directedGraph.edgeSet()) {
            Integer source = directedGraph.getEdgeSource(edge);
            Integer target = directedGraph.getEdgeTarget(edge);

            // Check if the reverse edge exists
            if (directedGraph.containsEdge(target, source)) {
                toKeep.add(edge);
            }
        }

        // Add only bidirectional edges to the new undirected graph
        for (DefaultWeightedEdge edge : toKeep) {
            Integer source = directedGraph.getEdgeSource(edge);
            Integer target = directedGraph.getEdgeTarget(edge);

            if (source != null & target != null) {
                undirectedGraph.addEdge(source, target);
            }
        }

        return undirectedGraph;
    }

    private Set<Integer> heuristicCandidateSet(){
        // Step 1: Compute maximal set of disjoint edges
        Graph<Integer, DefaultWeightedEdge> undirectedGraph = convertToUndirected(suspicionGraph);
        GreedyMaximumCardinalityMatching<Integer, DefaultWeightedEdge> matchingAlgo = new GreedyMaximumCardinalityMatching<>(undirectedGraph, false);
        Set<DefaultWeightedEdge> matchingEdges = matchingAlgo.getMatching().getEdges();

        logger.info(">>> OptiLog: SuspicionGraph: Maximal set of disjoint edges: {}", matchingEdges);

        // Step 2: Find all triangles in the graph
        Set<Set<Integer>> triangles = findTriangles(suspicionGraph);

        // Step 3: Identify vertices fulfilling the criteria from the paper:
        Set<Integer> condition = findVerticesMeetingConditions(suspicionGraph, triangles, matchingEdges);

        // Output the final result
        logger.info(">>> OptiLog: SuspicionGraph: Vertices fulfilling the second conditions: " + condition);


        // Init result set with *all* system nodes
        Set<Integer> resultSet = Arrays.stream(controller.getCurrentView().getProcesses()).boxed().collect(Collectors.toSet());

        // Now build the set that is excluded from being candidates:
        Set<Integer> exclusionList = new LinkedHashSet<>();
        for (DefaultWeightedEdge edge : matchingEdges) {
            exclusionList.add(suspicionGraph.getEdgeSource(edge));
            exclusionList.add(suspicionGraph.getEdgeTarget(edge));
        }
        exclusionList.addAll(condition); // Adds only non-duplicate elements

        // Ensure candidate set is at least of size f+1 so exclusion least cant be larger than n-f-1
        if (exclusionList.size() > (controller.getCurrentViewN() - controller.getCurrentViewF() - 1)) {
            exclusionList = reduceExclusionList(exclusionList);
        }
        resultSet.removeAll(exclusionList);

        logger.info(">>> OptiLog: SuspicionGraph: CandidateSet is " + resultSet);

        return resultSet;
    }

    public static void main(String[] args) {


        System.out.println("Starting experiment: Warm up JVM ..");
        // Warm-up loop
        for (int i = 0; i < 10; i++) {
            System.out.println("Starting experiment: Round .. " + i);
            unittest(10, false);
        }
        System.out.println("Ready to benchmark");
        System.out.println("f    n    time    std");
        for (int i = 1; i <= 67; i++) {
            unittest(i, true);
        }
    }

    private static long calcIndependentSetTime(int f, int n) {
        Graph<Integer, DefaultWeightedEdge> sGraph = new DirectedWeightedMultigraph<>(DefaultWeightedEdge.class);
        for (int i = 0; i < n; i++) {
            sGraph.addVertex(i);
        }
        // there should be roughly 2*f*f edges, i.e. each of f Byzantine nodes blames f random correct nodes
        for (int i = 0; i < f; i++) {
            Random r = new Random();
            for (int j=0; j<f; j++) {

                // Blame a random correct node
                int random = r.nextInt(3*f+1);
                if (i != random) {
                    sGraph.addEdge(i, random);
                    sGraph.addEdge(random, i);
                }

            }
        }
        long start = System.nanoTime();
        Set<Integer> maxIndependentSet = //greedyIndependentSet(sGraph);
                maxIndependentSet(sGraph, false);
        long end = System.nanoTime();
        return ((end - start));
    }

    public static void unittest(int scale, boolean verbose) {

        int f = scale;
        int n = 3*f+1;


        int repetitions = 100;
        long[] times = new long[repetitions];


        for (int i=0; i< repetitions; i++){
            // Cache eviction
            byte[] trash = new byte[10 * 1024*1024]; // 10MB
            for (int j = 0; j < trash.length; j++) {
                trash[j] = (byte) i;
            }
            times[i] = calcIndependentSetTime(f, n);
        }
        long average = 0;
        for (long time : times) {
            average += time;
        }
        average = average / repetitions;

        BigInteger sumSquaredDiffs = BigInteger.ZERO;
        BigInteger averageBI = BigInteger.valueOf(average);

        for (long time : times) {
            BigInteger timeBI = BigInteger.valueOf(time);
            BigInteger diff = timeBI.subtract(averageBI);
            BigInteger squaredDiff = diff.multiply(diff);
            sumSquaredDiffs = sumSquaredDiffs.add(squaredDiff);
        }

// Convert to double for standard deviation calculation
        double std = Math.sqrt(sumSquaredDiffs.doubleValue() / repetitions);

        if (verbose)    System.out.println(f + "    " + n + "    " + average/1000 + "    " + std/1000);

 /*
        System.out.println();
        System.out.println("______________________________________");
        System.out.println("____________Next Test_________________");
        System.out.println("______________________________________");
        System.out.println("n is " + n + ", f is " + f);
        System.out.println("Expect independent set to be of size 2f+1 which is " + (2*f+1));
        System.out.println( "Number of vertices: " + sGraph.vertexSet().size());
        System.out.println( "Number of edges: " + sGraph.edgeSet().size());

        System.out.println( "Start Search for independent set");

  */


        // Output the final result
        /*
        System.out.println( "maximum independent set: " + maxIndependentSet);

        System.out.println( "Size of maximum independent set: " + maxIndependentSet.size());
        System.out.println("Test completed in " + ((end - start)/1000000) + " ms");

         */


      //  System.out.println("______________________________________");
    }


    public static Set<Integer> greedyIndependentSet(Graph<Integer, DefaultWeightedEdge> graph) {
        Set<Integer> independentSet = new LinkedHashSet<>();
        Set<Integer> excluded = new HashSet<>();

        // Work on a copy of the vertex set, sorted by ascending degree (greedy heuristic)
        List<Integer> sortedVertices = new ArrayList<>(graph.vertexSet());
        sortedVertices.sort(Comparator.comparingInt(graph::degreeOf)); // prefer low-degree nodes

        for (Integer v : sortedVertices) {
            if (!excluded.contains(v)) {
                // Add v to independent set
                independentSet.add(v);

                // Exclude v and all its neighbors from future consideration
                excluded.add(v);
                excluded.addAll(Graphs.neighborListOf(graph, v));
            }
        }
        return independentSet;
    }


    public synchronized boolean existsEdge(int reporter, int suspect) {
        if (reporter == suspect) {
            return true; // graph does not allow self-loops. Return *true* because edge will never be inserted anyways
        }
        return suspicionGraph.containsEdge(reporter, suspect);
    }


}
