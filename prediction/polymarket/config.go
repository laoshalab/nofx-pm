package polymarket

import (
	"os"
	"strconv"
	"strings"
)

// Config holds Polymarket API endpoints and wallet settings (M1+).
type Config struct {
	GammaURL      string
	ClobURL       string
	DataURL       string
	ChainID       int64
	PrivateKey    string
	ProxyAddress  string // funder / maker when using proxy or Safe
	SignatureType int    // 0=EOA, 1=proxy, 2=Safe, 3=deposit wallet
	PreviewMode   bool   // when true, sign but do not POST orders

	// Live redeem (chain / relayer)
	PolygonRPCURL     string
	RelayerURL        string
	BuilderAPIKey     string
	BuilderSecret     string
	BuilderPassphrase string
	RedeemUSDCe       bool // when true redeem to USDC.e via legacy CTF; default pUSD adapters
}

// V2 Exchange verifying contracts (Polygon mainnet).
const (
	ExchangeV2Standard = "0xE111180000d2663C0091e4f400237545B87B996B"
	ExchangeV2NegRisk  = "0xe2222d279d744050d28e00520010520000310F59"
)

// DefaultConfig returns public read-only endpoints (no wallet required).
func DefaultConfig() Config {
	return MergeEnvConfig(Config{
		GammaURL:      "https://gamma-api.polymarket.com",
		ClobURL:       "https://clob.polymarket.com",
		DataURL:       "https://data-api.polymarket.com",
		ChainID:       137,
		SignatureType: 0,
		PreviewMode:   true,
		RelayerURL:    defaultRelayerURL,
	})
}

// MergeEnvConfig fills chain/redeem settings from environment when unset.
func MergeEnvConfig(cfg Config) Config {
	if strings.TrimSpace(cfg.RelayerURL) == "" {
		if v := strings.TrimSpace(os.Getenv("POLYMARKET_RELAYER_URL")); v != "" {
			cfg.RelayerURL = v
		} else {
			cfg.RelayerURL = defaultRelayerURL
		}
	}
	if strings.TrimSpace(cfg.PolygonRPCURL) == "" {
		if v := strings.TrimSpace(os.Getenv("POLYGON_RPC_URL")); v != "" {
			cfg.PolygonRPCURL = v
		}
	}
	if strings.TrimSpace(cfg.BuilderAPIKey) == "" {
		cfg.BuilderAPIKey = strings.TrimSpace(os.Getenv("POLYMARKET_BUILDER_API_KEY"))
	}
	if strings.TrimSpace(cfg.BuilderSecret) == "" {
		cfg.BuilderSecret = strings.TrimSpace(os.Getenv("POLYMARKET_BUILDER_SECRET"))
	}
	if strings.TrimSpace(cfg.BuilderPassphrase) == "" {
		cfg.BuilderPassphrase = strings.TrimSpace(os.Getenv("POLYMARKET_BUILDER_PASSPHRASE"))
	}
	if v := strings.TrimSpace(os.Getenv("POLYMARKET_REDEEM_USDCE")); v != "" {
		cfg.RedeemUSDCe = strings.EqualFold(v, "true") || v == "1"
	}
	if strings.TrimSpace(cfg.PrivateKey) == "" {
		cfg.PrivateKey = EnvPrivateKey()
	}
	if strings.TrimSpace(cfg.ProxyAddress) == "" {
		if v := strings.TrimSpace(os.Getenv("POLYMARKET_PROXY_ADDRESS")); v != "" {
			cfg.ProxyAddress = v
		}
	}
	if st, ok := EnvSignatureType(); ok && cfg.SignatureType == 0 {
		cfg.SignatureType = st
	}
	return cfg
}

// EnvSignatureType reads POLYMARKET_SIGNATURE_TYPE (0–3). Second value is false when unset/invalid.
func EnvSignatureType() (int, bool) {
	v := strings.TrimSpace(os.Getenv("POLYMARKET_SIGNATURE_TYPE"))
	if v == "" {
		return 0, false
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 || n > 3 {
		return 0, false
	}
	return n, true
}

// ResolveSignatureType picks CLOB signature type: row > env > proxy wallet (2) > EOA (0).
func ResolveSignatureType(rowSignatureType int, proxyAddress string) int {
	if rowSignatureType != 0 {
		return rowSignatureType
	}
	if st, ok := EnvSignatureType(); ok {
		return st
	}
	if strings.TrimSpace(proxyAddress) != "" {
		return 2
	}
	return 0
}

// EnvPrivateKey returns POLYMARKET_PRIVATE_KEY from the environment (never logged).
func EnvPrivateKey() string {
	return strings.TrimSpace(os.Getenv("POLYMARKET_PRIVATE_KEY"))
}

// ServerWalletConfigured reports whether a server-side Polymarket key is available.
func ServerWalletConfigured() bool {
	return EnvPrivateKey() != ""
}

// SignatureTypeForOrder returns the CLOB signature type for order signing.
// Without a proxy/funder address, EOA (0) is always used.
func (cfg Config) SignatureTypeForOrder() int {
	if strings.TrimSpace(cfg.ProxyAddress) == "" {
		return 0
	}
	if cfg.SignatureType < 0 || cfg.SignatureType > 3 {
		return 0
	}
	return cfg.SignatureType
}

// LiveRedeemReady reports whether this client can execute live redeem.
func (c *Client) LiveRedeemReady() bool {
	return c.liveRedeemReady()
}
