package server

import (
	"context"
	"fmt"
	"log"
	"mispilkabot/config"
	"mispilkabot/internal/server/prodamus"
	"net/http"
	"time"
)

type Server struct {
	server          *http.Server
	prodamusHandler *prodamus.Handler
}

func New(cfg *config.Config) *Server {
	mux := http.NewServeMux()

	handler := prodamus.NewHandler()
	handler.SetSecretKey(cfg.ProdamusSecret)
	handler.SetPrivateGroupID(cfg.PrivateGroupID)
	mux.Handle(cfg.WebhookPath, handler)

	addr := fmt.Sprintf("%s:%s", cfg.WebhookHost, cfg.WebhookPort)

	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &Server{server: srv, prodamusHandler: handler}
}

// SetGenerateInviteLinkCallback sets the callback for generating invite links
func (s *Server) SetGenerateInviteLinkCallback(callback func(userID, groupID string) (string, error)) {
	s.prodamusHandler.SetGenerateInviteLinkCallback(callback)
}

// SetInviteMessageCallback sets the callback for sending invite messages
func (s *Server) SetInviteMessageCallback(callback func(userID, inviteLink string)) {
	s.prodamusHandler.SetInviteMessageCallback(callback)
}

func (s *Server) Start(ctx context.Context) error {
	log.Printf("Starting HTTP server on %s", s.server.Addr)

	errChan := make(chan error, 1)

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	select {
	case <-ctx.Done():
		log.Println("Shutting down HTTP server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.server.Shutdown(shutdownCtx); err != nil {
			log.Printf("HTTP server shutdown error: %v", err)
		}
		return nil
	case err := <-errChan:
		return fmt.Errorf("HTTP server error: %w", err)
	}
}
