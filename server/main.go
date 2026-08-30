package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"

	"glam/server/api"
)

func init() {
	// Load .env from multiple locations — dotenv
	// 1) server/.env (when running from server dir)
	// 2) .env in project root (GLAM/.env)
	// 3) cwd .env
	for _, p := range []string{
		".env",
		"server/.env",
		"/home/curator/GLAM/server/.env",
		"/home/curator/GLAM/.env",
	} {
		if _, err := os.Stat(p); err == nil {
			_ = godotenv.Load(p)
		}
	}
	// Also try autodiscover (walks up)
	_ = godotenv.Load()
}

func main() {
	// Resolve schema and registry paths relative to binary or cwd
	schemaPath := resolvePath("schema/scenario.schema.json")
	registryPath := resolvePath("schema/asset-registry.json")

	// Fallback: if not found relative to cwd, try relative to executable dir's parent
	if _, err := os.Stat(schemaPath); os.IsNotExist(err) {
		exe, _ := os.Executable()
		base := filepath.Dir(exe)
		candidates := []string{
			filepath.Join(base, "schema/scenario.schema.json"),
			filepath.Join(base, "..", "schema/scenario.schema.json"),
			filepath.Join(filepath.Dir(base), "schema/scenario.schema.json"),
			"/home/curator/GLAM/schema/scenario.schema.json",
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
			registryPath = "/home/curator/GLAM/schema/asset-registry.json"
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
	// Try cwd
	if _, err := os.Stat(p); err == nil {
		abs, _ := filepath.Abs(p)
		return abs
	}
	// Try parent (when running from server dir)
	cwd, _ := os.Getwd()
	candidate := filepath.Join(cwd, p)
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	// Try /home/curator/GLAM
	abs := filepath.Join("/home/curator/GLAM", p)
	if _, err := os.Stat(abs); err == nil {
		return abs
	}
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
		// Allow localhost:5173 and any localhost for dev
		allowedOrigin := "http://localhost:5173"
		if origin == allowedOrigin {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		} else if origin != "" && isLocalhost(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			// Fallback: allow 5173 for vite proxy
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
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

func isLocalhost(origin string) bool {
	return origin == "http://localhost:5173" ||
		origin == "http://127.0.0.1:5173" ||
		origin == "http://localhost:3000" ||
		origin == "http://127.0.0.1:3000" ||
		origin == "http://127.0.0.1:8080"
}
