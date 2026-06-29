package polymarket

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetOrderBook(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/book" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"bids": []map[string]string{{"price": "0.44", "size": "100"}},
			"asks": []map[string]string{{"price": "0.46", "size": "200"}},
			"mid":  "0.45",
		})
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.ClobURL = srv.URL
	client := NewClient(cfg)
	book, err := client.GetOrderBook("12345")
	if err != nil {
		t.Fatal(err)
	}
	if book.Mid != 0.45 {
		t.Fatalf("mid: %f", book.Mid)
	}
	if len(book.Bids) != 1 || len(book.Asks) != 1 {
		t.Fatalf("levels: bids=%d asks=%d", len(book.Bids), len(book.Asks))
	}
}
