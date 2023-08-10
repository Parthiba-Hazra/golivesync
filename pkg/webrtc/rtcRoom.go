package webrtc

import (
	"encoding/json"
	"log"
	"os"
	"sync"

	"github.com/gofiber/websocket/v2"
	"github.com/pion/webrtc/v3"
)

func CustomRoomConnection(c *websocket.Conn, p *CustomPeerManager) {
	var config webrtc.Configuration
	if os.Getenv("ENVIRONMENT") == "PRODUCTION" {
		config = turnConfig
	}

	peerConnection := createPeerConnection(config, c, p)
	if peerConnection == nil {
		return
	}
	defer peerConnection.Close()

	newPeer := addPeerConnectionToList(peerConnection, c, p)
	defer removePeerConnectionFromList(newPeer, p)

	setupPeerConnectionCallbacks(peerConnection, newPeer, p) // Fix the argument count here
	p.SignalPeerConnectionHelper()

	handleIncomingData(c, peerConnection, p)
}

func createPeerConnection(config webrtc.Configuration, c *websocket.Conn, p *CustomPeerManager) *webrtc.PeerConnection {
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

func addPeerConnectionToList(peerConnection *webrtc.PeerConnection, c *websocket.Conn, p *CustomPeerManager) CustomPeerConnectionState {
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

func removePeerConnectionFromList(newPeer CustomPeerConnectionState, p *CustomPeerManager) {
	p.ListLock.Lock()
	defer p.ListLock.Unlock()

	p.Connections = removeCustomConnection(p.Connections, &newPeer)
}

func setupPeerConnectionCallbacks(peerConnection *webrtc.PeerConnection, newPeer CustomPeerConnectionState, p *CustomPeerManager) {
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
		handleConnectionStateChange(pp, peerConnection, p)
	})

	peerConnection.OnTrack(func(t *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		handleIncomingTrack(t, newPeer, p)
	})
}

func handleConnectionStateChange(pp webrtc.PeerConnectionState, peerConnection *webrtc.PeerConnection, p *CustomPeerManager) {
	switch pp {
	case webrtc.PeerConnectionStateFailed:
		if err := peerConnection.Close(); err != nil {
			log.Print(err)
		}
	case webrtc.PeerConnectionStateClosed:
		p.SignalPeerConnectionHelper()
	}
}

func handleIncomingData(c *websocket.Conn, peerConnection *webrtc.PeerConnection, p *CustomPeerManager) {
	for {
		_, raw, err := c.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}

		message := &CustomWebSocketMessage{}
		if err := json.Unmarshal(raw, &message); err != nil {
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

func handleICECandidate(candidateData string, peerConnection *webrtc.PeerConnection) {
	candidate := webrtc.ICECandidateInit{}
	if err := json.Unmarshal([]byte(candidateData), &candidate); err != nil {
		log.Println(err)
		return
	}

	if err := peerConnection.AddICECandidate(candidate); err != nil {
		log.Println(err)
	}
}

func handleSessionAnswer(answerData string, peerConnection *webrtc.PeerConnection) {
	answer := webrtc.SessionDescription{}
	if err := json.Unmarshal([]byte(answerData), &answer); err != nil {
		log.Println(err)
		return
	}

	if err := peerConnection.SetRemoteDescription(answer); err != nil {
		log.Println(err)
	}
}

func handleIncomingTrack(t *webrtc.TrackRemote, newPeer CustomPeerConnectionState, p *CustomPeerManager) {
	customTrackLocal := p.AddCustomTrack(t)
	if customTrackLocal == nil {
		return
	}
	defer p.RemoveCustomTrack(customTrackLocal)

	buf := make([]byte, 1500)
	for {
		i, _, err := t.Read(buf)
		if err != nil {
			return
		}

		if _, err = customTrackLocal.Write(buf[:i]); err != nil {
			return
		}
	}
}
