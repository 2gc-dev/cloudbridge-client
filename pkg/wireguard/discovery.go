package wireguard

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"go.uber.org/zap"
)

// PeerDiscovery represents a peer discovery service
type PeerDiscovery struct {
	localNode    *MeshNode
	knownPeers   map[string]*Peer
	peersMutex   sync.RWMutex
	discoveryCh  chan *Peer
	announceCh   chan *Announcement
	stopCh       chan struct{}
	logger       *zap.Logger
	metrics      *DiscoveryMetrics
	config       *DiscoveryConfig
}

// MeshNode represents a node in the mesh network
type MeshNode struct {
	ID          string
	PublicKey   *[32]byte
	Endpoint    *net.UDPAddr
	Location    *GeoLocation
	Capabilities []string
	Status      NodeStatus
	LastSeen    time.Time
	Version     string
}

// NodeStatus represents the status of a mesh node
type NodeStatus string

const (
	NodeStatusOffline   NodeStatus = "offline"
	NodeStatusOnline    NodeStatus = "online"
	NodeStatusConnecting NodeStatus = "connecting"
)

// GeoLocation represents geographical location
type GeoLocation struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Country   string  `json:"country"`
	City      string  `json:"city"`
	Region    string  `json:"region"`
}

// Announcement represents a peer announcement message
type Announcement struct {
	NodeID      string       `json:"node_id"`
	PublicKey   string       `json:"public_key"`
	Endpoint    string       `json:"endpoint"`
	Location    *GeoLocation `json:"location"`
	Capabilities []string    `json:"capabilities"`
	Version     string       `json:"version"`
	Timestamp   time.Time    `json:"timestamp"`
}

// DiscoveryMetrics represents metrics for peer discovery
type DiscoveryMetrics struct {
	TotalAnnouncements int64
	ActivePeers        int64
	DiscoveryLatency   time.Duration
	LastDiscovery      time.Time
}

// DiscoveryConfig represents configuration for peer discovery
type DiscoveryConfig struct {
	AnnounceInterval    time.Duration
	DiscoveryPort       int
	AnnouncementTimeout time.Duration
	MaxPeers            int
	EnableGeoDiscovery  bool
}

// NewPeerDiscovery creates a new peer discovery service
func NewPeerDiscovery(localNode *MeshNode, config *DiscoveryConfig, logger *zap.Logger) *PeerDiscovery {
	if config == nil {
		config = &DiscoveryConfig{
			AnnounceInterval:    30 * time.Second,
			DiscoveryPort:       51821,
			AnnouncementTimeout: 5 * time.Minute,
			MaxPeers:           100,
			EnableGeoDiscovery: true,
		}
	}

	return &PeerDiscovery{
		localNode:   localNode,
		knownPeers:  make(map[string]*Peer),
		discoveryCh: make(chan *Peer, 100),
		announceCh:  make(chan *Announcement, 100),
		stopCh:      make(chan struct{}),
		logger:      logger,
		metrics:     &DiscoveryMetrics{},
		config:      config,
	}
}

// Start starts the peer discovery service
func (pd *PeerDiscovery) Start() error {
	pd.logger.Info("Starting peer discovery service",
		zap.String("node_id", pd.localNode.ID),
		zap.Int("port", pd.config.DiscoveryPort))

	// Start UDP server for listening to announcements
	go pd.listenForAnnouncements()

	// Start periodic announcement of presence
	go pd.announcePresence()

	// Start processing announcements
	go pd.processAnnouncements()

	// Start cleanup of stale peers
	go pd.cleanupStalePeers()

	pd.logger.Info("Peer discovery service started successfully")
	return nil
}

// Stop stops the peer discovery service
func (pd *PeerDiscovery) Stop() error {
	pd.logger.Info("Stopping peer discovery service")
	close(pd.stopCh)
	return nil
}

