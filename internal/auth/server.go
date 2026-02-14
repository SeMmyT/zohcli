package auth

import (
	"context"
	"fmt"
	"html"
	"net"
	"net/http"
	"time"
)

// callbackResult holds the OAuth2 callback parameters.
type callbackResult struct {
	Code  string
	State string
	Error string
}

// startCallbackServer starts a temporary HTTP server to receive the OAuth2 callback.
// Returns a channel for the result, the callback URL, and a shutdown function.
// The server automatically shuts down after receiving one callback or on context cancellation.
func startCallbackServer(ctx context.Context, port int) (resultChan chan callbackResult, addr string, shutdown func()) {
	resultChan = make(chan callbackResult, 1)

	// Try to bind to the requested port, fall back to random port if unavailable
	var listener net.Listener
	var err error

	if port == 0 {
		port = 8080
	}

	listener, err = net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		// Port unavailable, try random port
		listener, err = net.Listen("tcp", ":0")
		if err != nil {
			// Return error via channel
			go func() {
				resultChan <- callbackResult{Error: fmt.Sprintf("failed to start callback server: %v", err)}
			}()
			return resultChan, "", func() {}
		}
	}

	actualPort := listener.Addr().(*net.TCPAddr).Port
	addr = fmt.Sprintf("http://localhost:%d/callback", actualPort)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")

		if code == "" {
			errorMsg := r.URL.Query().Get("error")
			if errorMsg == "" {
				errorMsg = "missing authorization code"
			}
			resultChan <- callbackResult{Error: errorMsg}

			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head><title>Authentication Failed</title></head>
<body>
<h1>Authentication Failed</h1>
<p>Error: %s</p>
<p>You can close this window and try again.</p>
</body>
</html>`, html.EscapeString(errorMsg))
			return
		}

		resultChan <- callbackResult{Code: code, State: state}

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head><title>Authentication Successful</title></head>
<body>
<h1>Authentication Successful!</h1>
<p>You can close this window and return to the terminal.</p>
</body>
</html>`)
	})

	server := &http.Server{
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		server.Serve(listener)
	}()

	shutdown = func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}

	// Auto-shutdown on context cancellation
	go func() {
		<-ctx.Done()
		shutdown()
	}()

	return resultChan, addr, shutdown
}
