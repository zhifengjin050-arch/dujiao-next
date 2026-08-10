package channel

import "github.com/dujiao-next/internal/provider"

// Handler ?? API ???(Telegram Bot ??????????)
type Handler struct {
	*provider.Container
}

// New ???????
func New(c *provider.Container) *Handler {
	return &Handler{Container: c}
}
