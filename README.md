# GoLiveSync

GoLiveSync is a real-time multimedia streaming and communication platform built using Go Fiber and WebRTC.

## Project Description

GoLiveSync is designed to facilitate real-time multimedia streaming and communication, enabling users to create and join live streaming rooms. It uses the WebRTC protocol for audio and video streaming and offers chat functionality within streaming rooms.

The project is divided into several packages, each responsible for specific functionalities:

## `handlers` Package

The `handlers` package handles HTTP requests and WebSocket connections, providing the core functionalities for creating and managing streaming rooms, handling chat messages, and managing WebRTC connections.

### Functionality

- Creating and joining streaming rooms
- WebSocket connections for room management, chat, and viewers
- Handling video streaming using WebRTC

## `chat` Package

The `chat` package manages real-time chat functionality within streaming rooms.

### Functionality

- Managing chat messages between users in the same streaming room

## `webrtc` Package

The `webrtc` package handles WebRTC connections and streaming functionalities.

### Functionality

- Managing WebRTC peer connections and streams
- Sending and receiving video streams
- Managing ICE candidates for establishing connections

## `server` Package

The `server` package configures and runs the Fiber web server, setting up routes and middleware for the application.

### Functionality

- Configuring the web server
- Setting up routes for different functionalities
- Running the server to handle incoming connections

## Getting Started

To run the project locally, follow these steps:

1. Clone the repository: `git clone https://github.com/Parthiba-Hazra/GoLiveSync.git`
2. Navigate to the project directory: `cd GoLiveSync`
3. Install dependencies: `go mod tidy`
4. Run the server: `go run main.go`

NOTE: currently it has only the bacckend will add the front end in future.

## Contributing

Contributions to GoLiveSync are welcome! Feel free to submit pull requests or open issues for any enhancements or bug fixes.

