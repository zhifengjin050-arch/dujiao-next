package app

import (
	"context"
	"errors"
	"net/http"
)

// HTTPService HTTP ????
type HTTPService struct {
	name   string
	server *http.Server
}

// NewHTTPService ?? HTTP ??
func NewHTTPService(addr string, handler http.Handler) *HTTPService {
	return &HTTPService{
		name: "http",
		server: &http.Server{
			Addr:    addr,
			Handler: handler,
		},
	}
}

// Name ????
func (s *HTTPService) Name() string {
	if s == nil || s.name == "" {
		return "http"
	}
	return s.name
}

// Start ????
func (s *HTTPService) Start(ctx context.Context) error {
	if s == nil || s.server == nil {
		return errors.New("http server not initialized")
	}
	if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Stop ????
func (s *HTTPService) Stop(ctx context.Context) error {
	if s == nil || s.server == nil {
		return nil
	}
	return s.server.Shutdown(ctx)
}
