package handlers

import (
	"crypto/sha256"
	"fmt"
	"os"
	"time"

	"github.com/Parthiba-Hazra/golivesync/pkg/customchat"
	"github.com/Parthiba-Hazra/golivesync/pkg/webrtc"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	gguid "github.com/google/uuid"
	w "github.com/pion/webrtc/v3"
)

// GenerateNewRoomUUID generates a new room UUID and redirects to the room.
func GenerateNewRoomUUID(c *fiber.Ctx) error {
	uuid := gguid.New().String()
	return c.Redirect(fmt.Sprintf("/room/%s", uuid))
}

// HandleRoomWebsocket handles WebSocket connections for the room.
func HandleRoomWebsocket(c *websocket.Conn) {
	handleWebsocket(c, "room")
}

// HandleRoomViewerWebsocket handles WebSocket connections for room viewers.
func HandleRoomViewerWebsocket(conn *websocket.Conn) {
	handleWebsocket(conn, "viewer")
}

func handleWebsocket(c *websocket.Conn, streamType string) {
	uuid := c.Params("uuid")
	if uuid == "" {
		return
	}

	webrtc.StreamsLock.Lock()
	defer webrtc.StreamsLock.Unlock()

	var peer *webrtc.CustomRoomManager
	if streamType == "room" {
		peer, _ = getOrCreateRoom(uuid)
	} else if streamType == "viewer" {
		peer = webrtc.CustomRooms[uuid]
	}

	if peer != nil {
		webrtc.CustomRoomConnection(c, peer.Peers)
	}
}

// ServeRoom serves the room view.
func ServeRoom(c *fiber.Ctx) error {
	uuid := c.Params("uuid")
	if uuid == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Bad Request")
	}

	wsScheme := "ws"
	if os.Getenv("ENVIRONMENT") == "PRODUCTION" {
		wsScheme = "wss"
	}

	uuid, suuid, _ := CreateOrRetrieveRoom(uuid)
	// (Uncomment and complete this line as needed)

	return c.Render("room", generateRoomRenderData(c, uuid, suuid, wsScheme))
}

func generateRoomRenderData(c *fiber.Ctx, uuid, suuid, wsScheme string) fiber.Map {
	return fiber.Map{
		"RoomWebSocketAddr":   fmt.Sprintf("%s://%s/room/%s/websocket", wsScheme, c.Hostname(), uuid),
		"RoomLink":            fmt.Sprintf("%s://%s/room/%s", c.Protocol(), c.Hostname(), uuid),
		"ChatWebSocketAddr":   fmt.Sprintf("%s://%s/room/%s/chat/websocket", wsScheme, c.Hostname(), uuid),
		"ViewerWebSocketAddr": fmt.Sprintf("%s://%s/room/%s/viewer/websocket", wsScheme, c.Hostname(), uuid),
		"StreamLink":          fmt.Sprintf("%s://%s/stream/%s", c.Protocol(), c.Hostname(), suuid),
		"Type":                "room",
	}
}

func CreateOrRetrieveRoom(uuid string) (string, string, *webrtc.CustomRoomManager) {
	webrtc.StreamsLock.Lock()
	defer webrtc.StreamsLock.Unlock()

	suuid := generateStreamUUID(uuid)

	if room := webrtc.CustomRooms[uuid]; room != nil {
		if _, ok := webrtc.CustomRooms[suuid]; !ok {
			webrtc.CustomStreams[suuid] = room
		}
		return uuid, suuid, room
	}

	room := createNewRoom(uuid, suuid)
	return uuid, suuid, room
}

func generateStreamUUID(uuid string) string {
	h := sha256.New()
	h.Write([]byte(uuid))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func createNewRoom(uuid, suuid string) *webrtc.CustomRoomManager {
	hub := customchat.NewCustomHub()
	p := &webrtc.CustomPeerManager{}
	p.TrackLocals = make(map[string]*w.TrackLocalStaticRTP)

	room := &webrtc.CustomRoomManager{
		Peers: p,
		Hub:   hub,
	}
	webrtc.CustomRooms[uuid] = room
	webrtc.CustomStreams[suuid] = room
	go hub.Start()
	return room
}

// HandleRoomViewerConnection handles the WebSocket connection for a room viewer.
func HandleRoomViewerConnection(conn *websocket.Conn, peers *webrtc.CustomPeerManager) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	defer conn.Close()

	for range ticker.C {
		sendViewerConnectionCount(conn, peers)
	}
}

func sendViewerConnectionCount(conn *websocket.Conn, peers *webrtc.CustomPeerManager) {
	w, err := conn.Conn.NextWriter(websocket.TextMessage)
	if err != nil {
		return
	}
	w.Write([]byte(fmt.Sprintf("%d", len(peers.Connections))))
}
