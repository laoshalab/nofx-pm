package polymarket

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"nofx/prediction/types"
)

// Live mainnet acceptance (≤$1 notional, place + cancel).
//
// Local:
//
//	POLYMARKET_LIVE_E2E=1 POLYMARKET_PRIVATE_KEY=0x... \
//	  go test -v -count=1 ./prediction/polymarket -run TestLiveMainnetE2E -timeout 5m
//
// Optional: POLYMARKET_PROXY_ADDRESS, POLYMARKET_E2E_MARKET_SLUG, POLYMARKET_E2E_MAX_USD,
// POLYMARKET_SIGNATURE_TYPE (0–3).

func TestLiveMainnetE2E(t *testing.T) {
	skipIfNoLiveE2E(t)
	cfg := liveE2EConfig()
	readOnly := NewClient(MergeEnvConfig(DefaultConfig()))

	t.Run("PreviewSignNoPost", func(t *testing.T) {
		market, err := pickE2EMarket(readOnly)
		if err != nil {
			t.Fatalf("pick market: %v", err)
		}
		previewCfg := cfg
		previewCfg.PreviewMode = true
		client := NewClient(previewCfg)
		price, size := e2eOrderParams(client, market, maxE2ENotionalUSD())
		result, err := client.PlaceLimitOrder(types.LimitOrderReq{
			TokenID: market.YesTokenID,
			Side:    "BUY",
			Price:   price,
			Size:    size,
			NegRisk: market.NegRisk,
		})
		if err != nil {
			t.Fatalf("preview place: %v", err)
		}
		if result.OrderID == "" || !strings.HasPrefix(result.OrderID, "preview-") {
			t.Fatalf("expected preview order id, got %+v", result)
		}
		if result.PreviewWire == "" {
			t.Fatal("expected preview_wire payload")
		}
		t.Logf("preview ok market=%s price=%.4f size=%.0f notional<=$%.2f", market.Slug, price, size, price*size)
	})

	t.Run("PlaceCancelLive", func(t *testing.T) {
		cfg.PreviewMode = false
		client := NewClient(cfg)
		market, err := pickE2EMarket(readOnly)
		if err != nil {
			t.Fatalf("pick market: %v", err)
		}
		price, size := e2eOrderParams(client, market, maxE2ENotionalUSD())
		notional := price * size
		if notional > maxE2ENotionalUSD()+1e-6 {
			t.Fatalf("notional $%.4f exceeds cap $%.2f", notional, maxE2ENotionalUSD())
		}

		result, err := client.PlaceLimitOrder(types.LimitOrderReq{
			TokenID: market.YesTokenID,
			Side:    "BUY",
			Price:   price,
			Size:    size,
			NegRisk: market.NegRisk,
		})
		if err != nil {
			t.Fatalf("live place: %v", err)
		}
		if result.OrderID == "" || strings.HasPrefix(result.OrderID, "preview-") {
			t.Fatalf("expected live order id, got %+v err=%q", result, result.Error)
		}
		if result.Error != "" {
			t.Fatalf("order error: %s", result.Error)
		}
		t.Logf("posted order_id=%s market=%s price=%.4f size=%.0f", result.OrderID, market.Slug, price, size)

		status, err := client.GetOrderStatus(result.OrderID)
		if err != nil {
			t.Fatalf("get status: %v", err)
		}
		if status.SizeMatched > 0 {
			t.Logf("warning: order partially matched before cancel (matched=%.4f)", status.SizeMatched)
		}

		if err := client.CancelOrder(result.OrderID); err != nil {
			t.Fatalf("cancel: %v", err)
		}

		deadline := time.Now().Add(45 * time.Second)
		for time.Now().Before(deadline) {
			st, err := client.GetOrderStatus(result.OrderID)
			if err != nil {
				t.Fatalf("poll status: %v", err)
			}
			s := strings.ToLower(st.Status)
			if st.Terminal || s == "cancelled" || s == "canceled" {
				t.Logf("terminal status=%s matched=%.4f", st.Status, st.SizeMatched)
				return
			}
			time.Sleep(2 * time.Second)
		}
		t.Fatal("order not cancelled within timeout")
	})
}

func skipIfNoLiveE2E(t *testing.T) {
	t.Helper()
	if os.Getenv("POLYMARKET_LIVE_E2E") != "1" {
		t.Skip("set POLYMARKET_LIVE_E2E=1 to run live mainnet E2E")
	}
	if strings.TrimSpace(os.Getenv("POLYMARKET_PRIVATE_KEY")) == "" {
		t.Skip("POLYMARKET_PRIVATE_KEY required for live E2E")
	}
}

func liveE2EConfig() Config {
	cfg := MergeEnvConfig(DefaultConfig())
	cfg.PrivateKey = strings.TrimSpace(os.Getenv("POLYMARKET_PRIVATE_KEY"))
	cfg.ProxyAddress = strings.TrimSpace(os.Getenv("POLYMARKET_PROXY_ADDRESS"))
	cfg.PreviewMode = false
	if v := strings.TrimSpace(os.Getenv("POLYMARKET_SIGNATURE_TYPE")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.SignatureType = n
		}
	}
	return cfg
}

func maxE2ENotionalUSD() float64 {
	if v := strings.TrimSpace(os.Getenv("POLYMARKET_E2E_MAX_USD")); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			return f
		}
	}
	return 1.0
}

func pickE2EMarket(client *Client) (*types.Market, error) {
	if slug := strings.TrimSpace(os.Getenv("POLYMARKET_E2E_MARKET_SLUG")); slug != "" {
		return client.GetMarketBySlug(slug)
	}
	closed := false
	markets, err := client.SearchMarkets(types.MarketFilter{
		Tag:      "crypto",
		Keywords: []string{"up or down"},
		Closed:   &closed,
		Limit:    30,
	})
	if err != nil {
		return nil, err
	}
	for _, m := range markets {
		if m.Closed || m.YesTokenID == "" {
			continue
		}
		if m.Liquidity >= 500 {
			cp := m
			return &cp, nil
		}
	}
	for _, m := range markets {
		if !m.Closed && m.YesTokenID != "" {
			cp := m
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("no suitable open crypto market for E2E (set POLYMARKET_E2E_MARKET_SLUG)")
}

// e2eOrderParams returns a deep OTM bid unlikely to fill; notional = price * size ≤ maxUSD.
func e2eOrderParams(client *Client, market *types.Market, maxUSD float64) (price, size float64) {
	price = 0.01
	if meta, err := client.GetBookMeta(market.YesTokenID); err == nil && meta.TickSize != "" {
		if tick, err := strconv.ParseFloat(meta.TickSize, 64); err == nil && tick > 0 {
			price = tick
		}
	}
	size = math.Floor(maxUSD / price)
	if size < 1 {
		size = 1
	}
	return price, size
}
