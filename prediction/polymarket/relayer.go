package polymarket

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type relayerClient struct {
	baseURL    string
	http       *http.Client
	builderKey string
	secret     string
	passphrase string
}

type relayerNonceResp struct {
	Nonce string `json:"nonce"`
}

type relayerPayloadResp struct {
	Address string `json:"address"`
	Nonce   string `json:"nonce"`
}

type relayerSubmitResp struct {
	TransactionID   string `json:"transactionID"`
	State           string `json:"state"`
	TransactionHash string `json:"transactionHash"`
}

type relayerTransaction struct {
	State           string `json:"state"`
	TransactionHash string `json:"transactionHash"`
}

func newRelayerClient(cfg Config) *relayerClient {
	base := strings.TrimRight(strings.TrimSpace(cfg.RelayerURL), "/")
	if base == "" {
		base = defaultRelayerURL
	}
	return &relayerClient{
		baseURL:    base,
		http:       newHTTPClient(45 * time.Second),
		builderKey: cfg.BuilderAPIKey,
		secret:     cfg.BuilderSecret,
		passphrase: cfg.BuilderPassphrase,
	}
}

func (r *relayerClient) hasCreds() bool {
	return strings.TrimSpace(r.builderKey) != "" &&
		strings.TrimSpace(r.secret) != "" &&
		strings.TrimSpace(r.passphrase) != ""
}

func (r *relayerClient) getNonce(owner, txType string) (string, error) {
	u := fmt.Sprintf("%s/nonce?address=%s&type=%s", r.baseURL, url.QueryEscape(owner), url.QueryEscape(txType))
	body, err := r.do(http.MethodGet, u, nil, false)
	if err != nil {
		return "", err
	}
	var resp relayerNonceResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", err
	}
	if resp.Nonce == "" {
		return "", fmt.Errorf("relayer: empty nonce")
	}
	return resp.Nonce, nil
}

func (r *relayerClient) getRelayPayload(owner, txType string) (relayerPayloadResp, error) {
	u := fmt.Sprintf("%s/relay-payload?address=%s&type=%s", r.baseURL, url.QueryEscape(owner), url.QueryEscape(txType))
	body, err := r.do(http.MethodGet, u, nil, false)
	if err != nil {
		return relayerPayloadResp{}, err
	}
	var resp relayerPayloadResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return relayerPayloadResp{}, err
	}
	if resp.Address == "" || resp.Nonce == "" {
		return relayerPayloadResp{}, fmt.Errorf("relayer: invalid relay payload")
	}
	return resp, nil
}

func (r *relayerClient) submit(req relayerSubmitRequest) (relayerSubmitResp, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return relayerSubmitResp{}, err
	}
	body, err := r.do(http.MethodPost, r.baseURL+"/submit", payload, true)
	if err != nil {
		return relayerSubmitResp{}, err
	}
	var resp relayerSubmitResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return relayerSubmitResp{}, err
	}
	if resp.TransactionID == "" {
		return relayerSubmitResp{}, fmt.Errorf("relayer: empty transaction id")
	}
	return resp, nil
}

func (r *relayerClient) waitConfirmed(txID string, maxPolls int) (string, error) {
	if maxPolls <= 0 {
		maxPolls = 60
	}
	for i := 0; i < maxPolls; i++ {
		u := fmt.Sprintf("%s/transaction?id=%s", r.baseURL, url.QueryEscape(txID))
		body, err := r.do(http.MethodGet, u, nil, false)
		if err != nil {
			return "", err
		}
		var rows []relayerTransaction
		if err := json.Unmarshal(body, &rows); err != nil {
			return "", err
		}
		if len(rows) == 0 {
			time.Sleep(2 * time.Second)
			continue
		}
		state := strings.ToUpper(rows[0].State)
		switch {
		case strings.Contains(state, "CONFIRMED"), strings.Contains(state, "MINED"):
			return rows[0].TransactionHash, nil
		case strings.Contains(state, "FAILED"):
			return "", fmt.Errorf("relayer transaction failed")
		default:
			time.Sleep(2 * time.Second)
		}
	}
	return "", fmt.Errorf("relayer: timeout waiting for confirmation")
}

func (r *relayerClient) do(method, rawURL string, body []byte, authed bool) ([]byte, error) {
	var reader io.Reader
	if len(body) > 0 {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, rawURL, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "nofx-prediction/0.2")
	req.Header.Set("Content-Type", "application/json")
	if authed {
		if !r.hasCreds() {
			return nil, fmt.Errorf("relayer: builder API credentials required")
		}
		path := strings.TrimPrefix(rawURL, r.baseURL)
		if path == "" {
			path = "/"
		}
		ts := time.Now().Unix()
		bodyStr := ""
		if len(body) > 0 {
			bodyStr = string(body)
		}
		sig := buildBuilderHMAC(r.secret, ts, method, path, bodyStr)
		req.Header.Set("POLY_BUILDER_API_KEY", r.builderKey)
		req.Header.Set("POLY_BUILDER_PASSPHRASE", r.passphrase)
		req.Header.Set("POLY_BUILDER_SIGNATURE", sig)
		req.Header.Set("POLY_BUILDER_TIMESTAMP", strconv.FormatInt(ts, 10))
	}
	resp, err := r.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("relayer HTTP %d: %s", resp.StatusCode, truncate(string(respBody), 300))
	}
	return respBody, nil
}

func buildBuilderHMAC(secret string, ts int64, method, path, body string) string {
	msg := fmt.Sprintf("%d%s%s", ts, method, path)
	if body != "" {
		msg += body
	}
	key, _ := base64.StdEncoding.DecodeString(secret)
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(msg))
	sig := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	sig = strings.ReplaceAll(sig, "+", "-")
	sig = strings.ReplaceAll(sig, "/", "_")
	return sig
}
