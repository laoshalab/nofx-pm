package engine

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

func (e *PredictionEngine) BuildSystemPrompt() string {
	zh := langZH(e.cfg)
	r := e.cfg.Risk
	var sb strings.Builder

	if zh {
		sb.WriteString("# 你是 Polymarket 预测市场 AI 交易员\n\n")
		sb.WriteString("你交易 YES/NO 份额（价格 0–1），**无杠杆**。最大亏损 = 投入 USDC。\n\n")
		sb.WriteString("## 硬约束（系统强制执行）\n\n")
		fmt.Fprintf(&sb, "- 单笔最大: $%.0f USDC\n", r.MaxOrderUsd)
		fmt.Fprintf(&sb, "- 单市场最大持仓: $%.0f\n", r.MaxPositionMarketUsd)
		fmt.Fprintf(&sb, "- 价格区间: %.2f – %.2f\n", r.MinPrice, r.MaxPrice)
		fmt.Fprintf(&sb, "- 最低置信度: %d\n", r.MinConfidence)
		fmt.Fprintf(&sb, "- 最低 edge: %.1f%%\n\n", r.MinEdgePct)
		sb.WriteString("## 概率思维\n\n")
		sb.WriteString("- 隐含概率 ≈ YES midpoint\n")
		sb.WriteString("- edge_pct = (你的主观概率 − 隐含概率) × 100\n\n")
	} else {
		sb.WriteString("# You are a Polymarket prediction-market AI trader\n\n")
		sb.WriteString("You trade YES/NO outcome shares priced 0–1 with **no leverage**. Max loss = USDC deployed.\n\n")
		sb.WriteString("## Hard constraints (code-enforced)\n\n")
		fmt.Fprintf(&sb, "- Max order: $%.0f USDC\n", r.MaxOrderUsd)
		fmt.Fprintf(&sb, "- Max per-market exposure: $%.0f\n", r.MaxPositionMarketUsd)
		fmt.Fprintf(&sb, "- Price band: %.2f – %.2f\n", r.MinPrice, r.MaxPrice)
		fmt.Fprintf(&sb, "- Min confidence: %d\n", r.MinConfidence)
		fmt.Fprintf(&sb, "- Min edge: %.1f%%\n\n", r.MinEdgePct)
		sb.WriteString("## Probability framing\n\n")
		sb.WriteString("- Implied prob ≈ YES midpoint\n")
		sb.WriteString("- edge_pct = (your estimate − implied) × 100\n\n")
	}

	sb.WriteString("## Output format\n\n")
	sb.WriteString("```xml\n<reasoning>\nYour analysis...\n</reasoning>\n\n")
	sb.WriteString("<decision>\n[ ... JSON array ... ]\n</decision>\n```\n\n")

	sb.WriteString("## Decision JSON schema\n\n")
	sb.WriteString(`[
  {
    "market_slug": "example-market",
    "action": "buy_yes",
    "size_usd": 10,
    "limit_price": 0.58,
    "confidence": 80,
    "edge_pct": 3.5,
    "reasoning": "..."
  }
]` + "\n\n")

	if zh {
		sb.WriteString("合法 action: buy_yes | buy_no | sell | hold | wait | redeem\n")
		sb.WriteString("同一 market 禁止同时 buy_yes 与 buy_no。\n")
	} else {
		sb.WriteString("Valid actions: buy_yes | buy_no | sell | hold | wait | redeem\n")
		sb.WriteString("Never buy_yes and buy_no on the same market in one cycle.\n")
	}
	return sb.String()
}

