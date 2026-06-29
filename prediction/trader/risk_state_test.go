package trader

import (
	"testing"

	"nofx/store"
)

func TestSyncDailyVolumeFromStore(t *testing.T) {
	st, err := store.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	const traderID = "t1"
	day := utcVolumeDay()
	if err := st.Prediction().SaveRiskState(traderID, day, 25); err != nil {
		t.Fatal(err)
	}

	pt := &PredictionTrader{traderDBID: traderID, st: st}
	pt.syncDailyVolumeFromStore()
	if pt.dailyVolume != 25 {
		t.Fatalf("volume: %f", pt.dailyVolume)
	}

	pt.recordFillVolume(10)
	vol, gotDay, _ := st.Prediction().LoadRiskState(traderID)
	if gotDay != day || vol != 35 {
		t.Fatalf("persisted vol=%f day=%s", vol, gotDay)
	}
}
