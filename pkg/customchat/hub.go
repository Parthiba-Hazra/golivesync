package customchat

// CustomHub manages clients and message broadcasting.
type CustomHub struct {
	Clients    map[*CustomClient]bool
	Broadcast  chan []byte
	Register   chan *CustomClient
	Unregister chan *CustomClient
}

// NewCustomHub creates a new CustomHub instance.
func NewCustomHub() *CustomHub {
	return &CustomHub{
		Clients:    make(map[*CustomClient]bool),
		Broadcast:  make(chan []byte),
		Register:   make(chan *CustomClient),
		Unregister: make(chan *CustomClient),
	}
}

func (h *CustomHub) registerClient(client *CustomClient) {
	h.Clients[client] = true
}

func (h *CustomHub) unregisterClient(client *CustomClient) {
	if _, ok := h.Clients[client]; ok {
		delete(h.Clients, client)
		close(client.Send)
	}
}

func (h *CustomHub) broadcastMessage(message []byte) {
	for client := range h.Clients {
		select {
		case client.Send <- message:
		default:
			close(client.Send)
			delete(h.Clients, client)
		}
	}
}

// Start starts the CustomHub's event loop for managing clients and broadcasting messages.
func (h *CustomHub) Start() {
	for {
		select {
		case client := <-h.Register:
			h.registerClient(client)

		case client := <-h.Unregister:
			h.unregisterClient(client)

		case message := <-h.Broadcast:
			h.broadcastMessage(message)
		}
	}
}
