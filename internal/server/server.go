package server

import (
	"flag"
	"os"
	"time"

	"github.com/Parthiba-Hazra/golivesync/internal/handlers"
	"github.com/Parthiba-Hazra/golivesync/pkg/webrtc"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/template/html/v2"
	"github.com/gofiber/websocket/v2"
)

var (
	defaultPort = "8000"
)

func StartServer() error {
	// Parse command line flags and environment variables
	port := flag.String("port", ":"+os.Getenv("PORT"), "Port for the server")
	cert := flag.String("cert", "", "Path to SSL certificate")
	key := flag.String("key", "", "Path to SSL key")
	flag.Parse()

	// Set default port if not provided
	if *port == "" {
		*port = defaultPort
	}

	// TODO: add the view folder and necessary HTML files
	// Create HTML template engine TODO: front end is not created yet
	engine := html.New("./frontEnd/views", ".html")

	// Create Fiber app
	app := fiber.New(fiber.Config{
		Views: engine,
	})

	// Middleware
	app.Use(logger.New())
	app.Use(cors.New())

	// Define routes and WebSocket handlers
	defineRoutes(app)

	// Initialize the Custom WebRTC Rooms and Streams
	webrtc.CustomRooms = make(map[string]*webrtc.CustomRoomManager)
	webrtc.CustomStreams = make(map[string]*webrtc.CustomRoomManager)

	// Launch the routine to dispatch key frames
	go customDispatchKeyFrames()

	// Listen for incoming connections
	if *cert != "" {
		return app.ListenTLS(*port, *cert, *key)
	}

	return app.Listen(*port)
}

func defineRoutes(app *fiber.App) {
	// Room routes
	app.Get("/room/create", handlers.GenerateNewRoomUUID)
	app.Get("/room/:uuid", handlers.ServeRoom)
	app.Get("/room/:uuid/websocket", websocket.New(handlers.HandleRoomWebsocket, websocket.Config{
		HandshakeTimeout: 10 * time.Second,
	}))

	// Chat routes
	app.Get("/room/:uuid/chat", handlers.ServeLiveChat)
	app.Get("/room/:uuid/chat/websocket", websocket.New(handlers.HandleLiveRoomChatWebsocket))
	app.Get("/room/:uuid/viewer/websocket", websocket.New(handlers.HandleRoomViewerWebsocket))

	// Stream routes
	app.Get("/stream/:ssuid", handlers.ServeCustomStream)
	app.Get("/stream/:ssuid/websocket", websocket.New(handlers.HandleCustomStreamWebsocket, websocket.Config{HandshakeTimeout: 10 * time.Second}))
	app.Get("/stream/:ssuid/chat/websocket", websocket.New(handlers.HandleStreamChatWebsocket))
	app.Get("/stream/:ssuid/viewer/websocket", websocket.New(handlers.HandleCustomStreamViewerWebsocket))
}

// customDispatchKeyFrames periodically sends key frames to connected peers.
func customDispatchKeyFrames() {
	for range time.NewTicker(time.Second * 3).C {
		for _, room := range webrtc.CustomRooms {
			room.Peers.DispatchCustomKeyFrame()
		}
	}
}
