package telemetry

import (
	"strconv"
	"strings"
	"sync"
	"time"
)

const maxEvents = 300

// Event is one in-memory activity record for the prediction live console.
type Event struct {
	ID         int64                  `json:"id"`
	Timestamp  time.Time              `json:"timestamp"`
	Category   string                 `json:"category"`
	Level      string                 `json:"level"`
	TraderID   string                 `json:"trader_id,omitempty"`
	TraderName string                 `json:"trader_name,omitempty"`
	Message    string                 `json:"message"`
	Detail     string                 `json:"detail,omitempty"`
	DurationMs int64                  `json:"duration_ms,omitempty"`
	Meta       map[string]interface{} `json:"meta,omitempty"`
}

// Connectivity summarizes the latest Polymarket API probe results.
type Connectivity struct {
	GammaOK        bool      `json:"gamma_ok"`
	GammaLatencyMs int64     `json:"gamma_latency_ms"`
	GammaError     string    `json:"gamma_error,omitempty"`
	GammaCheckedAt time.Time `json:"gamma_checked_at,omitempty"`
	ClobOK         bool      `json:"clob_ok"`
	ClobLatencyMs  int64     `json:"clob_latency_ms"`
	ClobError      string    `json:"clob_error,omitempty"`
	ClobCheckedAt  time.Time `json:"clob_checked_at,omitempty"`
}

var (
	mu           sync.RWMutex
	events       []Event
	nextID       int64
	connectivity Connectivity
)

func Record(e Event) {
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now().UTC()
	}
	if e.Level == "" {
		e.Level = "info"
	}
	if e.Category == "" {
		e.Category = "system"
	}

	mu.Lock()
	nextID++
	e.ID = nextID
	events = append(events, e)
	if len(events) > maxEvents {
		events = events[len(events)-maxEvents:]
	}
	mu.Unlock()
}

func RecordTrader(traderID, traderName, category, level, message, detail string, meta map[string]interface{}) {
	Record(Event{
		Category:   category,
		Level:      level,
		TraderID:   traderID,
		TraderName: traderName,
		Message:    message,
		Detail:     detail,
		Meta:       meta,
	})
}

func RecordHTTP(category, method, rawURL string, duration time.Duration, err error, statusCode int, bodyLen int) {
	level := "info"
	msg := "OK"
	detail := ""
	if err != nil {
		level = "error"
		msg = err.Error()
	} else if statusCode >= 400 {
		level = "error"
		msg = "HTTP error"
		detail = strings.TrimSpace(rawURL)
	}

	meta := map[string]interface{}{
		"method":      method,
		"url":         truncate(rawURL, 240),
		"status_code": statusCode,
		"body_bytes":  bodyLen,
	}
	if category == "gamma" || category == "clob" {
		updateConnectivity(category, err == nil && statusCode < 400, duration, msg)
	}

	Record(Event{
		Category:   category,
		Level:      level,
		Message:    formatHTTPMessage(category, method, msg, bodyLen),
		Detail:     detail,
		DurationMs: duration.Milliseconds(),
		Meta:       meta,
	})
}

func updateConnectivity(category string, ok bool, duration time.Duration, errMsg string) {
	mu.Lock()
	defer mu.Unlock()
	now := time.Now().UTC()
	switch category {
	case "gamma":
		connectivity.GammaOK = ok
		connectivity.GammaLatencyMs = duration.Milliseconds()
		connectivity.GammaError = errMsg
		if !ok {
			connectivity.GammaError = errMsg
		} else {
			connectivity.GammaError = ""
		}
		connectivity.GammaCheckedAt = now
	case "clob":
		connectivity.ClobOK = ok
		connectivity.ClobLatencyMs = duration.Milliseconds()
		if !ok {
			connectivity.ClobError = errMsg
		} else {
			connectivity.ClobError = ""
		}
		connectivity.ClobCheckedAt = now
	}
}

func formatHTTPMessage(category, method, msg string, bodyLen int) string {
	prefix := strings.ToUpper(category)
	if method != "" {
		prefix += " " + method
	}
	if msg == "OK" && bodyLen > 0 {
		return prefix + " · " + strconv.Itoa(bodyLen) + " bytes"
	}
	return prefix + " · " + msg
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// List returns recent events, optionally filtered by trader and minimum ID.
func List(traderID string, limit int, sinceID int64) []Event {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	mu.RLock()
	defer mu.RUnlock()

	out := make([]Event, 0, limit)
	for i := len(events) - 1; i >= 0 && len(out) < limit; i-- {
		e := events[i]
		if sinceID > 0 && e.ID <= sinceID {
			continue
		}
		if traderID != "" && e.TraderID != "" && e.TraderID != traderID {
			continue
		}
		out = append(out, e)
	}
	// Return chronological order (oldest first within page).
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

// GetConnectivity returns the latest Polymarket API health snapshot.
func GetConnectivity() Connectivity {
	mu.RLock()
	defer mu.RUnlock()
	return connectivity
}

// ResetForTest clears the in-memory feed (tests only).
func ResetForTest() {
	mu.Lock()
	defer mu.Unlock()
	events = nil
	nextID = 0
	connectivity = Connectivity{}
}
