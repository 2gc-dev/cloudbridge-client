package wireguard

import (
	"container/heap"
	"fmt"
	"math"
	"sync"
	"time"

	"go.uber.org/zap"
)

// MeshTopology represents the topology of the mesh network
type MeshTopology struct {
	nodes       map[string]*MeshNode
	nodesMutex  sync.RWMutex
	connections map[string]*MeshConnection
	connMutex   sync.RWMutex
	routes      map[string]*MeshRoute
	routesMutex sync.RWMutex
	discovery   *PeerDiscovery
	logger      *zap.Logger
	metrics     *TopologyMetrics
}

// MeshConnection represents a connection between two nodes
type MeshConnection struct {
	ID           string
	SourceNode   string
	TargetNode   string
	Latency      time.Duration
	Bandwidth    int64 // bytes per second
	Reliability  float64 // 0.0 to 1.0
	Status       ConnectionStatus
	LastUpdated  time.Time
	Cost         float64 // calculated cost for routing
}

// ConnectionStatus represents the status of a mesh connection
type ConnectionStatus string

const (
	ConnectionStatusDown     ConnectionStatus = "down"
	ConnectionStatusUp       ConnectionStatus = "up"
	ConnectionStatusDegraded ConnectionStatus = "degraded"
)

// MeshRoute represents a route in the mesh network
type MeshRoute struct {
	ID           string
	Source       string
	Destination  string
	Path         []string // sequence of node IDs
	Latency      time.Duration
	Bandwidth    int64
	Reliability  float64
	Cost         float64
	LastUpdated  time.Time
}

// TopologyMetrics represents metrics for the mesh topology
type TopologyMetrics struct {
	TotalNodes       int64
	TotalConnections int64
	TotalRoutes      int64
	AverageLatency   time.Duration
	AverageBandwidth int64
	NetworkDiameter  int
	LastOptimization time.Time
}

// MeshTopologyManager manages the mesh topology
type MeshTopologyManager struct {
	topology    *MeshTopology
	discovery   *PeerDiscovery
	router      *MeshRouter
	logger      *zap.Logger
	config      *TopologyConfig
}

// TopologyConfig represents configuration for topology management
type TopologyConfig struct {
	OptimizationInterval time.Duration
	MaxConnections       int
	MinReliability       float64
	MaxLatency           time.Duration
	EnableAutoOptimization bool
}

// NewMeshTopology creates a new mesh topology
func NewMeshTopology(discovery *PeerDiscovery, logger *zap.Logger) *MeshTopology {
	return &MeshTopology{
		nodes:       make(map[string]*MeshNode),
		connections: make(map[string]*MeshConnection),
		routes:      make(map[string]*MeshRoute),
		discovery:   discovery,
		logger:      logger,
		metrics:     &TopologyMetrics{},
	}
}

// NewMeshTopologyManager creates a new topology manager
func NewMeshTopologyManager(topology *MeshTopology, config *TopologyConfig, logger *zap.Logger) *MeshTopologyManager {
	if config == nil {
		config = &TopologyConfig{
			OptimizationInterval:   5 * time.Minute,
			MaxConnections:        10,
			MinReliability:        0.8,
			MaxLatency:            100 * time.Millisecond,
			EnableAutoOptimization: true,
		}
	}

	router := NewMeshRouter(topology, logger)
	return &MeshTopologyManager{
		topology: topology,
		discovery: topology.discovery,
		router:   router,
		logger:   logger,
		config:   config,
	}
}

// AddNode adds a node to the topology
func (mt *MeshTopology) AddNode(node *MeshNode) {
	mt.nodesMutex.Lock()
	defer mt.nodesMutex.Unlock()

	mt.nodes[node.ID] = node
	mt.metrics.TotalNodes++

	mt.logger.Info("Added node to topology",
		zap.String("node_id", node.ID),
		zap.String("endpoint", node.Endpoint.String()))
}

