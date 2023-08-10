package handlers

import (
	"fmt"
	"os"
	"time"

	"github.com/Parthiba-Hazra/golivesync/pkg/webrtc"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

func ServeCustomStream(c *fiber.Ctx) error {
	customStreamID := c.Params("streamID")
	if customStreamID == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Bad Request")
	}

	wsScheme := "ws"
	if os.Getenv("ENVIRONMENT") == "PRODUCTION" {
		wsScheme = "wss"
	}

	if _, ok := webrtc.CustomStreams[customStreamID]; ok {
		return serveCustomStreamPage(c, customStreamID, wsScheme)
	}

	return serveNoStreamPage(c)
}

func serveCustomStreamPage(c *fiber.Ctx, customStreamID, wsScheme string) error {
	return c.Render("customstream", fiber.Map{
		"StreamWebSocketAddr": fmt.Sprintf("%s://%s/stream/%s/websocket", wsScheme, c.Hostname(), customStreamID),
		"ChatWebSocketAddr":   fmt.Sprintf("%s://%s/stream/%s/chat/websocket", wsScheme, c.Hostname(), customStreamID),
		"ViewerWebSocketAddr": fmt.Sprintf("%s://%s/stream/%s/viewer/websocket", wsScheme, c.Hostname(), customStreamID),
		"Type":                "stream",
	}, "layouts/main")
}

func serveNoStreamPage(c *fiber.Ctx) error {
	return c.Render("customstream", fiber.Map{
		"NoStream": "true",
		"Leave":    "true",
	}, "layouts/main")
}

func HandleCustomStreamWebsocket(c *websocket.Conn) {
	handleCustomStreamWebsocket(c)
}

func HandleCustomStreamViewerWebsocket(c *websocket.Conn) {
	handleCustomStreamViewerWebsocket(c)
}

func handleCustomStreamWebsocket(c *websocket.Conn) {
	customStreamID := c.Params("customStreamID")
	if customStreamID == "" {
		return
	}
	streams, ok := getCustomStream(customStreamID)
	if !ok {
		return
	}
	webrtc.CustomStreamConnection(c, streams.Peers)
}

func handleCustomStreamViewerWebsocket(c *websocket.Conn) {
	customStreamID := c.Params("customStreamID")
	if customStreamID == "" {
		return
	}
	streams, ok := getCustomStream(customStreamID)
	if !ok {
		return
	}
	viewerConnection(c, streams.Peers)
}

func getCustomStream(customStreamID string) (*webrtc.CustomRoomManager, bool) {
	webrtc.StreamsLock.Lock()
	defer webrtc.StreamsLock.Unlock()

	streams, ok := webrtc.CustomStreams[customStreamID]
	return streams, ok
}

func viewerConnection(c *websocket.Conn, p *webrtc.CustomPeerManager) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	defer c.Close()

	for range ticker.C {
		sendViewerConnectionCount(c, p)
	}
}
