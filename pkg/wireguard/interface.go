package wireguard

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net"
	"sync"
	"time"

	"go.uber.org/zap"
)

// WireGuardInterface represents a WireGuard network interface
type WireGuardInterface struct {
	name        string
	privateKey  *[32]byte
	publicKey   *[32]byte
	listenPort  int
	mtu         int
	peers       map[string]*Peer
	peersMutex  sync.RWMutex
	routes      map[string]*Route
	routesMutex sync.RWMutex
	logger      *zap.Logger
	metrics     *WireGuardMetrics
	status      InterfaceStatus
}

// InterfaceStatus represents the status of a WireGuard interface
type InterfaceStatus string

const (
	InterfaceStatusDown   InterfaceStatus = "down"
	InterfaceStatusUp     InterfaceStatus = "up"
	InterfaceStatusError  InterfaceStatus = "error"
)

// Peer represents a WireGuard peer
type Peer struct {
	PublicKey           *[32]byte
	AllowedIPs          []net.IPNet
	Endpoint            *net.UDPAddr
	PersistentKeepalive time.Duration
	LastHandshake       time.Time
	RxBytes             int64
	TxBytes             int64
	Status              PeerStatus
	LastSeen            time.Time
}

// PeerStatus represents the status of a peer
type PeerStatus string

const (
	PeerStatusOffline   PeerStatus = "offline"
	PeerStatusOnline    PeerStatus = "online"
	PeerStatusConnecting PeerStatus = "connecting"
)

// Route represents a network route
type Route struct {
	Destination net.IPNet
	Gateway     net.IP
	Interface   string
	Metric      int
}

// WireGuardMetrics represents metrics for WireGuard interface
type WireGuardMetrics struct {
	TotalPeers       int64
	OnlinePeers      int64
	TotalRxBytes     int64
	TotalTxBytes     int64
	LastHandshake    time.Time
	InterfaceUpTime  time.Duration
}

// NewWireGuardInterface creates a new WireGuard interface
func NewWireGuardInterface(name string, listenPort int, mtu int, logger *zap.Logger) (*WireGuardInterface, error) {
	// Generate private key
	privateKey := new([32]byte)
	if _, err := rand.Read(privateKey[:]); err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	// Generate public key from private key
	publicKey := new([32]byte)
	// In a real implementation, you would use WireGuard's key generation
	// For now, we'll use a simple XOR operation as placeholder
	for i := 0; i < 32; i++ {
		publicKey[i] = privateKey[i] ^ 0x42
	}

	return &WireGuardInterface{
		name:       name,
		privateKey: privateKey,
		publicKey:  publicKey,
		listenPort: listenPort,
		mtu:        mtu,
		peers:      make(map[string]*Peer),
		routes:     make(map[string]*Route),
		logger:     logger,
		metrics:    &WireGuardMetrics{},
		status:     InterfaceStatusDown,
	}, nil
}

// Start initializes and starts the WireGuard interface
func (wgi *WireGuardInterface) Start() error {
	wgi.logger.Info("Starting WireGuard interface", 
		zap.String("name", wgi.name),
		zap.Int("port", wgi.listenPort))

	// In a real implementation, you would:
	// 1. Create the WireGuard interface using wgctrl
	// 2. Configure the interface with private key and listen port
	// 3. Set up the interface in the kernel
	// 4. Start listening for incoming connections

	wgi.status = InterfaceStatusUp
	wgi.metrics.InterfaceUpTime = time.Since(time.Now())

	wgi.logger.Info("WireGuard interface started successfully")
	return nil
}

// Stop stops the WireGuard interface
func (wgi *WireGuardInterface) Stop() error {
	wgi.logger.Info("Stopping WireGuard interface", zap.String("name", wgi.name))

	// In a real implementation, you would:
	// 1. Stop listening for connections
	// 2. Remove all peers
	// 3. Bring down the interface
	// 4. Clean up kernel resources

	wgi.status = InterfaceStatusDown
	wgi.logger.Info("WireGuard interface stopped")
	return nil
}

