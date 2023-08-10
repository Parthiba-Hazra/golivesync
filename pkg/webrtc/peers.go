package webrtc

import (
	"log"
	"sync"

	"github.com/Parthiba-Hazra/golivesync/pkg/customchat"
	"github.com/gofiber/websocket/v2"
	"github.com/pion/webrtc/v3"
)

var (
	StreamsLock   sync.RWMutex
	CustomRooms   map[string]*CustomRoomManager
	CustomStreams map[string]*CustomRoomManager
)

var (
	turnConfig = webrtc.Configuration{
		ICETransportPolicy: webrtc.ICETransportPolicyRelay,
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:turn.localhost:4000"},
			},
			{
				URLs:           []string{"turn:turn.localhost:4000"},
				Username:       "yourusername",
				Credential:     "yourpassword",
				CredentialType: webrtc.ICECredentialTypePassword,
			},
		},
	}
)

// CustomRoomManager manages WebRTC rooms and peers.
type CustomRoomManager struct {
	Peers *CustomPeerManager    // Manage peer connections
	Hub   *customchat.CustomHub // Manage chat messages
}

// CustomPeerManager manages WebRTC peer connections.
type CustomPeerManager struct {
	ListLock    sync.RWMutex
	Connections []CustomPeerConnectionState // List of peer connections
	TrackLocals map[string]*webrtc.TrackLocalStaticRTP
}

// CustomPeerConnectionState holds the state of a WebRTC peer connection.
type CustomPeerConnectionState struct {
	PeerConnection *webrtc.PeerConnection
	Websocket      *CustomThreadSafeWriter
}

// CustomThreadSafeWriter wraps a websocket connection to provide thread-safe writing.
type CustomThreadSafeWriter struct {
	Conn  *websocket.Conn
	Mutex sync.Mutex
}

func (t *CustomThreadSafeWriter) WriteJSON(v interface{}) error {
	t.Mutex.Lock()
	defer t.Mutex.Unlock()
	return t.Conn.WriteJSON(v)
}

// AddCustomTrack adds a track to the peer connection.
func (p *CustomPeerManager) AddCustomTrack(t *webrtc.TrackRemote) *webrtc.TrackLocalStaticRTP {
	p.ListLock.Lock()
	defer func() {
		p.ListLock.Unlock()
		p.SignalPeerConnectionHelper()
	}()

	TrackLocal, err := webrtc.NewTrackLocalStaticRTP(t.Codec().RTPCodecCapability, t.ID(), t.StreamID())

	if err != nil {
		log.Printf("custom track local error: %v", err.Error())
		return nil
	}
	p.TrackLocals[t.ID()] = TrackLocal
	return TrackLocal
}

// RemoveCustomTrack removes a track from the peer connection.
func (p *CustomPeerManager) RemoveCustomTrack(t *webrtc.TrackLocalStaticRTP) {
	p.ListLock.Lock()
	defer func() {
		p.ListLock.Unlock()
		p.SignalPeerConnectionHelper()
	}()
	delete(p.TrackLocals, t.ID())
}

func (p *CustomPeerManager) SignalPeerConnectionHelper() {
	p.ListLock.Lock()
	defer func() {
		p.ListLock.Unlock()
		p.DispatchCustomKeyFrame()
	}()

	for i := range p.Connections {
		p.handleConnectionSync(&p.Connections[i])
	}
}

func (p *CustomPeerManager) handleConnectionSync(connection *CustomPeerConnectionState) {
	if p.shouldRemoveConnection(connection) {
		p.removeConnection(connection)
		log.Println("Removed closed connection:", connection)
		return
	}

	p.syncConnectionTracks(connection)
}

func (p *CustomPeerManager) shouldRemoveConnection(connection *CustomPeerConnectionState) bool {
	return connection.PeerConnection.ConnectionState() == webrtc.PeerConnectionStateClosed
}

func (p *CustomPeerManager) removeConnection(connection *CustomPeerConnectionState) {
	p.Connections = removeCustomConnection(p.Connections, connection)
}

func (p *CustomPeerManager) syncConnectionTracks(connection *CustomPeerConnectionState) {
	existingSenders := p.collectExistingSenders(connection)

	p.removeUnwantedSenders(connection, existingSenders)
	p.addMissingSenders(connection, existingSenders)
}

func (p *CustomPeerManager) collectExistingSenders(connection *CustomPeerConnectionState) map[string]bool {
	existingSenders := make(map[string]bool)
	for _, senders := range connection.PeerConnection.GetSenders() {
		if senders.Track() != nil {
			existingSenders[senders.Track().ID()] = true
		}
	}
	return existingSenders
}

func (p *CustomPeerManager) removeUnwantedSenders(connection *CustomPeerConnectionState, existingSenders map[string]bool) {
	for _, senders := range connection.PeerConnection.GetSenders() {
		if senders.Track() == nil {
			continue
		}
		if !existingSenders[senders.Track().ID()] {
			p.removeSender(connection, senders)
		}
	}
}

func (p *CustomPeerManager) removeSender(connection *CustomPeerConnectionState, sender *webrtc.RTPSender) {
	if err := connection.PeerConnection.RemoveTrack(sender); err != nil {
		log.Printf("Error removing custom track: %v", err)
	}
}

func (p *CustomPeerManager) addMissingSenders(connection *CustomPeerConnectionState, existingSenders map[string]bool) {
	for trackID := range p.TrackLocals {
		if !existingSenders[trackID] {
			p.addTrack(connection, trackID)
		}
	}
}

func (p *CustomPeerManager) addTrack(connection *CustomPeerConnectionState, trackID string) {
	if _, err := connection.PeerConnection.AddTrack(p.TrackLocals[trackID]); err != nil {
		log.Printf("Error adding custom track: %v", err)
	}
}

// DispatchCustomKeyFrame sends a keyframe to all connected peers.
func (p *CustomPeerManager) DispatchCustomKeyFrame() {
	p.ListLock.RLock()
	defer p.ListLock.RUnlock()

	for i := range p.Connections {
		p.Connections[i].Websocket.WriteJSON(CustomWebSocketMessage{
			Event: "custom-keyframe",
			Data:  "",
		})
	}
}

// removeCustomConnection removes a connection from the connections list.
func removeCustomConnection(connections []CustomPeerConnectionState, connection *CustomPeerConnectionState) []CustomPeerConnectionState {
	for i, conn := range connections {
		if conn == *connection {
			return append(connections[:i], connections[i+1:]...)
		}
	}
	return connections
}

// CustomWebSocketMessage represents a custom WebSocket message structure.
type CustomWebSocketMessage struct {
	Event string `json:"event"`
	Data  string `json:"data"`
}

// NewCustomRoomManager creates a new CustomRoomManager instance.
func NewCustomRoomManager() *CustomRoomManager {
	return &CustomRoomManager{
		Peers: NewCustomPeerManager(),
		Hub:   customchat.NewCustomHub(),
	}
}

// NewCustomPeerManager creates a new CustomPeerManager instance.
func NewCustomPeerManager() *CustomPeerManager {
	return &CustomPeerManager{
		Connections: make([]CustomPeerConnectionState, 0),
		TrackLocals: make(map[string]*webrtc.TrackLocalStaticRTP),
	}
}
