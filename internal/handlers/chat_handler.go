package handlers

import (
	"errors"

	"github.com/Parthiba-Hazra/golivesync/pkg/customchat"
	"github.com/Parthiba-Hazra/golivesync/pkg/webrtc"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

// ServeLiveChat serves the live chat room view.
func ServeLiveChat(c *fiber.Ctx) error {

	// TODO: Add the layouts
	return c.Render("livechat", fiber.Map{}, "layouts/main")
}

// HandleStreamChatWebsocket handles WebSocket connections for stream chat.
func HandleStreamChatWebsocket(c *websocket.Conn) {
	streamID := c.Params("streamID")
	if streamID == "" {
		return
	}

	stream, err := getOrCreateStream(streamID)
	if err != nil {
		return
	}

	// Create a new peer connection for chat
	customchat.NewPeerChatConnection(c.Conn, stream.Hub)
}

// HandleLiveRoomChatWebsocket handles WebSocket connections for live room chat.
func HandleLiveRoomChatWebsocket(c *websocket.Conn) {
	roomID := c.Params("roomID")
	if roomID == "" {
		return
	}

	room, err := getOrCreateRoom(roomID)
	if err != nil {
		return
	}

	// Create a new peer connection for chat
	customchat.NewPeerChatConnection(c.Conn, room.Hub)
}

// getOrCreateStream retrieves an existing stream or creates a new one.
func getOrCreateStream(streamID string) (*webrtc.CustomRoomManager, error) {
	webrtc.StreamsLock.Lock()
	defer webrtc.StreamsLock.Unlock()

	if stream, ok := webrtc.CustomStreams[streamID]; ok {
		return stream, nil
	}

	// Create a new stream and hub
	stream := &webrtc.CustomRoomManager{}
	hub := customchat.NewCustomHub()
	stream.Hub = hub
	go hub.Start()

	webrtc.CustomStreams[streamID] = stream
	return stream, nil
}

// getOrCreateRoom retrieves an existing room or returns an error.
func getOrCreateRoom(roomID string) (*webrtc.CustomRoomManager, error) {
	webrtc.StreamsLock.Lock()
	defer webrtc.StreamsLock.Unlock()

	if room, ok := webrtc.CustomRooms[roomID]; ok {
		return room, nil
	}

	return nil, errors.New("room not found")
}
