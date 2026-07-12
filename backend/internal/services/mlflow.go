package services

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type MLflowService struct {
	enabled     bool
	trackingURI string
}

// NewMLflowService initializes the MLflow proxy service
func NewMLflowService(enabled bool, trackingURI string) *MLflowService {
	return &MLflowService{
		enabled:     enabled,
		trackingURI: trackingURI,
	}
}

// IsEnabled returns whether MLflow tracking is enabled
func (s *MLflowService) IsEnabled() bool {
	return s.enabled
}

// ProxyRequest forwards an HTTP request to the MLflow tracking server.
// If the tracking URI contains username:password credentials, basic authorization is added.
func (s *MLflowService) ProxyRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	if !s.enabled {
		return nil, fmt.Errorf("mlflow service is disabled")
	}

	// Parse the base tracking URI
	baseURI, err := url.Parse(s.trackingURI)
	if err != nil {
		return nil, fmt.Errorf("invalid tracking URI: %w", err)
	}

	// Parse the relative path (which may contain query parameters)
	relPath, err := url.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("invalid request path: %w", err)
	}

	// Resolve the full URL
	targetURL := baseURI.ResolveReference(relPath)

	// Create a new request with the provided context
	req, err := http.NewRequestWithContext(ctx, method, targetURL.String(), body)
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}

	// Forward client request headers stored in the context if they exist
	if reqHeaders, ok := ctx.Value("headers").(http.Header); ok {
		for k, vv := range reqHeaders {
			for _, v := range vv {
				req.Header.Add(k, v)
			}
		}
	}

	// Add basic authorization header if tracking URI has user:pass credentials
	if baseURI.User != nil {
		username := baseURI.User.Username()
		password, _ := baseURI.User.Password()
		req.SetBasicAuth(username, password)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute proxy request: %w", err)
	}

	return resp, nil
}