// RemoveNode removes a node from the topology
func (mt *MeshTopology) RemoveNode(nodeID string) {
	mt.nodesMutex.Lock()
	defer mt.nodesMutex.Unlock()

	if _, exists := mt.nodes[nodeID]; exists {
		delete(mt.nodes, nodeID)
		mt.metrics.TotalNodes--

		// Remove all connections involving this node
		mt.connMutex.Lock()
		for connID, conn := range mt.connections {
			if conn.SourceNode == nodeID || conn.TargetNode == nodeID {
				delete(mt.connections, connID)
				mt.metrics.TotalConnections--
			}
		}
		mt.connMutex.Unlock()

		// Remove all routes involving this node
		mt.routesMutex.Lock()
		for routeID, route := range mt.routes {
			if route.Source == nodeID || route.Destination == nodeID {
				delete(mt.routes, routeID)
				mt.metrics.TotalRoutes--
			}
		}
		mt.routesMutex.Unlock()

		mt.logger.Info("Removed node from topology", zap.String("node_id", nodeID))
	}
}

// AddConnection adds a connection between two nodes
func (mt *MeshTopology) AddConnection(sourceNode, targetNode string, latency time.Duration, bandwidth int64, reliability float64) {
	mt.connMutex.Lock()
	defer mt.connMutex.Unlock()

	connID := fmt.Sprintf("%s-%s", sourceNode, targetNode)
	
	connection := &MeshConnection{
		ID:          connID,
		SourceNode:  sourceNode,
		TargetNode:  targetNode,
		Latency:     latency,
		Bandwidth:   bandwidth,
		Reliability: reliability,
		Status:      ConnectionStatusUp,
		LastUpdated: time.Now(),
		Cost:        mt.calculateConnectionCost(latency, bandwidth, reliability),
	}

	mt.connections[connID] = connection
	mt.metrics.TotalConnections++

	mt.logger.Debug("Added connection to topology",
		zap.String("connection_id", connID),
		zap.String("source", sourceNode),
		zap.String("target", targetNode),
		zap.Duration("latency", latency))
}

// RemoveConnection removes a connection
func (mt *MeshTopology) RemoveConnection(connID string) {
	mt.connMutex.Lock()
	defer mt.connMutex.Unlock()

	if _, exists := mt.connections[connID]; exists {
		delete(mt.connections, connID)
		mt.metrics.TotalConnections--

		mt.logger.Info("Removed connection from topology", zap.String("connection_id", connID))
	}
}

// GetNode returns a node by ID
func (mt *MeshTopology) GetNode(nodeID string) (*MeshNode, bool) {
	mt.nodesMutex.RLock()
	defer mt.nodesMutex.RUnlock()

	node, exists := mt.nodes[nodeID]
	return node, exists
}

// GetAllNodes returns all nodes
func (mt *MeshTopology) GetAllNodes() []*MeshNode {
	mt.nodesMutex.RLock()
	defer mt.nodesMutex.RUnlock()

	nodes := make([]*MeshNode, 0, len(mt.nodes))
	for _, node := range mt.nodes {
		nodes = append(nodes, node)
	}
	return nodes
}

// GetConnection returns a connection by ID
func (mt *MeshTopology) GetConnection(connID string) (*MeshConnection, bool) {
	mt.connMutex.RLock()
	defer mt.connMutex.RUnlock()

	conn, exists := mt.connections[connID]
	return conn, exists
}

// GetAllConnections returns all connections
func (mt *MeshTopology) GetAllConnections() []*MeshConnection {
	mt.connMutex.RLock()
	defer mt.connMutex.RUnlock()

	connections := make([]*MeshConnection, 0, len(mt.connections))
	for _, conn := range mt.connections {
		connections = append(connections, conn)
	}
	return connections
}

// calculateConnectionCost calculates the cost of a connection
func (mt *MeshTopology) calculateConnectionCost(latency time.Duration, bandwidth int64, reliability float64) float64 {
	// Normalize latency (0-1 scale, lower is better)
	latencyCost := float64(latency.Milliseconds()) / 1000.0 // normalize to seconds
	
	// Normalize bandwidth (0-1 scale, higher is better)
	bandwidthCost := 1.0 - (float64(bandwidth) / (100 * 1024 * 1024)) // normalize to 100MB/s
	
	// Reliability cost (0-1 scale, higher is better)
	reliabilityCost := 1.0 - reliability
	
	// Weighted combination
	return latencyCost*0.4 + bandwidthCost*0.3 + reliabilityCost*0.3
}

