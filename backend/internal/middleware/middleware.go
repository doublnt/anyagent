package middleware

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const UserIDKey contextKey = "user_id"
const ScopesKey contextKey = "scopes"
const EntitlementIDKey contextKey = "entitlement_id"
const AgentIDKey contextKey = "agent_id"

var jwtSecret []byte

func getJWTSecret() []byte {
	if jwtSecret == nil {
		secret := os.Getenv("JWT_SECRET")
		if secret == "" {
			secret = "anyagent-dev-secret-change-in-production"
		}
		jwtSecret = []byte(secret)
	}
	return jwtSecret
}

// Chain applies middleware in order
func Chain(h http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}

// Logger logs request details
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

// CORS adds CORS headers
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Recovery catches panics
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Panic recovered: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// Auth validates JWT bearer token and sets user_id and scopes in context.
func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error":"missing authorization header"}`, http.StatusUnauthorized)
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenStr == authHeader {
			http.Error(w, `{"error":"invalid authorization format"}`, http.StatusUnauthorized)
			return
		}

		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			return getJWTSecret(), nil
		})
		if err != nil || !token.Valid {
			http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			http.Error(w, `{"error":"invalid token claims"}`, http.StatusUnauthorized)
			return
		}

		userID, ok := claims["sub"].(string)
		if !ok || userID == "" {
			http.Error(w, `{"error":"invalid user id in token"}`, http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, userID)

		// Extract scopes if present
		if scopesRaw, ok := claims["scopes"].([]interface{}); ok {
			var scopes []string
			for _, s := range scopesRaw {
				if ss, ok := s.(string); ok {
					scopes = append(scopes, ss)
				}
			}
			ctx = context.WithValue(ctx, ScopesKey, scopes)
		}

		// Extract agent_id if present (for entitlement tokens)
		if agentID, ok := claims["agent_id"].(string); ok {
			ctx = context.WithValue(ctx, AgentIDKey, agentID)
		}
		// Extract entitlement_id if present
		if eid, ok := claims["entitlement_id"].(string); ok {
			ctx = context.WithValue(ctx, EntitlementIDKey, eid)
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireScope returns a middleware that enforces the given scope.
// A valid token must contain the required scope in its "scopes" claim.
// Read = browse/store, Use = call hosted agent, Manage = publish/edit agents.
func RequireScope(required string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			scopesVal := ctx.Value(ScopesKey)
			if scopesVal == nil {
				http.Error(w, `{"error":"insufficient permissions: no scope claim"}`, http.StatusForbidden)
				return
			}
			scopes, ok := scopesVal.([]string)
			if !ok {
				http.Error(w, `{"error":"insufficient permissions: invalid scope claim"}`, http.StatusForbidden)
				return
			}
			for _, s := range scopes {
				if s == required {
					next.ServeHTTP(w, r)
					return
				}
			}
			http.Error(w, `{"error":"insufficient permissions: missing `+required+` scope"}`, http.StatusForbidden)
		})
	}
}

// GetUserID extracts user_id from context
func GetUserID(ctx context.Context) string {
	if v, ok := ctx.Value(UserIDKey).(string); ok {
		return v
	}
	return ""
}

// GetScopes extracts scopes from context
func GetScopes(ctx context.Context) []string {
	if v, ok := ctx.Value(ScopesKey).([]string); ok {
		return v
	}
	return nil
}

// GetEntitlementID extracts entitlement_id from context
func GetEntitlementID(ctx context.Context) string {
	if v, ok := ctx.Value(EntitlementIDKey).(string); ok {
		return v
	}
	return ""
}

// GetAgentID extracts agent_id from context
func GetAgentID(ctx context.Context) string {
	if v, ok := ctx.Value(AgentIDKey).(string); ok {
		return v
	}
	return ""
}

// GenerateToken creates a JWT token for a user with optional scopes.
// scopes: "read" (browse), "use" (call hosted agent), "manage" (publish/edit).
func GenerateToken(userID string, scopes ...string) (string, error) {
	claims := jwt.MapClaims{
		"sub":    userID,
		"iat":    time.Now().Unix(),
		"exp":    time.Now().Add(30 * 24 * time.Hour).Unix(), // 30 days
		"scopes": scopes,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(getJWTSecret())
}

// GenerateEntitlementToken creates a short-lived token scoped to a specific entitlement.
// It carries scope=use, the entitlement_id, and the agent_id so the MCP gateway
// can validate it without a DB round-trip.
func GenerateEntitlementToken(buyerUserID, entitlementID, agentID string) (string, error) {
	claims := jwt.MapClaims{
		"sub":           buyerUserID,
		"entitlement_id": entitlementID,
		"agent_id":      agentID,
		"iat":           time.Now().Unix(),
		"exp":           time.Now().Add(24 * time.Hour).Unix(), // 24h; gateway re-checks quota each call
		"scopes":        []string{"use"},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(getJWTSecret())
}