// listenForAnnouncements listens for peer announcements on UDP
func (pd *PeerDiscovery) listenForAnnouncements() {
	addr := &net.UDPAddr{Port: pd.config.DiscoveryPort}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		pd.logger.Error("Failed to listen for announcements", zap.Error(err))
		return
	}
	defer conn.Close()

	pd.logger.Info("Listening for peer announcements", zap.String("address", addr.String()))

	buffer := make([]byte, 2048)
	for {
		select {
		case <-pd.stopCh:
			return
		default:
			conn.SetReadDeadline(time.Now().Add(1 * time.Second))
			n, remoteAddr, err := conn.ReadFromUDP(buffer)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				pd.logger.Error("Error reading from UDP", zap.Error(err))
				continue
			}

			// Process announcement
			go pd.handleAnnouncement(buffer[:n], remoteAddr)
		}
	}
}

// handleAnnouncement processes an incoming announcement
func (pd *PeerDiscovery) handleAnnouncement(data []byte, remoteAddr *net.UDPAddr) {
	var announcement Announcement
	if err := json.Unmarshal(data, &announcement); err != nil {
		pd.logger.Error("Failed to unmarshal announcement", zap.Error(err))
		return
	}

	// Ignore our own announcements
	if announcement.NodeID == pd.localNode.ID {
		return
	}

	// Validate announcement
	if err := pd.validateAnnouncement(&announcement); err != nil {
		pd.logger.Error("Invalid announcement", zap.Error(err))
		return
	}

	// Send to processing channel
	select {
	case pd.announceCh <- &announcement:
	default:
		pd.logger.Warn("Announcement channel full, dropping announcement")
	}
}

// validateAnnouncement validates an announcement message
func (pd *PeerDiscovery) validateAnnouncement(announcement *Announcement) error {
	if announcement.NodeID == "" {
		return fmt.Errorf("empty node ID")
	}
	if announcement.PublicKey == "" {
		return fmt.Errorf("empty public key")
	}
	if announcement.Endpoint == "" {
		return fmt.Errorf("empty endpoint")
	}
	if time.Since(announcement.Timestamp) > pd.config.AnnouncementTimeout {
		return fmt.Errorf("announcement too old")
	}
	return nil
}

// announcePresence periodically announces our presence to the network
func (pd *PeerDiscovery) announcePresence() {
	ticker := time.NewTicker(pd.config.AnnounceInterval)
	defer ticker.Stop()

	for {
		select {
		case <-pd.stopCh:
			return
		case <-ticker.C:
			if err := pd.sendAnnouncement(); err != nil {
				pd.logger.Error("Failed to send announcement", zap.Error(err))
			}
		}
	}
}

// sendAnnouncement sends an announcement to the network
func (pd *PeerDiscovery) sendAnnouncement() error {
	announcement := &Announcement{
		NodeID:      pd.localNode.ID,
		PublicKey:   fmt.Sprintf("%x", pd.localNode.PublicKey[:]),
		Endpoint:    pd.localNode.Endpoint.String(),
		Location:    pd.localNode.Location,
		Capabilities: pd.localNode.Capabilities,
		Version:     pd.localNode.Version,
		Timestamp:   time.Now(),
	}

	data, err := json.Marshal(announcement)
	if err != nil {
		return fmt.Errorf("failed to marshal announcement: %w", err)
	}

	// Send to broadcast address
	broadcastAddr := &net.UDPAddr{
		IP:   net.IPv4(255, 255, 255, 255),
		Port: pd.config.DiscoveryPort,
	}

	conn, err := net.DialUDP("udp", nil, broadcastAddr)
	if err != nil {
		return fmt.Errorf("failed to create UDP connection: %w", err)
	}
	defer conn.Close()

	if _, err := conn.Write(data); err != nil {
		return fmt.Errorf("failed to send announcement: %w", err)
	}

	pd.metrics.TotalAnnouncements++
	pd.logger.Debug("Sent announcement", zap.String("node_id", pd.localNode.ID))
	return nil
}

// processAnnouncements processes incoming announcements
func (pd *PeerDiscovery) processAnnouncements() {
	for {
		select {
		case <-pd.stopCh:
			return
		case announcement := <-pd.announceCh:
			pd.handleProcessedAnnouncement(announcement)
		}
	}
}

