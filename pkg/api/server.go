package api

import (
	"log/slog"
	"net/http"
	"sync"
	"time"

	"zanguard/pkg/engine"
	"zanguard/pkg/schema"
	"zanguard/pkg/storage"
	"zanguard/pkg/tenant"
)

// Server is the combined HTTP server for the ZanGuard management and AuthZen 1.0 APIs.
type Server struct {
	store storage.TupleStore
	eng   *engine.Engine
	mgr   *tenant.Manager
	mux   *http.ServeMux
	log   *slog.Logger

	// schema registry: tenantID → raw YAML + compiled schema
	schemasMu sync.RWMutex
	schemas   map[string][]byte
	compiled  map[string]*schema.CompiledSchema
}

// NewServer creates a Server with all routes registered.
func NewServer(
	store storage.TupleStore,
	eng *engine.Engine,
	mgr *tenant.Manager,
	log *slog.Logger,
) *Server {
	s := &Server{
		store:    store,
		eng:      eng,
		mgr:      mgr,
		mux:      http.NewServeMux(),
		log:      log,
		schemas:  make(map[string][]byte),
		compiled: make(map[string]*schema.CompiledSchema),
	}
	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {
	// Health
	s.mux.HandleFunc("GET /healthz", s.handleHealth)

	// ── Management: Tenants ──────────────────────────────────────────────────
	s.mux.HandleFunc("POST /api/v1/tenants", s.handleCreateTenant)
	s.mux.HandleFunc("GET /api/v1/tenants", s.handleListTenants)
	s.mux.HandleFunc("GET /api/v1/tenants/{tenantID}", s.handleGetTenant)
	s.mux.HandleFunc("DELETE /api/v1/tenants/{tenantID}", s.handleDeleteTenant)
	s.mux.HandleFunc("POST /api/v1/tenants/{tenantID}/activate", s.handleActivateTenant)
	s.mux.HandleFunc("POST /api/v1/tenants/{tenantID}/suspend", s.handleSuspendTenant)

	// ── Management: Schema ───────────────────────────────────────────────────
	s.mux.HandleFunc("PUT /api/v1/tenants/{tenantID}/schema", s.handleLoadSchema)
	s.mux.HandleFunc("GET /api/v1/tenants/{tenantID}/schema", s.handleGetSchema)

	// ── Management: Tuples ───────────────────────────────────────────────────
	// /batch must be registered before the bare /tuples POST to ensure correct matching
	s.mux.HandleFunc("POST /api/v1/t/{tenantID}/tuples/batch", s.handleWriteTuples)
	s.mux.HandleFunc("POST /api/v1/t/{tenantID}/tuples", s.handleWriteTuple)
	s.mux.HandleFunc("DELETE /api/v1/t/{tenantID}/tuples", s.handleDeleteTuple)
	s.mux.HandleFunc("GET /api/v1/t/{tenantID}/tuples", s.handleReadTuples)

	// ── Management: Attributes ───────────────────────────────────────────────
	s.mux.HandleFunc("GET /api/v1/t/{tenantID}/attributes/objects", s.handleListObjectAttributes)
	s.mux.HandleFunc("GET /api/v1/t/{tenantID}/attributes/objects/{type}/{id}", s.handleGetObjectAttributes)
	s.mux.HandleFunc("PUT /api/v1/t/{tenantID}/attributes/objects/{type}/{id}", s.handleSetObjectAttributes)
	s.mux.HandleFunc("GET /api/v1/t/{tenantID}/attributes/subjects", s.handleListSubjectAttributes)
	s.mux.HandleFunc("GET /api/v1/t/{tenantID}/attributes/subjects/{type}/{id}", s.handleGetSubjectAttributes)
	s.mux.HandleFunc("PUT /api/v1/t/{tenantID}/attributes/subjects/{type}/{id}", s.handleSetSubjectAttributes)

	// ── Management: Changelog ────────────────────────────────────────────────
	s.mux.HandleFunc("GET /api/v1/t/{tenantID}/changelog", s.handleReadChangelog)

	// ── Management: Expand ───────────────────────────────────────────────────
	s.mux.HandleFunc("POST /api/v1/t/{tenantID}/expand", s.handleExpand)

	// ── AuthZen 1.0 ──────────────────────────────────────────────────────────
	s.mux.HandleFunc("POST /access/v1/evaluation", s.handleAuthZenEvaluation)
	s.mux.HandleFunc("POST /access/v1/evaluations", s.handleAuthZenEvaluations)
}

// ServeHTTP implements http.Handler with structured request logging.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	rw := &statusWriter{ResponseWriter: w, code: http.StatusOK}
	s.mux.ServeHTTP(rw, r)
	s.log.Info("http",
		"method", r.Method,
		"path", r.URL.Path,
		"status", rw.code,
		"ms", time.Since(start).Milliseconds(),
	)
}

// Start begins listening on addr (e.g. ":1997").
func (s *Server) Start(addr string) error {
	s.log.Info("server listening", "addr", addr)
	return http.ListenAndServe(addr, s)
}

// statusWriter captures the HTTP status code written by handlers for logging.
type statusWriter struct {
	http.ResponseWriter
	code int
}

func (sw *statusWriter) WriteHeader(code int) {
	sw.code = code
	sw.ResponseWriter.WriteHeader(code)
}
