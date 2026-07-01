package trader

import (
	"testing"

	"nofx/prediction/risk"
)

func TestSyncOrdersAfterCycle_SkipsPreview(t *testing.T) {
	pt := &PredictionTrader{
		riskGate: risk.NewGate(risk.Config{PreviewMode: true}),
		venue:    stubVenue{venueID: "polymarket"},
	}
	pt.syncOrdersAfterCycle() // must not panic
}

func TestSyncOrderRecordsFromCLOB_NilStoreNoOp(t *testing.T) {
	var filled float64
	SyncOrderRecordsFromCLOB(nil, "t1", stubVenue{}, func(usd float64) { filled += usd })
	if filled != 0 {
		t.Fatalf("expected no fill, got %v", filled)
	}
}
