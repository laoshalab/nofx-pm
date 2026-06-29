package polymarket

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListOpenOrders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/derive-api-key":
			_ = json.NewEncoder(w).Encode(map[string]string{
				"apiKey":     "test-key",
				"secret":     "dGVzdC1zZWNyZXQ=",
				"passphrase": "pass",
			})
		case "/data/orders":
			if r.Header.Get("POLY_API_KEY") == "" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []map[string]string{
					{
						"id":            "ord-1",
						"status":        "live",
						"asset_id":      "12345",
						"market":        "0xabc",
						"side":          "BUY",
						"price":         "0.55",
						"original_size": "10",
						"size_matched":  "0",
						"created_at":    "2026-01-01T00:00:00Z",
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.ClobURL = srv.URL
	cfg.PrivateKey = "0123456789012345678901234567890123456789012345678901234567890123"
	client := NewClient(cfg)

	orders, err := client.ListOpenOrders()
	if err != nil {
		t.Fatal(err)
	}
	if len(orders) != 1 {
		t.Fatalf("orders: %d", len(orders))
	}
	if orders[0].OrderID != "ord-1" || orders[0].Price != 0.55 {
		t.Fatalf("order: %+v", orders[0])
	}
}
