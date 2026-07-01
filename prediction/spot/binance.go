package spot

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const binanceSpotURL = "https://api.binance.com"

// Quote is a cached spot price snapshot with short-horizon change.
type Quote struct {
	Symbol       string  `json:"symbol"`
	Price        float64 `json:"price"`
	Change5mPct  float64 `json:"change_5m_pct"`
	Change24hPct float64 `json:"change_24h_pct"`
	FetchedAt    time.Time
}

var (
	cacheMu sync.RWMutex
	cache   = map[string]cachedQuote{}
	cacheTTL = 30 * time.Second
	httpClient = &http.Client{Timeout: 8 * time.Second}
)

type cachedQuote struct {
	quote Quote
	at    time.Time
}

// FetchQuote returns Binance spot price and 5m / 24h change for symbol (e.g. BTCUSDT).
func FetchQuote(symbol string) (Quote, error) {
	symbol = normalizeSymbol(symbol)
	if symbol == "" {
		return Quote{}, fmt.Errorf("empty symbol")
	}

	cacheMu.RLock()
	if c, ok := cache[symbol]; ok && time.Since(c.at) < cacheTTL {
		q := c.quote
		cacheMu.RUnlock()
		return q, nil
	}
	cacheMu.RUnlock()

	price, chg24, err := fetchTicker24h(symbol)
	if err != nil {
		return Quote{}, err
	}
	chg5m, err := fetchChange5m(symbol)
	if err != nil {
		chg5m = 0
	}

	q := Quote{
		Symbol:       symbol,
		Price:        price,
		Change5mPct:  chg5m,
		Change24hPct: chg24,
		FetchedAt:    time.Now().UTC(),
	}

	cacheMu.Lock()
	cache[symbol] = cachedQuote{quote: q, at: time.Now()}
	cacheMu.Unlock()
	return q, nil
}

// NormalizeSymbol uppercases and ensures a USDT suffix (e.g. BTC → BTCUSDT).
func NormalizeSymbol(symbol string) string {
	return normalizeSymbol(symbol)
}

func normalizeSymbol(symbol string) string {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if symbol == "" {
		return ""
	}
	if !strings.HasSuffix(symbol, "USDT") {
		symbol += "USDT"
	}
	return symbol
}

func fetchTicker24h(symbol string) (price, changePct float64, err error) {
	url := fmt.Sprintf("%s/api/v3/ticker/24hr?symbol=%s", binanceSpotURL, symbol)
	body, err := getJSON(url)
	if err != nil {
		return 0, 0, err
	}
	var t struct {
		LastPrice          string `json:"lastPrice"`
		PriceChangePercent string `json:"priceChangePercent"`
	}
	if err := json.Unmarshal(body, &t); err != nil {
		return 0, 0, err
	}
	price, err = strconv.ParseFloat(t.LastPrice, 64)
	if err != nil {
		return 0, 0, err
	}
	changePct, _ = strconv.ParseFloat(t.PriceChangePercent, 64)
	return price, changePct, nil
}

func fetchChange5m(symbol string) (float64, error) {
	url := fmt.Sprintf("%s/api/v3/klines?symbol=%s&interval=5m&limit=2", binanceSpotURL, symbol)
	body, err := getJSON(url)
	if err != nil {
		return 0, err
	}
	var rows [][]json.RawMessage
	if err := json.Unmarshal(body, &rows); err != nil {
		return 0, err
	}
	if len(rows) < 2 {
		return 0, fmt.Errorf("insufficient klines")
	}
	open, err := parseKlineClose(rows[0])
	if err != nil {
		return 0, err
	}
	close, err := parseKlineClose(rows[1])
	if err != nil {
		return 0, err
	}
	if open <= 0 {
		return 0, fmt.Errorf("invalid open")
	}
	return ((close - open) / open) * 100, nil
}

func parseKlineClose(row []json.RawMessage) (float64, error) {
	if len(row) < 5 {
		return 0, fmt.Errorf("short kline row")
	}
	var s string
	if err := json.Unmarshal(row[1], &s); err != nil {
		return 0, err
	}
	return strconv.ParseFloat(s, 64)
}

func getJSON(url string) ([]byte, error) {
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("binance http %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 256*1024))
}
