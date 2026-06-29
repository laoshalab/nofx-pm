package telemetry

import (
	"testing"
	"time"
)

func TestRecordAndList(t *testing.T) {
	ResetForTest()
	RecordTrader("t1", "Test", "cycle", "info", "cycle start", "", nil)
	RecordTrader("t2", "Other", "gamma", "error", "timeout", "", nil)
	RecordTrader("t1", "Test", "execute", "info", "filled", "", map[string]interface{}{"usd": 10.0})

	all := List("", 10, 0)
	if len(all) != 3 {
		t.Fatalf("expected 3 events, got %d", len(all))
	}
	t1only := List("t1", 10, 0)
	if len(t1only) != 2 {
		t.Fatalf("expected 2 t1 events, got %d", len(t1only))
	}
	since := List("", 10, 1)
	if len(since) != 2 {
		t.Fatalf("expected 2 events since id=1, got %d", len(since))
	}
}

func TestRecordHTTPUpdatesConnectivity(t *testing.T) {
	ResetForTest()
	RecordHTTP("gamma", "GET", "https://gamma-api.polymarket.com/markets", 120*time.Millisecond, nil, 200, 1024)
	c := GetConnectivity()
	if !c.GammaOK || c.GammaLatencyMs != 120 {
		t.Fatalf("gamma connectivity: %+v", c)
	}
	RecordHTTP("gamma", "GET", "https://gamma-api.polymarket.com/markets", 5*time.Second, errTest("timeout"), 0, 0)
	c = GetConnectivity()
	if c.GammaOK {
		t.Fatal("expected gamma failure")
	}
}

type errTest string

func (e errTest) Error() string { return string(e) }