func (e *PredictionEngine) BuildUserPrompt(ctx *Context) string {
	zh := langZH(e.cfg)
	var sb strings.Builder

	if zh {
		fmt.Fprintf(&sb, "时间: %s | 周期 #%d | 运行 %d 分钟\n\n", ctx.CurrentTime, ctx.CallCount, ctx.RuntimeMinutes)
		if ctx.WalletAvailable {
			fmt.Fprintf(&sb, "账户: 可用 $%.2f | 总权益 $%.2f\n\n", ctx.AvailableUsdc, ctx.TotalEquity)
		} else {
			sb.WriteString("账户: 钱包未连接或余额不可用（只读模式）\n\n")
		}
	} else {
		fmt.Fprintf(&sb, "Time: %s | Cycle #%d | Runtime %d min\n\n", ctx.CurrentTime, ctx.CallCount, ctx.RuntimeMinutes)
		if ctx.WalletAvailable {
			fmt.Fprintf(&sb, "Account: available $%.2f | equity $%.2f\n\n", ctx.AvailableUsdc, ctx.TotalEquity)
		} else {
			sb.WriteString("Account: wallet unavailable (read-only context)\n\n")
		}
	}

	if len(ctx.Positions) > 0 {
		if zh {
			sb.WriteString("## 当前持仓\n\n")
		} else {
			sb.WriteString("## Open positions\n\n")
		}
		for _, p := range ctx.Positions {
			fmt.Fprintf(&sb, "- %s [%s] %.2f shares @ avg %.3f mark %.3f\n",
				p.MarketSlug, p.Outcome, p.Shares, p.AvgCost, p.MidPrice)
		}
		sb.WriteString("\n")
	}

	if len(ctx.SpotQuotes) > 0 {
		if zh {
			sb.WriteString("## 现货参考\n\n")
		} else {
			sb.WriteString("## Spot reference\n\n")
		}
		for _, q := range ctx.SpotQuotes {
			fmt.Fprintf(&sb, "- %s: $%.2f | 5m %+.2f%% | 24h %+.2f%%\n",
				q.Symbol, q.Price, q.Change5mPct, q.Change24hPct)
		}
		sb.WriteString("\n")
	}

	if zh {
		sb.WriteString("## 候选市场\n\n")
	} else {
		sb.WriteString("## Candidate markets\n\n")
	}
	for _, snap := range ctx.CandidateMarkets {
		m := snap.Market
		fmt.Fprintf(&sb, "### %s\n", m.Question)
		fmt.Fprintf(&sb, "slug: %s | liquidity: $%.0f | neg_risk: %v", m.Slug, m.Liquidity, m.NegRisk)
		if m.Volume24h > 0 {
			fmt.Fprintf(&sb, " | 24h vol: $%.0f", m.Volume24h)
		}
		sb.WriteString("\n")
		if !m.EndDate.IsZero() {
			hours := time.Until(m.EndDate).Hours()
			if zh {
				fmt.Fprintf(&sb, "结算: %s (约 %.1f 小时后)\n", m.EndDate.UTC().Format(time.RFC3339), hours)
			} else {
				fmt.Fprintf(&sb, "expires: %s (~%.1f h left)\n", m.EndDate.UTC().Format(time.RFC3339), hours)
			}
		}
		fmt.Fprintf(&sb, "YES mid: %.4f | NO mid: %.4f", snap.YesMid, snap.NoMid)
		if snap.Spread != 0 {
			fmt.Fprintf(&sb, " | spread: %.4f", snap.Spread)
		}
		sb.WriteString("\n")
		if snap.YesBook.BestBid > 0 || snap.YesBook.BestAsk > 0 {
			if zh {
				fmt.Fprintf(&sb, "YES 订单簿: bid %.4f (深度 %.0f) | ask %.4f (深度 %.0f)\n",
					snap.YesBook.BestBid, snap.YesBook.BidDepth, snap.YesBook.BestAsk, snap.YesBook.AskDepth)
			} else {
				fmt.Fprintf(&sb, "YES book: bid %.4f (depth %.0f) | ask %.4f (depth %.0f)\n",
					snap.YesBook.BestBid, snap.YesBook.BidDepth, snap.YesBook.BestAsk, snap.YesBook.AskDepth)
			}
		}
		sb.WriteString("\n")
	}

	if len(ctx.RecentDecisions) > 0 {
		b, _ := json.MarshalIndent(ctx.RecentDecisions, "", "  ")
		if zh {
			sb.WriteString("## 近期决策\n\n")
		} else {
			sb.WriteString("## Recent decisions\n\n")
		}
		sb.Write(b)
		sb.WriteString("\n\n")
	}

	if zh {
		sb.WriteString("请分析以上市场并输出 <reasoning> 与 <decision> JSON。\n")
	} else {
		sb.WriteString("Analyze the markets above and output <reasoning> plus <decision> JSON.\n")
	}
	return sb.String()
}
