package ui

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// ── OAuth 2.0 PKCE flow ───────────────────────────────────────────────────────

// oauthTokenMsg is the Bubble Tea message returned when the OAuth flow completes.
type oauthTokenMsg struct {
	token string
	err   error
}

// oauthProviderConfig holds the OAuth 2.0 endpoints and credentials for one provider.
type oauthProviderConfig struct {
	authURL  string
	tokenURL string
	scope    string
	clientID string
}

// geminiDefaultClientID is the public OAuth client ID bundled with the
// official Gemini CLI (open-source). Using it avoids requiring users to
// register their own Google Cloud project.
const geminiDefaultClientID = "681255809395-oo8t2oprdrnp9e3aqf6av3hmdib135j.apps.googleusercontent.com"

// resolveOAuthConfig returns the OAuth 2.0 endpoints for provider.
// clientIDOverride is used first; if empty the VIBEMENU_*_CLIENT_ID env var is
// tried; for Gemini the bundled default client ID is used as final fallback.
// Returns an error only if the provider is unknown.
func resolveOAuthConfig(provider, clientIDOverride string) (oauthProviderConfig, error) {
	switch provider {
	case "Gemini":
		clientID := clientIDOverride
		if clientID == "" {
			clientID = os.Getenv("VIBEMENU_GOOGLE_CLIENT_ID")
		}
		if clientID == "" {
			clientID = geminiDefaultClientID
		}
		return oauthProviderConfig{
			authURL:  "https://accounts.google.com/o/oauth2/v2/auth",
			tokenURL: "https://oauth2.googleapis.com/token",
			scope:    "https://www.googleapis.com/auth/cloud-platform https://www.googleapis.com/auth/userinfo.email https://www.googleapis.com/auth/userinfo.profile",
			clientID: clientID,
		}, nil
	case "ChatGPT":
		clientID := clientIDOverride
		if clientID == "" {
			clientID = os.Getenv("VIBEMENU_OPENAI_CLIENT_ID")
		}
		return oauthProviderConfig{
			authURL:  "https://auth.openai.com/authorize",
			tokenURL: "https://auth.openai.com/oauth/token",
			scope:    "openid profile email",
			clientID: clientID,
		}, nil
	default:
		return oauthProviderConfig{}, fmt.Errorf("OAuth not supported for %s", provider)
	}
}

// generateCodeVerifier returns a random PKCE code verifier (RFC 7636).
func generateCodeVerifier() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// generateCodeChallenge returns the S256 PKCE code challenge for the verifier.
func generateCodeChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// generateState returns a random OAuth state nonce.
func generateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// startOAuthCmd wraps startOAuthFlow as a Bubble Tea command.
// clientID is passed directly so no env var lookup is needed at call time.
func startOAuthCmd(provider, clientID string) tea.Cmd {
	return func() tea.Msg {
		token, err := startOAuthFlow(provider, clientID)
		return oauthTokenMsg{token: token, err: err}
	}
}

// startOAuthFlow runs the PKCE OAuth 2.0 authorization code flow:
// starts a local HTTP server on :8080, opens the provider's auth URL in the
// browser, waits for the callback, exchanges the code for an access token,
// and returns it. Times out after 5 minutes.
func startOAuthFlow(provider, clientID string) (string, error) {
	cfg, err := resolveOAuthConfig(provider, clientID)
	if err != nil {
		return "", err
	}
	if cfg.clientID == "" {
		return "", fmt.Errorf("no OAuth client ID provided for %s", provider)
	}

	verifier, err := generateCodeVerifier()
	if err != nil {
		return "", fmt.Errorf("generate code verifier: %w", err)
	}
	challenge := generateCodeChallenge(verifier)

	state, err := generateState()
	if err != nil {
		return "", fmt.Errorf("generate state: %w", err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("failed to bind to local port: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	redirectURI := fmt.Sprintf("http://127.0.0.1:%d/callback", port)

	codeCh := make(chan string, 1)
	callbackErrCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			http.Error(w, "state mismatch", http.StatusBadRequest)
			callbackErrCh <- fmt.Errorf("OAuth state mismatch — possible CSRF")
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "missing authorization code", http.StatusBadRequest)
			callbackErrCh <- fmt.Errorf("no authorization code in callback")
			return
		}
		fmt.Fprint(w, "<html><body><h2>Authorization successful!</h2><p>You can close this tab.</p></body></html>")
		codeCh <- code
	})

	srv := &http.Server{Handler: mux}
	srvErrCh := make(chan error, 1)
	go func() {
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			srvErrCh <- err
		}
	}()
	defer func() {
		shutCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutCtx)
	}()

	params := url.Values{
		"response_type":         {"code"},
		"client_id":             {cfg.clientID},
		"redirect_uri":          {redirectURI},
		"scope":                 {cfg.scope},
		"state":                 {state},
		"access_type":           {"offline"},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
	}
	openBrowser(cfg.authURL + "?" + params.Encode())

	select {
	case code := <-codeCh:
		return exchangeCodeForToken(cfg, code, verifier, redirectURI)
	case err := <-callbackErrCh:
		return "", err
	case err := <-srvErrCh:
		return "", fmt.Errorf("OAuth callback server error: %w", err)
	case <-time.After(5 * time.Minute):
		return "", fmt.Errorf("OAuth timeout: no browser response within 5 minutes")
	}
}

