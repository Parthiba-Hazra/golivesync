package webrtc

import (
	"encoding/json"
	"log"
	"os"
	"sync"

	"github.com/gofiber/websocket/v2"
	"github.com/pion/webrtc/v3"
)

func CustomStreamConnection(c *websocket.Conn, p *CustomPeerManager) {
	config := getWebRTCConfiguration()
	peerConnection := createPeerConnectionStream(config)
	if peerConnection == nil {
		return
	}
	defer peerConnection.Close()

	newPeer := addPeerConnectionToListStream(peerConnection, c, p)
	defer removePeerConnectionFromListStream(newPeer, p)

	setupPeerConnectionCallbacksStream(peerConnection, newPeer, p)

	p.SignalPeerConnectionHelper()

	handleWebSocketMessages(c, peerConnection)
}

func getWebRTCConfiguration() webrtc.Configuration {
	var config webrtc.Configuration
	if os.Getenv("ENVIRONMENT") == "PRODUCTION" {
		config = turnConfig
	}
	return config
}

func createPeerConnectionStream(config webrtc.Configuration) *webrtc.PeerConnection {
	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		log.Print(err)
		return nil
	}

	for _, typ := range []webrtc.RTPCodecType{webrtc.RTPCodecTypeVideo, webrtc.RTPCodecTypeAudio} {
		if _, err := peerConnection.AddTransceiverFromKind(typ, webrtc.RTPTransceiverInit{
			Direction: webrtc.RTPTransceiverDirectionRecvonly,
		}); err != nil {
			log.Print(err)
			return nil
		}
	}

	return peerConnection
}

func addPeerConnectionToListStream(peerConnection *webrtc.PeerConnection, c *websocket.Conn, p *CustomPeerManager) CustomPeerConnectionState {
	newPeer := CustomPeerConnectionState{
		PeerConnection: peerConnection,
		Websocket: &CustomThreadSafeWriter{
			Conn:  c,
			Mutex: sync.Mutex{},
		},
	}

	p.ListLock.Lock()
	p.Connections = append(p.Connections, newPeer)
	p.ListLock.Unlock()

	log.Println(p.Connections)

	return newPeer
}

func removePeerConnectionFromListStream(newPeer CustomPeerConnectionState, p *CustomPeerManager) {
	p.ListLock.Lock()
	defer p.ListLock.Unlock()

	p.Connections = removeCustomConnection(p.Connections, &newPeer)
}

func setupPeerConnectionCallbacksStream(peerConnection *webrtc.PeerConnection, newPeer CustomPeerConnectionState, p *CustomPeerManager) {
	peerConnection.OnICECandidate(func(i *webrtc.ICECandidate) {
		if i == nil {
			return
		}

		candidateString, err := json.Marshal(i.ToJSON())
		if err != nil {
			log.Println(err)
			return
		}

		if writeErr := newPeer.Websocket.WriteJSON(&CustomWebSocketMessage{
			Event: "custom-candidate",
			Data:  string(candidateString),
		}); writeErr != nil {
			log.Println(writeErr)
		}
	})

	peerConnection.OnConnectionStateChange(func(pp webrtc.PeerConnectionState) {
		handleConnectionStateChangeStream(pp, peerConnection, p)
	})
}

func handleConnectionStateChangeStream(pp webrtc.PeerConnectionState, peerConnection *webrtc.PeerConnection, p *CustomPeerManager) {
	switch pp {
	case webrtc.PeerConnectionStateFailed:
		if err := peerConnection.Close(); err != nil {
			log.Print(err)
		}
	case webrtc.PeerConnectionStateClosed:
		p.SignalPeerConnectionHelper()
	}
}

func handleWebSocketMessages(c *websocket.Conn, peerConnection *webrtc.PeerConnection) {
	message := &CustomWebSocketMessage{}
	for {
		_, raw, err := c.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		} else if err := json.Unmarshal(raw, &message); err != nil {
			log.Println(err)
			return
		}

		switch message.Event {
		case "custom-candidate":
			handleICECandidate(message.Data, peerConnection)
		case "custom-answer":
			handleSessionAnswer(message.Data, peerConnection)
		}
	}
}
