package wireguard

import (
	"container/heap"
	"fmt"
	"math"
	"sync"
	"time"

	"go.uber.org/zap"
)

// MeshRouter represents a router for the mesh network
type MeshRouter struct {
	topology    *MeshTopology
	logger      *zap.Logger
	metrics     *RouterMetrics
	routesCache map[string]*CachedRoute
	cacheMutex  sync.RWMutex
	config      *RouterConfig
}

// RouterMetrics represents metrics for the mesh router
type RouterMetrics struct {
	TotalRoutesCalculated int64
	CacheHits             int64
	CacheMisses           int64
	AverageRouteLatency   time.Duration
	RoutingErrors         int64
	LastRouteCalculation  time.Time
}

// CachedRoute represents a cached route
type CachedRoute struct {
	Route      *MeshRoute
	ExpiresAt  time.Time
	AccessCount int64
}

// RouterConfig represents configuration for the mesh router
type RouterConfig struct {
	CacheTTL              time.Duration
	MaxCacheSize          int
	EnableLoadBalancing   bool
	EnableFailover        bool
	MaxRouteHops          int
	RouteCalculationTimeout time.Duration
}

// NewMeshRouter creates a new mesh router
func NewMeshRouter(topology *MeshTopology, logger *zap.Logger) *MeshRouter {
	return &MeshRouter{
		topology:    topology,
		logger:      logger,
		metrics:     &RouterMetrics{},
		routesCache: make(map[string]*CachedRoute),
		config: &RouterConfig{
			CacheTTL:                5 * time.Minute,
			MaxCacheSize:           1000,
			EnableLoadBalancing:    true,
			EnableFailover:         true,
			MaxRouteHops:           10,
			RouteCalculationTimeout: 10 * time.Second,
		},
	}
}

// FindRoute finds the best route between two nodes
func (mr *MeshRouter) FindRoute(source, destination string) (*MeshRoute, error) {
	// Check cache first
	if route := mr.getCachedRoute(source, destination); route != nil {
		mr.metrics.CacheHits++
		return route, nil
	}

	mr.metrics.CacheMisses++

	// Calculate new route
	route, err := mr.calculateRoute(source, destination)
	if err != nil {
		mr.metrics.RoutingErrors++
		return nil, fmt.Errorf("failed to calculate route: %w", err)
	}

	// Cache the route
	mr.cacheRoute(source, destination, route)

	mr.metrics.TotalRoutesCalculated++
	mr.metrics.LastRouteCalculation = time.Now()

	return route, nil
}

// calculateRoute calculates the optimal route between two nodes
func (mr *MeshRouter) calculateRoute(source, destination string) (*MeshRoute, error) {
	// Use Dijkstra's algorithm to find shortest path
	distances := make(map[string]float64)
	previous := make(map[string]string)
	visited := make(map[string]bool)

	// Initialize distances
	nodes := mr.topology.GetAllNodes()
	for _, node := range nodes {
		distances[node.ID] = math.Inf(1)
	}
	distances[source] = 0

	// Priority queue for unvisited nodes
	pq := &NodePriorityQueue{}
	heap.Init(pq)
	heap.Push(pq, &NodeDistance{ID: source, Distance: 0})

	for pq.Len() > 0 {
		current := heap.Pop(pq).(*NodeDistance)
		
		if visited[current.ID] {
			continue
		}
		visited[current.ID] = true

		if current.ID == destination {
			break
		}

		// Check all connections from current node
		connections := mr.getNodeConnections(current.ID)
		for _, conn := range connections {
			neighbor := conn.TargetNode
			if conn.TargetNode == current.ID {
				neighbor = conn.SourceNode
			}

			if visited[neighbor] {
				continue
			}

			newDistance := distances[current.ID] + conn.Cost
			if newDistance < distances[neighbor] {
				distances[neighbor] = newDistance
				previous[neighbor] = current.ID
				heap.Push(pq, &NodeDistance{ID: neighbor, Distance: newDistance})
			}
		}
	}

	// Reconstruct path
	if distances[destination] == math.Inf(1) {
		return nil, fmt.Errorf("no route found from %s to %s", source, destination)
	}

	path := mr.reconstructPath(previous, source, destination)
	
	// Calculate route metrics
	latency, bandwidth, reliability := mr.calculateRouteMetrics(path)

	route := &MeshRoute{
		ID:          fmt.Sprintf("%s-%s", source, destination),
		Source:      source,
		Destination: destination,
		Path:        path,
		Latency:     latency,
		Bandwidth:   bandwidth,
		Reliability: reliability,
		Cost:        distances[destination],
		LastUpdated: time.Now(),
	}

	return route, nil
}

// getNodeConnections returns all connections for a given node
func (mr *MeshRouter) getNodeConnections(nodeID string) []*MeshConnection {
	connections := mr.topology.GetAllConnections()
	var nodeConnections []*MeshConnection

	for _, conn := range connections {
		if conn.SourceNode == nodeID || conn.TargetNode == nodeID {
			nodeConnections = append(nodeConnections, conn)
		}
	}

	return nodeConnections
}

// reconstructPath reconstructs the path from the previous map
func (mr *MeshRouter) reconstructPath(previous map[string]string, source, destination string) []string {
	var path []string
	current := destination

	for current != source {
		path = append([]string{current}, path...)
		current = previous[current]
	}
	path = append([]string{source}, path...)

	return path
}

