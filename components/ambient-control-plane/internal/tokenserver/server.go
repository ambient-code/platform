package tokenserver

import (
	"context"
	"crypto/rsa"
	"fmt"
	"net/http"
	"time"

	"github.com/ambient-code/platform/components/ambient-control-plane/internal/auth"
	"github.com/rs/zerolog"
)

const (
	DefaultListenAddr   = ":8080"
	readTimeout         = 10 * time.Second
	writeTimeout        = 10 * time.Second
	idleTimeout         = 60 * time.Second
	shutdownGracePeriod = 5 * time.Second
)

type Server struct {
	srv    *http.Server
	logger zerolog.Logger
}

func New(
	listenAddr string,
	tokenProvider auth.TokenProvider,
	privateKey *rsa.PrivateKey,
	logger zerolog.Logger,
) (*Server, error) {
	h := &handler{
		tokenProvider: tokenProvider,
		privateKey:    privateKey,
		logger:        logger.With().Str("component", "tokenserver").Logger(),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/token", h.handleToken)
	mux.HandleFunc("/healthz", handleHealthz)

	return &Server{
		srv: &http.Server{
			Addr:         listenAddr,
			Handler:      mux,
			ReadTimeout:  readTimeout,
			WriteTimeout: writeTimeout,
			IdleTimeout:  idleTimeout,
		},
		logger: logger.With().Str("component", "tokenserver").Logger(),
	}, nil
}

func (s *Server) Start(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		s.logger.Info().Str("addr", s.srv.Addr).Msg("token server listening")
		if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownGracePeriod)
		defer cancel()
		if err := s.srv.Shutdown(shutdownCtx); err != nil {
			s.logger.Warn().Err(err).Msg("token server shutdown error")
		}
		return nil
	case err := <-errCh:
		return fmt.Errorf("token server: %w", err)
	}
}