// AddPeer adds a new peer to the WireGuard interface
func (wgi *WireGuardInterface) AddPeer(publicKey *[32]byte, allowedIPs []net.IPNet, endpoint *net.UDPAddr) error {
	wgi.peersMutex.Lock()
	defer wgi.peersMutex.Unlock()

	peerKey := base64.StdEncoding.EncodeToString(publicKey[:])
	
	peer := &Peer{
		PublicKey:           publicKey,
		AllowedIPs:          allowedIPs,
		Endpoint:            endpoint,
		PersistentKeepalive: 25 * time.Second,
		Status:              PeerStatusOffline,
		LastSeen:            time.Now(),
	}

	wgi.peers[peerKey] = peer
	wgi.metrics.TotalPeers++

	wgi.logger.Info("Added peer to WireGuard interface",
		zap.String("peer", peerKey),
		zap.String("endpoint", endpoint.String()))

	return nil
}

// RemovePeer removes a peer from the WireGuard interface
func (wgi *WireGuardInterface) RemovePeer(publicKey *[32]byte) error {
	wgi.peersMutex.Lock()
	defer wgi.peersMutex.Unlock()

	peerKey := base64.StdEncoding.EncodeToString(publicKey[:])
	
	if peer, exists := wgi.peers[peerKey]; exists {
		if peer.Status == PeerStatusOnline {
			wgi.metrics.OnlinePeers--
		}
		delete(wgi.peers, peerKey)
		wgi.metrics.TotalPeers--

		wgi.logger.Info("Removed peer from WireGuard interface", zap.String("peer", peerKey))
	}

	return nil
}

// GetPeer returns a peer by public key
func (wgi *WireGuardInterface) GetPeer(publicKey *[32]byte) (*Peer, bool) {
	wgi.peersMutex.RLock()
	defer wgi.peersMutex.RUnlock()

	peerKey := base64.StdEncoding.EncodeToString(publicKey[:])
	peer, exists := wgi.peers[peerKey]
	return peer, exists
}

// GetAllPeers returns all peers
func (wgi *WireGuardInterface) GetAllPeers() []*Peer {
	wgi.peersMutex.RLock()
	defer wgi.peersMutex.RUnlock()

	peers := make([]*Peer, 0, len(wgi.peers))
	for _, peer := range wgi.peers {
		peers = append(peers, peer)
	}
	return peers
}

// UpdatePeerStatus updates the status of a peer
func (wgi *WireGuardInterface) UpdatePeerStatus(publicKey *[32]byte, status PeerStatus) {
	wgi.peersMutex.Lock()
	defer wgi.peersMutex.Unlock()

	peerKey := base64.StdEncoding.EncodeToString(publicKey[:])
	if peer, exists := wgi.peers[peerKey]; exists {
		oldStatus := peer.Status
		peer.Status = status
		peer.LastSeen = time.Now()

		// Update metrics
		if oldStatus != PeerStatusOnline && status == PeerStatusOnline {
			wgi.metrics.OnlinePeers++
		} else if oldStatus == PeerStatusOnline && status != PeerStatusOnline {
			wgi.metrics.OnlinePeers--
		}

		wgi.logger.Debug("Updated peer status",
			zap.String("peer", peerKey),
			zap.String("status", string(status)))
	}
}

// GetPublicKey returns the public key of the interface
func (wgi *WireGuardInterface) GetPublicKey() *[32]byte {
	return wgi.publicKey
}

// GetPrivateKey returns the private key of the interface
func (wgi *WireGuardInterface) GetPrivateKey() *[32]byte {
	return wgi.privateKey
}

// GetStatus returns the current status of the interface
func (wgi *WireGuardInterface) GetStatus() InterfaceStatus {
	return wgi.status
}

// GetMetrics returns the current metrics
func (wgi *WireGuardInterface) GetMetrics() *WireGuardMetrics {
	return wgi.metrics
}

// GetName returns the interface name
func (wgi *WireGuardInterface) GetName() string {
	return wgi.name
}
