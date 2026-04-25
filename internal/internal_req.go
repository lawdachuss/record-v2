package internal

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	cycletls "github.com/Danny-Dasilva/CycleTLS/cycletls"
	"github.com/HeapOfChaos/goondvr/server"
)

// Req represents an HTTP client with customized settings.
type Req struct {
	client     *http.Client
	cycleTLS   cycletls.CycleTLS // TLS fingerprint spoofing client for GitHub Actions
	useCycle   bool              // when true, use CycleTLS instead of standard http.Client
	isMedia    bool              // when true, omits browser-spoofing headers not needed for CDN media requests
	referer    string            // CDN Referer/Origin override; only used when isMedia is true
}

// NewReq creates a new HTTP client for Chaturbate page requests.
func NewReq() *Req {
	// Check if we should use CycleTLS (GitHub Actions mode with FlareSolverr)
	useCycleTLS := os.Getenv("USE_FLARESOLVERR") == "true"
	
	req := &Req{
		client: &http.Client{
			Transport: CreateTransport(),
		},
		useCycle: useCycleTLS,
	}
	
	// Initialize CycleTLS if needed
	if useCycleTLS {
		req.cycleTLS = cycletls.Init()
	}
	
	return req
}

// NewMediaReq creates a new HTTP client for CDN media requests (playlists, segments).
// It omits headers like X-Requested-With that are only needed for Chaturbate page fetches.
func NewMediaReq() *Req {
	// Check if we should use CycleTLS (GitHub Actions mode with FlareSolverr)
	useCycleTLS := os.Getenv("USE_FLARESOLVERR") == "true"
	
	req := &Req{
		client: &http.Client{
			Transport: CreateTransport(),
		},
		isMedia:  true,
		useCycle: useCycleTLS,
	}
	
	// Initialize CycleTLS if needed
	if useCycleTLS {
		req.cycleTLS = cycletls.Init()
	}
	
	return req
}

// NewMediaReqWithReferer creates a media HTTP client that sends the given URL as
// Referer and Origin instead of the Chaturbate defaults. Use this for non-Chaturbate CDNs.
func NewMediaReqWithReferer(referer string) *Req {
	// Check if we should use CycleTLS (GitHub Actions mode with FlareSolverr)
	useCycleTLS := os.Getenv("USE_FLARESOLVERR") == "true"
	
	req := &Req{
		client: &http.Client{
			Transport: CreateTransport(),
		},
		isMedia:  true,
		referer:  referer,
		useCycle: useCycleTLS,
	}
	
	// Initialize CycleTLS if needed
	if useCycleTLS {
		req.cycleTLS = cycletls.Init()
	}
	
	return req
}

// CreateTransport initializes a custom HTTP transport.
func CreateTransport() *http.Transport {
	// The DefaultTransport allows user changes the proxy settings via environment variables
	// such as HTTP_PROXY, HTTPS_PROXY.
	defaultTransport := http.DefaultTransport.(*http.Transport)

	newTransport := defaultTransport.Clone()
	newTransport.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
	}
	return newTransport
}

// Get sends an HTTP GET request and returns the response as a string.
func (h *Req) Get(ctx context.Context, url string) (string, error) {
	// FlareSolverr is NOT used for API endpoints
	// It's only useful for HTML pages with Cloudflare challenges
	// API endpoints like /api/chatvideocontext/ don't have Cloudflare challenges
	// They just check cookies and IP reputation
	
	// Original implementation (works fine with valid cookies)
	resp, err := h.GetBytes(ctx, url)
	if err != nil {
		return "", fmt.Errorf("get bytes: %w", err)
	}
	return string(resp), nil
}

// GetBytes sends an HTTP GET request and returns the response as a byte slice.
func (h *Req) GetBytes(ctx context.Context, url string) ([]byte, error) {
	// Use CycleTLS if enabled (GitHub Actions mode)
	if h.useCycle {
		return h.GetBytesWithCycleTLS(ctx, url)
	}
	
	// Standard HTTP client path
	req, cancel, err := h.CreateRequest(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	defer cancel()

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("client do: %w", err)
	}
	defer resp.Body.Close()

	if server.Config.Debug && resp.StatusCode >= 400 {
		fmt.Printf("[DEBUG] HTTP %d: %s\n", resp.StatusCode, req.URL)
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	// Check for Cloudflare protection
	if strings.Contains(string(b), "<title>Just a moment...</title>") {
		if server.Config.Debug {
			fmt.Printf("[DEBUG] CF response for %s (status %d)\n", req.URL, resp.StatusCode)
			tmpFile, ferr := os.CreateTemp("", "chaturbate-debug-cf-*.html")
			if ferr == nil {
				if _, werr := tmpFile.Write(b); werr == nil {
					fmt.Printf("[DEBUG]   Full body written to: %s\n", tmpFile.Name())
				}
				tmpFile.Close()
			}
		}
		return nil, ErrCloudflareBlocked
	}
	// Check for Age Verification
	if strings.Contains(string(b), "Verify your age") {
		return nil, ErrAgeVerification
	}

	if resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("forbidden: %w", ErrPrivateStream)
	}

	return b, err
}