// BuildOptimalTopology builds an optimal topology using minimum spanning tree
func (mtm *MeshTopologyManager) BuildOptimalTopology() error {
	mtm.logger.Info("Building optimal topology")

	nodes := mtm.topology.GetAllNodes()
	if len(nodes) < 2 {
		return fmt.Errorf("need at least 2 nodes to build topology")
	}

	// Build minimum spanning tree
	mst := mtm.buildMinimumSpanningTree(nodes)
	
	// Add redundant connections for fault tolerance
	redundant := mtm.addRedundantConnections(mst)
	
	// Optimize routes
	optimized := mtm.optimizeRoutes(redundant)
	
	// Apply topology
	return mtm.applyTopology(optimized)
}

// buildMinimumSpanningTree builds a minimum spanning tree using Kruskal's algorithm
func (mtm *MeshTopologyManager) buildMinimumSpanningTree(nodes []*MeshNode) []*MeshConnection {
	// Create all possible connections
	var edges []*MeshConnection
	for i := 0; i < len(nodes); i++ {
		for j := i + 1; j < len(nodes); j++ {
			latency := mtm.calculateLatency(nodes[i], nodes[j])
			bandwidth := mtm.calculateBandwidth(nodes[i], nodes[j])
			reliability := mtm.calculateReliability(nodes[i], nodes[j])
			
			conn := &MeshConnection{
				SourceNode:  nodes[i].ID,
				TargetNode:  nodes[j].ID,
				Latency:     latency,
				Bandwidth:   bandwidth,
				Reliability: reliability,
				Cost:        mtm.topology.calculateConnectionCost(latency, bandwidth, reliability),
			}
			edges = append(edges, conn)
		}
	}

	// Sort edges by cost
	heap.Init(&EdgeHeap{edges})

	// Union-Find data structure for cycle detection
	uf := NewUnionFind(len(nodes))
	nodeMap := make(map[string]int)
	for i, node := range nodes {
		nodeMap[node.ID] = i
	}

	var mst []*MeshConnection
	for len(edges) > 0 && len(mst) < len(nodes)-1 {
		edge := heap.Pop(&EdgeHeap{edges}).(*MeshConnection)
		
		sourceIdx := nodeMap[edge.SourceNode]
		targetIdx := nodeMap[edge.TargetNode]
		
		if uf.Find(sourceIdx) != uf.Find(targetIdx) {
			uf.Union(sourceIdx, targetIdx)
			mst = append(mst, edge)
		}
	}

	return mst
}

// addRedundantConnections adds redundant connections for fault tolerance
func (mtm *MeshTopologyManager) addRedundantConnections(mst []*MeshConnection) []*MeshConnection {
	// For now, we'll add a few additional connections based on cost
	// In a real implementation, you might use more sophisticated algorithms
	
	connections := make([]*MeshConnection, len(mst))
	copy(connections, mst)
	
	// Add some redundant connections (up to MaxConnections)
	if len(connections) < mtm.config.MaxConnections {
		// This is a simplified approach - in reality you'd want more sophisticated logic
		mtm.logger.Debug("Adding redundant connections for fault tolerance")
	}
	
	return connections
}

// optimizeRoutes optimizes routes in the topology
func (mtm *MeshTopologyManager) optimizeRoutes(connections []*MeshConnection) []*MeshConnection {
	// For now, we'll just return the connections as-is
	// In a real implementation, you might:
	// 1. Calculate shortest paths between all pairs
	// 2. Optimize for latency, bandwidth, or reliability
	// 3. Implement load balancing
	
	mtm.logger.Debug("Optimizing routes")
	return connections
}

// applyTopology applies the topology to the network
func (mtm *MeshTopologyManager) applyTopology(connections []*MeshConnection) error {
	mtm.logger.Info("Applying topology", zap.Int("connections", len(connections)))
	
	// Clear existing connections
	mtm.topology.connMutex.Lock()
	mtm.topology.connections = make(map[string]*MeshConnection)
	mtm.topology.metrics.TotalConnections = 0
	mtm.topology.connMutex.Unlock()
	
	// Add new connections
	for _, conn := range connections {
		mtm.topology.AddConnection(
			conn.SourceNode,
			conn.TargetNode,
			conn.Latency,
			conn.Bandwidth,
			conn.Reliability,
		)
	}
	
	// Update metrics
	mtm.topology.metrics.LastOptimization = time.Now()
	
	mtm.logger.Info("Topology applied successfully")
	return nil
}

