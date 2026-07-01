package trader

import (
	"fmt"
	"strings"

	"nofx/prediction/types"
)

// TelegramNotify sends proactive Telegram alerts when wired by telegram.Start.
var TelegramNotify func(text string)

func (pt *PredictionTrader) notifyExecution(out types.ExecutionOutcome) {
	if TelegramNotify == nil {
		return
	}
	switch out.Status {
	case "filled", "preview", "redeem", "error", "rejected", "posted":
	default:
		return
	}
	text := formatExecutionTelegram(pt.name, out)
	if text == "" {
		return
	}
	go TelegramNotify(text)
}

func (pt *PredictionTrader) notifyCycleFailed(err error) {
	if TelegramNotify == nil || err == nil {
		return
	}
	go TelegramNotify(fmt.Sprintf(
		"🔮 *Prediction · Cycle Failed*\nTrader: %s\nError: %s",
		pt.name, err.Error(),
	))
}

func formatExecutionTelegram(traderName string, out types.ExecutionOutcome) string {
	emoji := map[string]string{
		"filled":   "✅",
		"preview":  "📝",
		"redeem":   "💰",
		"error":    "❌",
		"rejected": "⛔",
		"posted":   "📤",
	}[out.Status]
	if emoji == "" {
		emoji = "🔮"
	}
	title := out.Status
	var sb strings.Builder
	fmt.Fprintf(&sb, "%s *Prediction · %s*\n", emoji, title)
	fmt.Fprintf(&sb, "Trader: %s\n", traderName)
	if out.MarketSlug != "" {
		fmt.Fprintf(&sb, "Market: `%s`\n", out.MarketSlug)
	}
	if out.Action != "" {
		fmt.Fprintf(&sb, "Action: %s\n", out.Action)
	}
	if out.FillUsd > 0 {
		fmt.Fprintf(&sb, "Fill: $%.2f\n", out.FillUsd)
	}
	if out.OrderID != "" {
		fmt.Fprintf(&sb, "Order: `%s`\n", out.OrderID)
	}
	if out.Message != "" {
		fmt.Fprintf(&sb, "\n%s", out.Message)
	}
	return sb.String()
}
