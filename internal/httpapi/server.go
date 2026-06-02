package httpapi

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/pprof"
	"runtime"
	"runtime/debug"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"gc-mem-exporter/internal/metrics"
)

const prometheusContentType = "text/plain; version=0.0.4; charset=utf-8"

// Server содержит HTTP-обработчики приложения.
type Server struct {
	collector *metrics.Collector
	mux       *http.ServeMux
	retained  [][]byte
	retMu     sync.Mutex
	retBytes  atomic.Int64
}

// NewServer создает HTTP-сервер с endpoint-ами приложения и pprof.
func NewServer(collector *metrics.Collector) *Server {
	s := &Server{
		collector: collector,
		mux:       http.NewServeMux(),
	}

	s.registerRoutes()

	return s
}

// Handler возвращает корневой HTTP handler.
func (s *Server) Handler() http.Handler {
	return requestLogger(s.mux)
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("GET /", s.handleIndex)
	s.mux.HandleFunc("GET /health", s.handleHealth)
	s.mux.HandleFunc("GET /metrics", s.handleMetrics)
	// POST /gc запускает принудительную сборку мусора для демонстрации изменения метрик.
	s.mux.HandleFunc("POST /gc", s.handleRunGC)
	// GET /gc-percent показывает текущее значение GOGC, POST /gc-percent?value=50 меняет его.
	s.mux.HandleFunc("GET /gc-percent", s.handleGetGCPercent)
	s.mux.HandleFunc("POST /gc-percent", s.handleSetGCPercent)
	// POST /alloc?mb=10&keep=true создает нагрузку на память, DELETE /alloc очищает удерживаемую память.
	s.mux.HandleFunc("POST /alloc", s.handleAlloc)
	s.mux.HandleFunc("DELETE /alloc", s.handleClearAlloc)

	s.registerPprof()
}

func (s *Server) registerPprof() {
	s.mux.HandleFunc("GET /debug/pprof/", pprof.Index)
	s.mux.HandleFunc("GET /debug/pprof/cmdline", pprof.Cmdline)
	s.mux.HandleFunc("GET /debug/pprof/profile", pprof.Profile)
	s.mux.HandleFunc("GET /debug/pprof/symbol", pprof.Symbol)
	s.mux.HandleFunc("GET /debug/pprof/trace", pprof.Trace)
}

func (s *Server) handleIndex(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintln(w, "GC Memory Exporter")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Endpoints:")
	fmt.Fprintln(w, "  GET    /health")
	fmt.Fprintln(w, "  GET    /metrics")
	fmt.Fprintln(w, "  GET    /gc-percent")
	fmt.Fprintln(w, "  POST   /gc-percent?value=50")
	fmt.Fprintln(w, "  POST   /gc")
	fmt.Fprintln(w, "  POST   /alloc?mb=10&keep=true")
	fmt.Fprintln(w, "  DELETE /alloc")
	fmt.Fprintln(w, "  GET    /debug/pprof/")
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleMetrics(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", prometheusContentType)
	_, _ = w.Write([]byte(s.collector.Prometheus()))
}

func (s *Server) handleRunGC(w http.ResponseWriter, _ *http.Request) {
	start := time.Now()
	runtime.GC()

	writeJSON(w, http.StatusOK, map[string]any{
		"result":      "gc completed",
		"duration_ms": time.Since(start).Milliseconds(),
	})
}

func (s *Server) handleGetGCPercent(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]int{"gc_percent": s.collector.GCPercent()})
}

func (s *Server) handleSetGCPercent(w http.ResponseWriter, r *http.Request) {
	raw := r.URL.Query().Get("value")
	if raw == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "query parameter value is required"})
		return
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "value must be integer"})
		return
	}

	previous := debug.SetGCPercent(value)
	s.collector.SetGCPercent(value)

	writeJSON(w, http.StatusOK, map[string]int{
		"previous_gc_percent": previous,
		"current_gc_percent":  value,
	})
}

func (s *Server) handleAlloc(w http.ResponseWriter, r *http.Request) {
	mb := 10
	if raw := r.URL.Query().Get("mb"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "mb must be positive integer"})
			return
		}
		mb = value
	}

	keep := r.URL.Query().Get("keep") != "false"
	bytesCount := mb * 1024 * 1024
	data := make([]byte, bytesCount)
	for i := range data {
		data[i] = byte(i)
	}

	if keep {
		s.retMu.Lock()
		s.retained = append(s.retained, data)
		s.retMu.Unlock()
		s.retBytes.Add(int64(bytesCount))
	} else {
		runtime.KeepAlive(data)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"allocated_mb":      mb,
		"kept_in_memory":    keep,
		"retained_bytes":    s.retBytes.Load(),
		"retained_chunks":   s.retainedChunks(),
		"check_metrics_url": "/metrics",
	})
}

func (s *Server) handleClearAlloc(w http.ResponseWriter, _ *http.Request) {
	s.retMu.Lock()
	s.retained = nil
	s.retMu.Unlock()
	s.retBytes.Store(0)
	runtime.GC()

	writeJSON(w, http.StatusOK, map[string]any{
		"result":         "retained memory cleared",
		"retained_bytes": s.retBytes.Load(),
	})
}

func (s *Server) retainedChunks() int {
	s.retMu.Lock()
	defer s.retMu.Unlock()

	return len(s.retained)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("method=%s path=%s duration=%s", r.Method, r.URL.Path, time.Since(started))
	})
}
