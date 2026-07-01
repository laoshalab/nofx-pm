package trader

import (
	"strings"

	"nofx/prediction/types"
	"nofx/store"
)

// SyncOrderRecordsFromCLOB refreshes local order rows from the CLOB and imports open orders.
// onFill is called when a row transitions to filled (for risk volume accounting).
func SyncOrderRecordsFromCLOB(st *store.Store, traderID string, venue types.PredictionVenue, onFill func(usd float64)) {
	if st == nil || traderID == "" || venue == nil {
		return
	}
	records, _ := st.Prediction().ListOrderRecords(traderID, 100)
	for _, rec := range records {
		if rec.IsPreview || rec.OrderID == "" {
			continue
		}
		prev := strings.ToLower(rec.Status)
		if prev == "filled" || prev == "cancelled" || prev == "canceled" {
			continue
		}
		orderSt, err := venue.GetOrderStatus(rec.OrderID)
		if err != nil || orderSt == nil {
			continue
		}
		newStatus := strings.ToLower(orderSt.Status)
		if orderSt.SizeMatched > 0 && orderSt.OriginalSize > 0 && orderSt.SizeMatched >= orderSt.OriginalSize {
			newStatus = "filled"
		}
		if newStatus == prev {
			continue
		}
		_ = st.Prediction().UpdateOrderStatus(traderID, rec.OrderID, newStatus)
		if newStatus == "filled" && onFill != nil {
			fillUsd := rec.Price * rec.Size
			if orderSt.SizeMatched > 0 {
				fillUsd = rec.Price * orderSt.SizeMatched
			}
			onFill(fillUsd)
		}
	}

	if openOrders, err := venue.ListOpenOrders(); err == nil {
		for _, o := range openOrders {
			_ = st.Prediction().SaveOrderRecord(&store.PredictionOrderRecordDB{
				TraderID: traderID,
				OrderID:  o.OrderID,
				TokenID:  o.TokenID,
				Side:     o.Side,
				Price:    o.Price,
				Size:     o.OriginalSize,
				Status:   o.Status,
			})
		}
	}
}

func (pt *PredictionTrader) syncOrdersAfterCycle() {
	if pt.riskGate.Config().PreviewMode || pt.st == nil || pt.traderDBID == "" {
		return
	}
	SyncOrderRecordsFromCLOB(pt.st, pt.traderDBID, pt.venue, pt.recordFillVolume)
}
