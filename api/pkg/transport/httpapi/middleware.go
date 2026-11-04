package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"regexp"
	"runtime/debug"
	"strings"
	"time"
)

var requestIDPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{8,128}$`)

const (
	cspDefault = "default-src 'self'; script-src 'self'; style-src 'self'; " +
		"img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; " +
		"base-uri 'none'; form-action 'self'"
	// Swagger UI's stock template bootstraps itself with an inline <script>
	// and inline <style>. That page is admin-only and LAN-only, and no user
	// supplied content is rendered, so the docs subtree gets the narrowest
	// relaxation that still lets the UI start. The SPA keeps cspDefault.
	cspDocs = "default-src 'self'; script-src 'self' 'unsafe-inline'; " +
		"style-src 'self' 'unsafe-inline'; img-src 'self' data:; " +
		"connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'"
)

type requestIDKey struct{}

func requestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)

	return id
}

// withRequestID reuses a well formed inbound X-Request-Id, otherwise it
// generates a non-guessable one, and always echoes it back.
func (a *API) withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(r.Header.Get("X-Request-Id"))
		if !requestIDPattern.MatchString(id) {
			id = newRequestID()
		}

		w.Header().Set("X-Request-Id", id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey{}, id)))
	})
}

func newRequestID() string {
	var buf [8]byte

	if _, err := rand.Read(buf[:]); err != nil {
		// crypto/rand failing is catastrophic; still return a usable id.
		return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
	}

	return hex.EncodeToString(buf[:])
}

// recoverer turns a panic into a 500 without leaking the stack to the client.
func (a *API) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			recovered := recover()
			if recovered == nil {
				return
			}

			a.logger.Error(
				"panic recovered",
				slog.Any("panic", recovered),
				slog.String("stack", string(debug.Stack())),
				slog.String("request_id", requestIDFromContext(r.Context())),
				slog.String("path", r.URL.Path),
			)

			if recorder, ok := w.(*statusRecorder); ok && recorder.wroteHeader {
				return
			}

			writeEnvelope(w, http.StatusInternalServerError, "internal_error", "internal server error")
		}()

		next.ServeHTTP(w, r)
	})
}

// accessLog emits one JSON line per request to stderr. It intentionally logs
// only method/path/status/duration/ip/request id: headers (including
// Authorization), query strings and bodies are never logged.
func (a *API) accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &statusRecorder{ResponseWriter: w}

		next.ServeHTTP(recorder, r)

		status := recorder.status
		if status == 0 {
			status = http.StatusOK
		}

		a.logger.Info(
			"http request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", status),
			slog.Duration("duration", time.Since(start)),
			slog.String("ip", a.clientIP(r)),
			slog.String("request_id", requestIDFromContext(r.Context())),
		)
	})
}

// securityHeaders applies conservative defaults. API responses are never
// cached because they carry management state and, from P1, bearer tokens.
func (a *API) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := w.Header()
		header.Set("X-Content-Type-Options", "nosniff")
		header.Set("X-Frame-Options", "DENY")
		header.Set("Referrer-Policy", "no-referrer")

		if strings.HasPrefix(r.URL.Path, "/svnadmin/api/docs") {
			header.Set("Content-Security-Policy", cspDocs)
		} else {
			header.Set("Content-Security-Policy", cspDefault)
		}

		if strings.HasPrefix(r.URL.Path, "/svnadmin/") ||
			r.URL.Path == "/healthz" ||
			r.URL.Path == "/readyz" {
			header.Set("Cache-Control", "no-store")
		}

		next.ServeHTTP(w, r)
	})
}

// allowlistMiddleware guards every /svnadmin/ route, including the Swagger UI.
// Health endpoints stay reachable so container healthchecks keep working.
func (a *API) allowlistMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/svnadmin/") {
			next.ServeHTTP(w, r)

			return
		}

		addr, ok := a.allowlist.ResolveClient(r.RemoteAddr, r.Header)
		if !ok {
			a.logger.Warn(
				"request rejected: unparseable remote address",
				slog.String("remote_addr", r.RemoteAddr),
				slog.String("request_id", requestIDFromContext(r.Context())),
			)
			writeEnvelope(w, http.StatusForbidden, "forbidden", "source address is not allowed")

			return
		}

		ctx := context.WithValue(r.Context(), clientIPKey{}, addr)

		if !a.allowlist.Allows(addr) {
			a.logger.Warn(
				"request rejected by allowlist",
				slog.String("ip", addr.String()),
				slog.String("path", r.URL.Path),
				slog.String("request_id", requestIDFromContext(ctx)),
			)
			writeEnvelope(w, http.StatusForbidden, "forbidden", "source address is not allowed")

			return
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// clientIP resolves the effective client address for logging. It mirrors the
// allowlist so log and policy never disagree.
func (a *API) clientIP(r *http.Request) string {
	if addr, ok := a.allowlist.ResolveClient(r.RemoteAddr, r.Header); ok {
		return addr.String()
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return host
}

// ResolveTrustedProxies converts configured trusted proxy entries (CIDR, IP or
// hostname) to CIDR strings. Hostnames are resolved once at startup so request
// handling never performs DNS lookups.
func ResolveTrustedProxies(ctx context.Context, entries []string) (resolved, unresolved []string) {
	for _, entry := range entries {
		value := strings.TrimSpace(entry)
		if value == "" {
			continue
		}

		if _, err := parseNetwork(value); err == nil {
			resolved = append(resolved, value)

			continue
		}

		lookupCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		addrs, err := net.DefaultResolver.LookupNetIP(lookupCtx, "ip", value)
		cancel()

		if err != nil || len(addrs) == 0 {
			unresolved = append(unresolved, value)

			continue
		}

		for _, addr := range addrs {
			addr = addr.Unmap()
			resolved = append(resolved, netip.PrefixFrom(addr, addr.BitLen()).String())
		}
	}

	return resolved, unresolved
}