// handleProcessedAnnouncement handles a processed announcement
func (pd *PeerDiscovery) handleProcessedAnnouncement(announcement *Announcement) {
	pd.peersMutex.Lock()
	defer pd.peersMutex.Unlock()

	// Check if we already know this peer
	if _, exists := pd.knownPeers[announcement.NodeID]; exists {
		// Update existing peer
		pd.updateExistingPeer(announcement)
	} else {
		// Add new peer
		pd.addNewPeer(announcement)
	}

	pd.metrics.LastDiscovery = time.Now()
}

// addNewPeer adds a new peer from announcement
func (pd *PeerDiscovery) addNewPeer(announcement *Announcement) {
	// Check if we've reached the maximum number of peers
	if len(pd.knownPeers) >= pd.config.MaxPeers {
		pd.logger.Warn("Maximum number of peers reached, dropping new peer",
			zap.String("node_id", announcement.NodeID))
		return
	}

	// Parse public key
	publicKeyBytes := []byte(announcement.PublicKey)
	if len(publicKeyBytes) != 32 {
		pd.logger.Error("Invalid public key length", 
			zap.String("node_id", announcement.NodeID),
			zap.Int("length", len(publicKeyBytes)))
		return
	}

	publicKey := new([32]byte)
	copy(publicKey[:], publicKeyBytes)

	// Parse endpoint
	endpoint, err := net.ResolveUDPAddr("udp", announcement.Endpoint)
	if err != nil {
		pd.logger.Error("Failed to resolve endpoint",
			zap.String("node_id", announcement.NodeID),
			zap.String("endpoint", announcement.Endpoint),
			zap.Error(err))
		return
	}

	// Create peer
	peer := &Peer{
		PublicKey: publicKey,
		Endpoint:  endpoint,
		Status:    PeerStatusOffline,
		LastSeen:  announcement.Timestamp,
	}

	pd.knownPeers[announcement.NodeID] = peer
	pd.metrics.ActivePeers++

	pd.logger.Info("Added new peer",
		zap.String("node_id", announcement.NodeID),
		zap.String("endpoint", announcement.Endpoint))

	// Send to discovery channel
	select {
	case pd.discoveryCh <- peer:
	default:
		pd.logger.Warn("Discovery channel full, dropping peer")
	}
}

// updateExistingPeer updates an existing peer
func (pd *PeerDiscovery) updateExistingPeer(announcement *Announcement) {
	peer := pd.knownPeers[announcement.NodeID]
	peer.LastSeen = announcement.Timestamp

	// Update endpoint if changed
	if announcement.Endpoint != peer.Endpoint.String() {
		if endpoint, err := net.ResolveUDPAddr("udp", announcement.Endpoint); err == nil {
			peer.Endpoint = endpoint
			pd.logger.Debug("Updated peer endpoint",
				zap.String("node_id", announcement.NodeID),
				zap.String("endpoint", announcement.Endpoint))
		}
	}
}

// cleanupStalePeers removes peers that haven't been seen recently
func (pd *PeerDiscovery) cleanupStalePeers() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-pd.stopCh:
			return
		case <-ticker.C:
			pd.peersMutex.Lock()
			
			now := time.Now()
			for nodeID, peer := range pd.knownPeers {
				if now.Sub(peer.LastSeen) > pd.config.AnnouncementTimeout {
					delete(pd.knownPeers, nodeID)
					pd.metrics.ActivePeers--
					
					pd.logger.Info("Removed stale peer",
						zap.String("node_id", nodeID),
						zap.Duration("last_seen", now.Sub(peer.LastSeen)))
				}
			}
			
			pd.peersMutex.Unlock()
		}
	}
}

// GetDiscoveredPeers returns all discovered peers
func (pd *PeerDiscovery) GetDiscoveredPeers() []*Peer {
	pd.peersMutex.RLock()
	defer pd.peersMutex.RUnlock()

	peers := make([]*Peer, 0, len(pd.knownPeers))
	for _, peer := range pd.knownPeers {
		peers = append(peers, peer)
	}
	return peers
}

// GetDiscoveryChannel returns the discovery channel
func (pd *PeerDiscovery) GetDiscoveryChannel() <-chan *Peer {
	return pd.discoveryCh
}

// GetMetrics returns discovery metrics
func (pd *PeerDiscovery) GetMetrics() *DiscoveryMetrics {
	return pd.metrics
}
