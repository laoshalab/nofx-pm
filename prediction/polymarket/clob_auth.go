package polymarket

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

type apiCredentials struct {
	APIKey     string `json:"apiKey"`
	Secret     string `json:"secret"`
	Passphrase string `json:"passphrase"`
}

func (c *Client) ensureCredentials() error {
	if c.creds != nil {
		return nil
	}
	if c.cfg.PrivateKey == "" {
		return fmt.Errorf("polymarket: private key required for authenticated CLOB")
	}
	creds, err := c.deriveOrCreateAPIKey()
	if err != nil {
		return err
	}
	c.creds = creds
	return nil
}

func (c *Client) l1AuthHeaders(w *wallet) (map[string]string, error) {
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	nonce := uint64(0)
	sig, err := signClobAuth(w, c.cfg.ChainID, ts, nonce)
	if err != nil {
		return nil, err
	}
	return l1Headers(w.address.Hex(), ts, nonce, sig), nil
}

func (c *Client) deriveOrCreateAPIKey() (*apiCredentials, error) {
	w, err := loadWallet(c.cfg.PrivateKey)
	if err != nil {
		return nil, err
	}

	headers, err := c.l1AuthHeaders(w)
	if err != nil {
		return nil, err
	}

	nonce := uint64(0)
	deriveURL := fmt.Sprintf("%s/auth/derive-api-key?nonce=%d", c.cfg.ClobURL, nonce)
	req, err := http.NewRequest(http.MethodGet, deriveURL, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 400 {
		var creds apiCredentials
		if err := json.NewDecoder(resp.Body).Decode(&creds); err != nil {
			return nil, err
		}
		if creds.APIKey != "" {
			return &creds, nil
		}
	}

	headers, err = c.l1AuthHeaders(w)
	if err != nil {
		return nil, err
	}
	createURL := c.cfg.ClobURL + "/auth/api-key"
	req, err = http.NewRequest(http.MethodPost, createURL, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err = c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("create api key: HTTP %d", resp.StatusCode)
	}
	var creds apiCredentials
	if err := json.NewDecoder(resp.Body).Decode(&creds); err != nil {
		return nil, err
	}
	return &creds, nil
}

func l1Headers(address, timestamp string, nonce uint64, signature string) map[string]string {
	return map[string]string{
		"POLY_ADDRESS":   address,
		"POLY_SIGNATURE": signature,
		"POLY_TIMESTAMP": timestamp,
		"POLY_NONCE":     strconv.FormatUint(nonce, 10),
	}
}

func (c *Client) applyL2Headers(req *http.Request, method, path string, body []byte) error {
	if err := c.ensureCredentials(); err != nil {
		return err
	}
	addr, err := c.signerAddress()
	if err != nil {
		return err
	}
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	sig := buildL2Signature(c.creds.Secret, ts, method, path, body)
	req.Header.Set("POLY_ADDRESS", addr.Hex())
	req.Header.Set("POLY_SIGNATURE", sig)
	req.Header.Set("POLY_TIMESTAMP", ts)
	req.Header.Set("POLY_API_KEY", c.creds.APIKey)
	req.Header.Set("POLY_PASSPHRASE", c.creds.Passphrase)
	return nil
}

func buildL2Signature(secretB64, timestamp, method, path string, body []byte) string {
	secret, err := base64.URLEncoding.DecodeString(secretB64)
	if err != nil {
		secret, _ = base64.StdEncoding.DecodeString(secretB64)
	}
	msg := timestamp + method + path
	if len(body) > 0 {
		msg += string(body)
	}
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(msg))
	return base64.URLEncoding.EncodeToString(mac.Sum(nil))
}