// CreateRequest constructs an HTTP GET request with necessary headers.
func (h *Req) CreateRequest(ctx context.Context, url string) (*http.Request, context.CancelFunc, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second) // timed out after 10 seconds

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, cancel, err
	}
	h.SetRequestHeaders(req)
	return req, cancel, nil
}

// DoRequest executes an already-constructed *http.Request and returns the
// response body as a string. This allows callers to set extra headers on the
// request before executing it (e.g. site-specific Referer or X-Requested-With).
func (h *Req) DoRequest(req *http.Request) (string, error) {
	resp, err := h.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("client do: %w", err)
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}

	// Check for Cloudflare protection
	if strings.Contains(string(b), "<title>Just a moment...</title>") {
		return "", ErrCloudflareBlocked
	}
	// Check for Age Verification
	if strings.Contains(string(b), "Verify your age") {
		return "", ErrAgeVerification
	}

	if resp.StatusCode == http.StatusForbidden {
		return "", fmt.Errorf("forbidden: %w", ErrPrivateStream)
	}

	return string(b), nil
}

// SetRequestHeaders applies necessary headers to the request.
func (h *Req) SetRequestHeaders(req *http.Request) {
	if h.isMedia {
		ref := h.referer
		if ref == "" {
			ref = "https://chaturbate.com/"
		}
		req.Header.Set("Referer", ref)
		req.Header.Set("Origin", strings.TrimRight(ref, "/"))
	} else {
		// X-Requested-With helps bypass Cloudflare on chaturbate.com page fetches.
		// Do NOT send it to CDN media hosts (mmcdn.com) as it may cause rejection.
		req.Header.Set("X-Requested-With", "XMLHttpRequest")
	}
	if server.Config.UserAgent != "" {
		req.Header.Set("User-Agent", server.Config.UserAgent)
	}
	if server.Config.Cookies != "" {
		cookies := ParseCookies(server.Config.Cookies)
		for name, value := range cookies {
			req.AddCookie(&http.Cookie{Name: name, Value: value})
		}
	}
}

// ParseCookies converts a cookie string into a map.
func ParseCookies(cookieStr string) map[string]string {
	cookies := make(map[string]string)
	pairs := strings.Split(cookieStr, ";")

	// Iterate over each cookie pair and extract key-value pairs
	for _, pair := range pairs {
		parts := strings.SplitN(strings.TrimSpace(pair), "=", 2)
		if len(parts) == 2 {
			// Trim spaces around key and value
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			// Store cookie name and value in the map
			cookies[key] = value
		}
	}
	return cookies
}

// GetBytesWithCycleTLS sends an HTTP GET request using CycleTLS to spoof browser TLS fingerprint.
// This bypasses Cloudflare's TLS fingerprint detection in GitHub Actions.
func (h *Req) GetBytesWithCycleTLS(ctx context.Context, url string) ([]byte, error) {
	// Build headers map
	headers := make(map[string]string)
	
	if h.isMedia {
		ref := h.referer
		if ref == "" {
			ref = "https://chaturbate.com/"
		}
		headers["Referer"] = ref
		headers["Origin"] = strings.TrimRight(ref, "/")
	} else {
		headers["X-Requested-With"] = "XMLHttpRequest"
	}
	
	if server.Config.UserAgent != "" {
		headers["User-Agent"] = server.Config.UserAgent
	}
	
	// Add cookies
	if server.Config.Cookies != "" {
		headers["Cookie"] = server.Config.Cookies
	}
	
	// Make request with CycleTLS using Chrome 120 profile
	// This spoofs Chrome's TLS/HTTP2 fingerprint to bypass Cloudflare
	response, err := h.cycleTLS.Do(url, cycletls.Options{
		Body:      "",
		Ja3:       "771,4865-4866-4867-49195-49199-49196-49200-52393-52392-49171-49172-156-157-47-53,0-23-65281-10-11-35-16-5-13-18-51-45-43-27-17513,29-23-24,0",
		UserAgent: server.Config.UserAgent,
		Headers:   headers,
		Timeout:   10,
	}, "GET")
	
	if err != nil {
		return nil, fmt.Errorf("cycletls request: %w", err)
	}
	
	if server.Config.Debug && response.Status >= 400 {
		fmt.Printf("[DEBUG] HTTP %d: %s\n", response.Status, url)
	}
	
	if response.Status == http.StatusNotFound {
		return nil, ErrNotFound
	}
	
	body := []byte(response.Body)
	
	// Check for Cloudflare protection
	if strings.Contains(response.Body, "<title>Just a moment...</title>") {
		if server.Config.Debug {
			fmt.Printf("[DEBUG] CF response for %s (status %d)\n", url, response.Status)
			tmpFile, ferr := os.CreateTemp("", "chaturbate-debug-cf-*.html")
			if ferr == nil {
				if _, werr := tmpFile.Write(body); werr == nil {
					fmt.Printf("[DEBUG]   Full body written to: %s\n", tmpFile.Name())
				}
				tmpFile.Close()
			}
		}
		return nil, ErrCloudflareBlocked
	}
	
	// Check for Age Verification
	if strings.Contains(response.Body, "Verify your age") {
		return nil, ErrAgeVerification
	}
	
	if response.Status == http.StatusForbidden {
		return nil, fmt.Errorf("forbidden: %w", ErrPrivateStream)
	}
	
	return body, nil
}
