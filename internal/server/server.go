package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/lucinexus/lucinexus/internal/config"
	"github.com/lucinexus/lucinexus/internal/engine"
)

type Server struct {
	cfg   *config.Config
	orch  *engine.Orchestrator
	srv   *http.Server
}

func NewServer(cfg *config.Config, orch *engine.Orchestrator) *Server {
	s := &Server{cfg: cfg, orch: orch}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/health", s.handleHealth)
	mux.HandleFunc("/api/v1/execute", s.handleExecute)

	s.srv = &http.Server{
		Addr:         fmt.Sprintf("127.0.0.1:%d", cfg.ServerPort),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	return s
}

func (s *Server) Start() error {
	return s.srv.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  "healthy",
		"version": config.Version,
	})
}

func (s *Server) handleExecute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Query  string `json:"query"`
		DryRun *bool  `json:"dry_run,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	dryRun := s.cfg.DryRunDefault
	if req.DryRun != nil {
		dryRun = *req.DryRun
	}

	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(s.cfg.TimeoutSec)*time.Second)
	defer cancel()

	plan, err := s.orch.Execute(ctx, req.Query, dryRun)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": err.Error(),
			"plan":  plan,
		})
		return
	}

	_ = json.NewEncoder(w).Encode(plan)
}