// calculateRouteMetrics calculates metrics for a route
func (mr *MeshRouter) calculateRouteMetrics(path []string) (time.Duration, int64, float64) {
	if len(path) < 2 {
		return 0, 0, 0
	}

	var totalLatency time.Duration
	var minBandwidth int64 = math.MaxInt64
	var totalReliability float64
	var connectionCount int

	for i := 0; i < len(path)-1; i++ {
		source := path[i]
		target := path[i+1]

		connID := fmt.Sprintf("%s-%s", source, target)
		if conn, exists := mr.topology.GetConnection(connID); exists {
			totalLatency += conn.Latency
			if conn.Bandwidth < minBandwidth {
				minBandwidth = conn.Bandwidth
			}
			totalReliability += conn.Reliability
			connectionCount++
		} else {
			// Try reverse connection
			connID = fmt.Sprintf("%s-%s", target, source)
			if conn, exists := mr.topology.GetConnection(connID); exists {
				totalLatency += conn.Latency
				if conn.Bandwidth < minBandwidth {
					minBandwidth = conn.Bandwidth
				}
				totalReliability += conn.Reliability
				connectionCount++
			}
		}
	}

	if connectionCount == 0 {
		return 0, 0, 0
	}

	averageReliability := totalReliability / float64(connectionCount)
	if minBandwidth == math.MaxInt64 {
		minBandwidth = 0
	}

	return totalLatency, minBandwidth, averageReliability
}

// getCachedRoute retrieves a cached route
func (mr *MeshRouter) getCachedRoute(source, destination string) *MeshRoute {
	mr.cacheMutex.RLock()
	defer mr.cacheMutex.RUnlock()

	cacheKey := fmt.Sprintf("%s-%s", source, destination)
	if cached, exists := mr.routesCache[cacheKey]; exists {
		if time.Now().Before(cached.ExpiresAt) {
			cached.AccessCount++
			return cached.Route
		} else {
			// Remove expired cache entry
			delete(mr.routesCache, cacheKey)
		}
	}

	return nil
}

// cacheRoute caches a route
func (mr *MeshRouter) cacheRoute(source, destination string, route *MeshRoute) {
	mr.cacheMutex.Lock()
	defer mr.cacheMutex.Unlock()

	// Check cache size limit
	if len(mr.routesCache) >= mr.config.MaxCacheSize {
		mr.evictOldestCacheEntry()
	}

	cacheKey := fmt.Sprintf("%s-%s", source, destination)
	mr.routesCache[cacheKey] = &CachedRoute{
		Route:      route,
		ExpiresAt:  time.Now().Add(mr.config.CacheTTL),
		AccessCount: 1,
	}
}

// evictOldestCacheEntry removes the oldest cache entry
func (mr *MeshRouter) evictOldestCacheEntry() {
	var oldestKey string
	var oldestTime time.Time

	for key, cached := range mr.routesCache {
		if oldestKey == "" || cached.ExpiresAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = cached.ExpiresAt
		}
	}

	if oldestKey != "" {
		delete(mr.routesCache, oldestKey)
	}
}

// FindAlternativeRoutes finds alternative routes between two nodes
func (mr *MeshRouter) FindAlternativeRoutes(source, destination string, count int) ([]*MeshRoute, error) {
	var routes []*MeshRoute

	// Get primary route
	primaryRoute, err := mr.FindRoute(source, destination)
	if err != nil {
		return nil, err
	}
	routes = append(routes, primaryRoute)

	// Find alternative routes by temporarily removing edges
	connections := mr.topology.GetAllConnections()
	for i := 0; i < len(connections) && len(routes) < count; i++ {
		conn := connections[i]
		
		// Temporarily remove connection
		mr.topology.RemoveConnection(conn.ID)
		
		// Try to find alternative route
		if altRoute, err := mr.calculateRoute(source, destination); err == nil {
			// Check if this is a different route
			if !mr.isSameRoute(primaryRoute, altRoute) {
				routes = append(routes, altRoute)
			}
		}
		
		// Restore connection
		mr.topology.AddConnection(
			conn.SourceNode,
			conn.TargetNode,
			conn.Latency,
			conn.Bandwidth,
			conn.Reliability,
		)
	}

	return routes, nil
}

// isSameRoute checks if two routes are the same
func (mr *MeshRouter) isSameRoute(route1, route2 *MeshRoute) bool {
	if len(route1.Path) != len(route2.Path) {
		return false
	}

	for i, node := range route1.Path {
		if route2.Path[i] != node {
			return false
		}
	}

	return true
}

// UpdateRoute updates a route in the cache
func (mr *MeshRouter) UpdateRoute(route *MeshRoute) {
	mr.cacheMutex.Lock()
	defer mr.cacheMutex.Unlock()

	cacheKey := fmt.Sprintf("%s-%s", route.Source, route.Destination)
	if cached, exists := mr.routesCache[cacheKey]; exists {
		cached.Route = route
		cached.ExpiresAt = time.Now().Add(mr.config.CacheTTL)
	}
}

// ClearCache clears the route cache
func (mr *MeshRouter) ClearCache() {
	mr.cacheMutex.Lock()
	defer mr.cacheMutex.Unlock()

	mr.routesCache = make(map[string]*CachedRoute)
	mr.logger.Info("Route cache cleared")
}

// GetMetrics returns router metrics
func (mr *MeshRouter) GetMetrics() *RouterMetrics {
	return mr.metrics
}

// NodeDistance represents a node with its distance for priority queue
type NodeDistance struct {
	ID       string
	Distance float64
}

// NodePriorityQueue implements heap.Interface for node distances
type NodePriorityQueue []*NodeDistance

func (pq NodePriorityQueue) Len() int { return len(pq) }

func (pq NodePriorityQueue) Less(i, j int) bool {
	return pq[i].Distance < pq[j].Distance
}

func (pq NodePriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
}

func (pq *NodePriorityQueue) Push(x interface{}) {
	*pq = append(*pq, x.(*NodeDistance))
}

func (pq *NodePriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	x := old[n-1]
	*pq = old[0 : n-1]
	return x
}
