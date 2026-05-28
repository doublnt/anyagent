// Egress Proxy (Harpoon) — a forward-only HTTP proxy that only allows requests
// to a configured allowlist. Sandbox containers route ALL outbound HTTP through this proxy
// so the platform can inspect, meter, and rate-limit LLM calls.
package main

import (
	"log"
	"net/http"
	"os"
	"strings"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	addr := ":" + port
	log.Printf("Egress proxy listening on %s", addr)

	proxy := &HarpoonProxy{
		allowlist: parseAllowlist(os.Getenv("ALLOWLIST")),
		client:    &http.Client{Timeout: 60_000},
	}

	if err := http.ListenAndServe(addr, proxy); err != nil {
		log.Fatalf("Egress proxy failed: %v", err)
	}
}

type AllowEntry struct {
	Host   string
	Path   string
	Method []string
}

type HarpoonProxy struct {
	allowlist []AllowEntry
	client   *http.Client
}

func parseAllowlist(env string) []AllowEntry {
	// Default allowlist: api.anthropic.com only
	if env == "" {
		return []AllowEntry{
			{Host: "api.anthropic.com", Path: "/v1/messages", Method: []string{"POST"}},
			{Host: "api.anthropic.com", Path: "/v1/tokenize", Method: []string{"POST"}},
		}
	}
	// Format: host:path:method,host:path:method,...
	var entries []AllowEntry
	for _, part := range strings.Split(env, ",") {
		parts := strings.Split(part, ":")
		if len(parts) >= 2 {
			e := AllowEntry{Host: parts[0], Path: parts[1]}
			if len(parts) >= 3 {
				e.Method = strings.Split(parts[2], "|")
			}
			entries = append(entries, e)
		}
	}
	return entries
}

func (p *HarpoonProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	host := r.URL.Hostname()
	path := r.URL.Path

	if !p.isAllowed(host, path, r.Method) {
		http.Error(w, `{"error":"egress not allowed: destination not in allowlist"}`, 403)
		return
	}

	// Forward the request, stripping any proxy-specific hop-by-hop headers
	req, _ := http.NewRequestWithContext(r.Context(), r.Method, r.URL.String(), r.Body)
	req.Header = r.Header.Clone()
	// Let the platform's API key through
	if apiKey := os.Getenv("MODEL_API_KEY"); apiKey != "" {
		req.Header.Set("x-api-key", apiKey)
	}
	req.Header.Set("anthropic-version", "2023-06-01")
	// Strip hop-by-hop
	for _, h := range []string{"Proxy-Connection", "Proxy-Authenticate", "Te", "Trailer", "Upgrade"} {
		req.Header.Del(h)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		http.Error(w, `{"error":"upstream unreachable"}`, 502)
		return
	}
	defer resp.Body.Close()

	for k, v := range resp.Header {
		if k == "Transfer-Encoding" || k == "Connection" {
			continue
		}
		w.Header()[k] = v
	}
	w.WriteHeader(resp.StatusCode)
	// Stream response body
	buf := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			w.Write(buf[:n])
		}
		if err != nil {
			break
		}
	}
}

func (p *HarpoonProxy) isAllowed(host, path, method string) bool {
	for _, e := range p.allowlist {
		if e.Host != host {
			continue
		}
		if e.Path != "" && !strings.HasPrefix(path, e.Path) {
			continue
		}
		if len(e.Method) > 0 {
			allowed := false
			for _, m := range e.Method {
				if m == method {
					allowed = true
					break
				}
			}
			if !allowed {
				continue
			}
		}
		return true
	}
	return false
}