// calculateLatency calculates latency between two nodes
func (mtm *MeshTopologyManager) calculateLatency(node1, node2 *MeshNode) time.Duration {
	// In a real implementation, you would:
	// 1. Use actual network measurements
	// 2. Consider geographical distance
	// 3. Account for network conditions
	
	// For now, we'll use a simple calculation based on geographical distance
	if node1.Location != nil && node2.Location != nil {
		distance := mtm.calculateDistance(node1.Location, node2.Location)
		// Rough estimate: 1ms per 100km
		return time.Duration(distance/100) * time.Millisecond
	}
	
	// Default latency
	return 10 * time.Millisecond
}

// calculateBandwidth calculates bandwidth between two nodes
func (mtm *MeshTopologyManager) calculateBandwidth(node1, node2 *MeshNode) int64 {
	// In a real implementation, you would measure actual bandwidth
	// For now, we'll use a default value
	return 100 * 1024 * 1024 // 100 MB/s
}

// calculateReliability calculates reliability between two nodes
func (mtm *MeshTopologyManager) calculateReliability(node1, node2 *MeshNode) float64 {
	// In a real implementation, you would:
	// 1. Monitor packet loss
	// 2. Track connection stability
	// 3. Consider historical data
	
	// For now, we'll use a default value
	return 0.95
}

// calculateDistance calculates geographical distance between two locations
func (mtm *MeshTopologyManager) calculateDistance(loc1, loc2 *GeoLocation) float64 {
	const R = 6371 // Earth's radius in kilometers
	
	lat1 := loc1.Latitude * math.Pi / 180
	lat2 := loc2.Latitude * math.Pi / 180
	deltaLat := (loc2.Latitude - loc1.Latitude) * math.Pi / 180
	deltaLon := (loc2.Longitude - loc1.Longitude) * math.Pi / 180
	
	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1)*math.Cos(lat2)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	
	return R * c
}

// GetMetrics returns topology metrics
func (mt *MeshTopology) GetMetrics() *TopologyMetrics {
	return mt.metrics
}

// GetRouter returns the mesh router
func (mtm *MeshTopologyManager) GetRouter() *MeshRouter {
	return mtm.router
}

// EdgeHeap implements heap.Interface for sorting edges
type EdgeHeap struct {
	edges []*MeshConnection
}

func (eh EdgeHeap) Len() int { return len(eh.edges) }

func (eh EdgeHeap) Less(i, j int) bool {
	return eh.edges[i].Cost < eh.edges[j].Cost
}

func (eh EdgeHeap) Swap(i, j int) {
	eh.edges[i], eh.edges[j] = eh.edges[j], eh.edges[i]
}

func (eh *EdgeHeap) Push(x interface{}) {
	eh.edges = append(eh.edges, x.(*MeshConnection))
}

func (eh *EdgeHeap) Pop() interface{} {
	old := eh.edges
	n := len(old)
	x := old[n-1]
	eh.edges = old[0 : n-1]
	return x
}

// UnionFind implements union-find data structure
type UnionFind struct {
	parent []int
	rank   []int
}

func NewUnionFind(n int) *UnionFind {
	uf := &UnionFind{
		parent: make([]int, n),
		rank:   make([]int, n),
	}
	for i := 0; i < n; i++ {
		uf.parent[i] = i
	}
	return uf
}

func (uf *UnionFind) Find(x int) int {
	if uf.parent[x] != x {
		uf.parent[x] = uf.Find(uf.parent[x])
	}
	return uf.parent[x]
}

func (uf *UnionFind) Union(x, y int) {
	px, py := uf.Find(x), uf.Find(y)
	if px == py {
		return
	}
	if uf.rank[px] < uf.rank[py] {
		uf.parent[px] = py
	} else if uf.rank[px] > uf.rank[py] {
		uf.parent[py] = px
	} else {
		uf.parent[py] = px
		uf.rank[px]++
	}
}
