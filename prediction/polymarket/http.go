package polymarket

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"nofx/prediction/telemetry"
	"os"
	"strings"
	"time"
)

func (c *Client) getJSON(rawURL string) ([]byte, error) {
	start := time.Now()
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "nofx-prediction/0.2")
	resp, err := c.http.Do(req)
	if err != nil {
		telemetry.RecordHTTP(httpCategory(rawURL), http.MethodGet, rawURL, time.Since(start), err, 0, 0)
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		telemetry.RecordHTTP(httpCategory(rawURL), http.MethodGet, rawURL, time.Since(start), err, resp.StatusCode, 0)
		return nil, err
	}
	if resp.StatusCode >= 400 {
		httpErr := fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncate(string(body), 200))
		telemetry.RecordHTTP(httpCategory(rawURL), http.MethodGet, rawURL, time.Since(start), httpErr, resp.StatusCode, len(body))
		return nil, httpErr
	}
	telemetry.RecordHTTP(httpCategory(rawURL), http.MethodGet, rawURL, time.Since(start), nil, resp.StatusCode, len(body))
	return body, nil
}

func (c *Client) doRequest(method, path string, body []byte, l2 bool) ([]byte, error) {
	u := c.cfg.ClobURL + path
	start := time.Now()
	var bodyReader io.Reader
	if len(body) > 0 {
		bodyReader = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, u, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "nofx-prediction/0.2")
	req.Header.Set("Content-Type", "application/json")
	if l2 {
		if err := c.applyL2Headers(req, method, path, body); err != nil {
			return nil, err
		}
	}
	resp, err := c.http.Do(req)
	if err != nil {
		telemetry.RecordHTTP("clob", method, u, time.Since(start), err, 0, 0)
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		telemetry.RecordHTTP("clob", method, u, time.Since(start), err, resp.StatusCode, 0)
		return nil, err
	}
	if resp.StatusCode >= 400 {
		httpErr := fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncate(string(respBody), 300))
		telemetry.RecordHTTP("clob", method, u, time.Since(start), httpErr, resp.StatusCode, len(respBody))
		return nil, httpErr
	}
	telemetry.RecordHTTP("clob", method, u, time.Since(start), nil, resp.StatusCode, len(respBody))
	return respBody, nil
}

func httpCategory(rawURL string) string {
	lower := strings.ToLower(rawURL)
	if strings.Contains(lower, "clob") {
		return "clob"
	}
	return "gamma"
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func (c *Client) ensureHTTP() {
	if c.http == nil {
		c.http = newHTTPClient(30 * time.Second)
	}
}

// newHTTPClient builds an outbound client for Polymarket APIs.
// POLYMARKET_HTTP_PROXY overrides HTTPS_PROXY/HTTP_PROXY when set.
func newHTTPClient(timeout time.Duration) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	for _, key := range []string{"POLYMARKET_HTTP_PROXY", "HTTPS_PROXY", "HTTP_PROXY"} {
		if raw := strings.TrimSpace(os.Getenv(key)); raw != "" {
			if u, err := url.Parse(raw); err == nil {
				transport.Proxy = http.ProxyURL(u)
			}
			break
		}
	}
	return &http.Client{Timeout: timeout, Transport: transport}
}
