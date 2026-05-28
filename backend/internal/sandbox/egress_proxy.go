package sandbox

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

// EgressProxy is the "Harpoon" component: a forward-only HTTP proxy that only
// allows requests to a configured allowlist of host+method pairs.
// All LLM traffic from sandbox containers must go through this proxy so the
// platform can inspect, rate-limit, and meter it — sellers' private prompts
// never go directly to the model provider.
type EgressProxy struct {
	allowlist []AllowEntry
	client    *http.Client
}

// AllowEntry describes a single allowed destination.
type AllowEntry struct {
	Host   string   // e.g. "api.anthropic.com"
	Port   int      // e.g. 443
	Path   string   // e.g. "/v1/messages"; empty = all paths on host
	Method []string // e.g. ["POST"]; empty = all methods
}

var defaultAllowlist = []AllowEntry{
	{
		Host:   "api.anthropic.com",
		Port:   443,
		Path:   "/v1/messages",
		Method: []string{"POST"},
	},
	{
		Host:   "api.anthropic.com",
		Port:   443,
		Path:   "/v1/tokenize",
		Method: []string{"POST"},
	},
}

// NewEgressProxy creates a proxy that allows only the given allowlist.
// If allowlist is nil, the default Anthropic-only allowlist is used.
func NewEgressProxy(allowlist []AllowEntry) *EgressProxy {
	if allowlist == nil {
		allowlist = defaultAllowlist
	}
	return &EgressProxy{
		allowlist: allowlist,
		client: &http.Client{
			Timeout: 60_000, // 60s — LLM calls can be slow
		},
	}
}

// RoundTrip implements http.RoundTripper. It validates the request against
// the allowlist before forwarding it.
func (p *EgressProxy) RoundTrip(req *http.Request) (*http.Response, error) {
	host := req.URL.Hostname()
	port := 443
	if ps := req.URL.Port(); ps != "" {
		fmt.Sscanf(ps, "%d", &port)
	}
	path := req.URL.Path

	if !p.isAllowed(host, port, path, req.Method) {
		return &http.Response{
			StatusCode: 403,
			Body:       io.NopCloser(strings.NewReader(`{"error":"egress not allowed"}`)),
			Header:     http.Header{"Content-Type": {"application/json"}},
		}, nil
	}

	// Forward the request
	return p.client.Do(req)
}

func (p *EgressProxy) isAllowed(host string, port int, path, method string) bool {
	for _, entry := range p.allowlist {
		if entry.Host != host || entry.Port != port {
			continue
		}
		// Path: empty means all paths; otherwise match prefix
		if entry.Path != "" && !strings.HasPrefix(path, entry.Path) {
			continue
		}
		// Method: empty means all methods; otherwise match
		if len(entry.Method) > 0 {
			allowed := false
			for _, m := range entry.Method {
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

// ProxyHandler returns an http.Handler that proxies through this EgressProxy.
// The sandbox container's egress traffic is routed to this handler.
func (p *EgressProxy) ProxyHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !p.isAllowed(r.URL.Hostname(), 443, r.URL.Path, r.Method) {
			http.Error(w, `{"error":"egress not allowed"}`, 403)
			return
		}
		req, _ := http.NewRequestWithContext(r.Context(), r.Method, r.URL.String(), r.Body)
		req.Header = r.Header.Clone()
		resp, err := p.client.Do(req)
		if err != nil {
			http.Error(w, `{"error":"upstream unreachable"}`, 502)
			return
		}
		defer resp.Body.Close()
		for k, v := range resp.Header {
			w.Header()[k] = v
		}
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	})
}
