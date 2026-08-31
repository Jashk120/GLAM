package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"

	"glam/server/api"
)

// allowedOrigins holds the CORS allowlist parsed once at startup.
var allowedOrigins []string
var allowedOriginsSet map[string]struct{}

func init() {
	// Load .env from multiple locations — dotenv precedence:
	// 1) GLAM_ROOT env var if set (GLAM_ROOT/.env and GLAM_ROOT/server/.env)
	// 2) .env and server/.env relative to cwd
	// 3) autodiscover via godotenv.Load() (walks up directories)
	// System env always overrides file values.
	var candidates []string
	if root := os.Getenv("GLAM_ROOT"); root != "" {
		candidates = append(candidates,
			filepath.Join(root, ".env"),
			filepath.Join(root, "server", ".env"),
		)
	}
	candidates = append(candidates, ".env", "server/.env")
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			_ = godotenv.Load(p)
		}
	}
	// Also try autodiscover (walks up)
	_ = godotenv.Load()

	allowedOrigins = parseAllowedOrigins()
	allowedOriginsSet = make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowedOriginsSet[o] = struct{}{}
	}
}

func parseAllowedOrigins() []string {
	raw := os.Getenv("CORS_ALLOWED_ORIGINS")
	if strings.TrimSpace(raw) == "" {
		raw = "http://localhost:5173,http://127.0.0.1:5173,http://localhost:3000,http://127.0.0.1:3000"
	}
	parts := strings.Split(raw, ",")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		out = []string{"http://localhost:5173", "http://127.0.0.1:5173", "http://localhost:3000", "http://127.0.0.1:3000"}
	}
	return out
}

func main() {
	// Resolve schema and registry paths relative to GLAM_ROOT / cwd / executable
	schemaPath := resolvePath("schema/scenario.schema.json")
	registryPath := resolvePath("schema/asset-registry.json")

	// Fallback: if not found, try additional candidates in priority order:
	// GLAM_ROOT → executable dir and its parent → cwd absolute
	if _, err := os.Stat(schemaPath); os.IsNotExist(err) {
		var candidates []string
		if root := os.Getenv("GLAM_ROOT"); root != "" {
			candidates = append(candidates, filepath.Join(root, "schema/scenario.schema.json"))
		}
		if exe, err := os.Executable(); err == nil {
			base := filepath.Dir(exe)
			candidates = append(candidates,
				filepath.Join(base, "schema/scenario.schema.json"),
				filepath.Join(base, "..", "schema/scenario.schema.json"),
				filepath.Join(filepath.Dir(base), "schema/scenario.schema.json"),
			)
		}
		if abs, err := filepath.Abs("schema/scenario.schema.json"); err == nil {
			candidates = append(candidates, abs)
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				schemaPath = c
				registryPath = filepath.Join(filepath.Dir(c), "asset-registry.json")
				break
			}
		}
	}
	if _, err := os.Stat(registryPath); os.IsNotExist(err) {
		registryPath = filepath.Join(filepath.Dir(schemaPath), "asset-registry.json")
		if _, err := os.Stat(registryPath); os.IsNotExist(err) {
			// Try GLAM_ROOT and executable dir before giving up
			var regCandidates []string
			if root := os.Getenv("GLAM_ROOT"); root != "" {
				regCandidates = append(regCandidates, filepath.Join(root, "schema/asset-registry.json"))
			}
			if exe, err := os.Executable(); err == nil {
				base := filepath.Dir(exe)
				regCandidates = append(regCandidates,
					filepath.Join(base, "schema/asset-registry.json"),
					filepath.Join(base, "..", "schema/asset-registry.json"),
					filepath.Join(filepath.Dir(base), "schema/asset-registry.json"),
				)
			}
			for _, c := range regCandidates {
				if _, err := os.Stat(c); err == nil {
					registryPath = c
					break
				}
			}
		}
	}

	log.Printf("Using schema: %s", schemaPath)
	log.Printf("Using registry: %s", registryPath)

	h, err := api.NewHandler(schemaPath, registryPath)
	if err != nil {
		log.Fatalf("failed to init handler: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", api.HandleHealth)
	mux.HandleFunc("/api/assets", h.HandleAssets)
	mux.HandleFunc("/api/scenario/validate", h.HandleValidate)
	mux.HandleFunc("/api/scenario/generate", h.HandleGenerate)

	handler := withCORS(withLogging(mux))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port
	log.Printf("GLAM server listening on %s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func resolvePath(p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	// 1) GLAM_ROOT if set (abs or relative)
	if root := os.Getenv("GLAM_ROOT"); root != "" {
		cand := filepath.Join(root, p)
		if _, err := os.Stat(cand); err == nil {
			if abs, err := filepath.Abs(cand); err == nil {
				return abs
			}
			return cand
		}
	}
	// 2) cwd (relative p and cwd/p)
	if _, err := os.Stat(p); err == nil {
		abs, _ := filepath.Abs(p)
		return abs
	}
	if cwd, err := os.Getwd(); err == nil && cwd != "" {
		cand := filepath.Join(cwd, p)
		if _, err := os.Stat(cand); err == nil {
			return cand
		}
	}
	// 3) executable dir and its parent
	if exe, err := os.Executable(); err == nil {
		base := filepath.Dir(exe)
		cand := filepath.Join(base, p)
		if _, err := os.Stat(cand); err == nil {
			return cand
		}
		cand = filepath.Join(base, "..", p)
		if _, err := os.Stat(cand); err == nil {
			if abs, err := filepath.Abs(cand); err == nil {
				if _, err := os.Stat(abs); err == nil {
					return abs
				}
			}
			return cand
		}
		parent := filepath.Dir(base)
		cand = filepath.Join(parent, p)
		if _, err := os.Stat(cand); err == nil {
			return cand
		}
	}
	// 4) relative fallback
	return p
}

func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s", r.Method, r.URL.Path, r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && isAllowedOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "false")
		w.Header().Set("Vary", "Origin")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func isAllowedOrigin(origin string) bool {
	_, ok := allowedOriginsSet[origin]
	return ok
}

func isLocalhost(origin string) bool {
	return isAllowedOrigin(origin)
}