// exchangeCodeForToken exchanges an authorization code for an access token
// using the PKCE verifier.
func exchangeCodeForToken(cfg oauthProviderConfig, code, verifier, redirectURI string) (string, error) {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {cfg.clientID},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"code_verifier": {verifier},
	}

	resp, err := http.PostForm(cfg.tokenURL, form)
	if err != nil {
		return "", fmt.Errorf("token exchange request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read token response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		preview := string(body)
		if len(preview) > 300 {
			preview = preview[:300]
		}
		return "", fmt.Errorf("token exchange failed (HTTP %d): %s", resp.StatusCode, preview)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("parse token response: %w", err)
	}
	if tokenResp.Error != "" {
		return "", fmt.Errorf("token exchange error: %s — %s", tokenResp.Error, tokenResp.ErrorDesc)
	}
	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("empty access token in response")
	}
	return tokenResp.AccessToken, nil
}

// enterCredentialStep prepares the credential input for the current auth method.
// For OAuth providers it either auto-starts the browser flow (if a client ID is
// already known) or prompts the user to enter one first.
func (p ProviderMenu) enterCredentialStep() (ProviderMenu, tea.Cmd) {
	p.focus = pmFocusCredential
	p.oauthStatus = ""
	p.oauthAwaitingClientID = false

	authMethod := ""
	provLabel := ""
	if p.selectedProv >= 0 {
		provLabel = p.providers[p.selectedProv].label
		if p.selectedAuth >= 0 {
			authMethod = p.providers[p.selectedProv].authMethods[p.selectedAuth]
		}
	}

	if authMethod == "OAuth" {
		p.credInput.EchoMode = textinput.EchoNormal
		// Check for a usable client ID (cached from this session or env var).
		cfg, _ := resolveOAuthConfig(provLabel, p.oauthClientID)
		if cfg.clientID != "" {
			// Client ID already known — launch the browser immediately.
			p.credInput.SetValue("")
			p.credInput.Placeholder = "token will appear here after browser authorization"
			p.oauthStatus = "Opening browser…"
			return p, tea.Batch(p.credInput.Focus(), startOAuthCmd(provLabel, cfg.clientID))
		}
		// No client ID yet — collect it first.
		p.oauthAwaitingClientID = true
		p.credInput.SetValue("")
		p.credInput.Placeholder = oauthClientIDPlaceholder(provLabel)
		return p, p.credInput.Focus()
	}

	// API Key path.
	p.credInput.EchoMode = textinput.EchoPassword
	p.credInput.EchoCharacter = '•'
	p.credInput.Placeholder = "sk-…"
	if p.selectedProv >= 0 {
		if existing, ok := p.configured[provLabel]; ok && existing.Credential != "" {
			p.credInput.SetValue(existing.Credential)
		} else {
			p.credInput.SetValue("")
		}
	}
	return p, p.credInput.Focus()
}

// oauthClientIDPlaceholder returns the placeholder text for the client ID input.
func oauthClientIDPlaceholder(provider string) string {
	switch provider {
	case "Gemini":
		return "Google OAuth Client ID (from console.cloud.google.com)"
	case "ChatGPT":
		return "OpenAI OAuth Client ID"
	default:
		return "OAuth Client ID"
	}
}

// Update handles keyboard input and returns a new ProviderMenu and optional command.
