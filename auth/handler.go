package auth

type Client struct {
	Name   string
	Weight int
}

type AuthHandler struct {
	clients map[string]Client
}

func NewAuthHandler() *AuthHandler {
	return &AuthHandler{
		clients: make(map[string]Client),
	}
}

// VerifyRegistered validates if the client is registered
func (h *AuthHandler) VerifyRegistered(name string) bool {
	_, ok := h.clients[name]
	return ok
}

// RegisterClient registers a client
func (h *AuthHandler) RegisterClient(name string, weight int) {
	h.clients[name] = Client{
		Name:   name,
		Weight: weight,
	}
}
