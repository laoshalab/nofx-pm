package telegram

import (
	"fmt"
	"strings"

	"nofx/manager"
	"nofx/store"
)

func predictionStatusMsg(st *store.Store, pm *manager.PredictionManager, userID, lang string) string {
	rows, err := st.Prediction().ListTraders(userID)
	if err != nil || len(rows) == 0 {
		if lang == "zh" {
			return "🔮 *预测市场*\n\n暂无 Prediction Trader。\n在 Web `/prediction` 页面创建。"
		}
		return "🔮 *Prediction Markets*\n\nNo prediction traders yet.\nCreate one at `/prediction` in the web UI."
	}

	var sb strings.Builder
	if lang == "zh" {
		sb.WriteString(fmt.Sprintf("🔮 *预测市场* (%d)\n\n", len(rows)))
	} else {
		sb.WriteString(fmt.Sprintf("🔮 *Prediction Markets* (%d)\n\n", len(rows)))
	}

	for _, row := range rows {
		venue := manager.BuildVenueForTrader(st, row)
		avail, equity, _ := venue.GetCollateral()
		_ = avail
		mode := row.EffectiveTradingMode()
		status := "○"
		statusLabel := "stopped"
		if row.IsRunning {
			status = "●"
			statusLabel = "running"
		}
		if lang == "zh" {
			if row.IsRunning {
				statusLabel = "运行中"
			} else {
				statusLabel = "已停止"
			}
		}
		sb.WriteString(fmt.Sprintf("%s *%s* — %s · %s · $%.2f\n",
			status, row.Name, statusLabel, mode, equity))
		if pm != nil {
			if rs, ok := pm.RuntimeStatusFor(row.ID); ok && rs.CycleCount > 0 {
				if lang == "zh" {
					sb.WriteString(fmt.Sprintf("  周期 #%d · 日成交 $%.0f\n", rs.CycleCount, rs.DailyVolumeUsd))
				} else {
					sb.WriteString(fmt.Sprintf("  cycle #%d · 24h vol $%.0f\n", rs.CycleCount, rs.DailyVolumeUsd))
				}
			}
		}
	}

	if lang == "zh" {
		sb.WriteString("\n/prediction\\_positions 查看持仓")
	} else {
		sb.WriteString("\n/prediction\\_positions for open positions")
	}
	return sb.String()
}

func predictionPositionsMsg(st *store.Store, userID, lang string) string {
	rows, err := st.Prediction().ListTraders(userID)
	if err != nil || len(rows) == 0 {
		if lang == "zh" {
			return "🔮 *预测持仓*\n\n暂无 Trader。"
		}
		return "🔮 *Prediction Positions*\n\nNo traders."
	}

	var sb strings.Builder
	if lang == "zh" {
		sb.WriteString("🔮 *预测持仓*\n\n")
	} else {
		sb.WriteString("🔮 *Prediction Positions*\n\n")
	}

	total := 0
	for _, row := range rows {
		venue := manager.BuildVenueForTrader(st, row)
		positions, _ := venue.GetOutcomePositions()
		if len(positions) == 0 {
			continue
		}
		sb.WriteString(fmt.Sprintf("*%s*\n", row.Name))
		for _, p := range positions {
			if p.Shares <= 0 {
				continue
			}
			total++
			mark := p.MidPrice
			if mark <= 0 {
				mark = p.AvgCost
			}
			sb.WriteString(fmt.Sprintf("  • %s [%s] %.1f @ %.3f mark %.3f\n",
				p.MarketSlug, p.Outcome, p.Shares, p.AvgCost, mark))
		}
		sb.WriteString("\n")
	}

	if total == 0 {
		if lang == "zh" {
			return "🔮 *预测持仓*\n\n当前无 YES/NO 持仓。"
		}
		return "🔮 *Prediction Positions*\n\nNo open YES/NO positions."
	}
	return strings.TrimSpace(sb.String())
}

func predictionHelpMsg(lang string) string {
	if lang == "zh" {
		return `🔮 *预测市场命令*

/prediction — Trader 状态与权益
/prediction\\_status — 同上
/prediction\\_positions — YES/NO 持仓`
	}
	return `🔮 *Prediction commands*

/prediction — trader status & equity
/prediction\\_status — same as above
/prediction\\_positions — YES/NO holdings`
}
