package engine

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"nofx/logger"
	"nofx/mcp"
	"nofx/prediction/config"
	"nofx/prediction/spot"
	"nofx/prediction/types"
)

var (
	reReasoningTag = regexp.MustCompile(`(?s)<reasoning>(.*?)</reasoning>`)
	reDecisionTag  = regexp.MustCompile(`(?s)<decision>(.*?)</decision>`)
)

// GetPredictionDecisions calls AI and parses prediction decisions.
func GetPredictionDecisions(ctx *Context, client mcp.AIClient, eng *PredictionEngine) (*FullDecision, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context is nil")
	}
	if eng == nil {
		return nil, fmt.Errorf("engine is nil")
	}
	if client == nil {
		return nil, fmt.Errorf("AI client is nil")
	}

	systemPrompt := eng.BuildSystemPrompt()
	userPrompt := eng.BuildUserPrompt(ctx)

	start := time.Now()
	raw, err := client.CallWithMessages(systemPrompt, userPrompt)
	duration := time.Since(start)
	if err != nil {
		return nil, fmt.Errorf("AI call failed: %w", err)
	}

	decisions, cot, err := parsePredictionResponse(raw)
	full := &FullDecision{
		Timestamp:           time.Now(),
		SystemPrompt:        systemPrompt,
		UserPrompt:          userPrompt,
		CoTTrace:            cot,
		Decisions:           decisions,
		RawResponse:         raw,
		AIRequestDurationMs: duration.Milliseconds(),
	}
	if err != nil {
		return full, err
	}
	resolveTokenIDs(decisions, ctx)
	return full, nil
}

func parsePredictionResponse(raw string) ([]types.PredictionDecision, string, error) {
	cot := ""
	if m := reReasoningTag.FindStringSubmatch(raw); len(m) > 1 {
		cot = strings.TrimSpace(m[1])
	}

	jsonStr := ""
	if m := reDecisionTag.FindStringSubmatch(raw); len(m) > 1 {
		jsonStr = strings.TrimSpace(m[1])
	}
	if jsonStr == "" {
		jsonStr = extractJSONArray(raw)
	}
	if jsonStr == "" {
		return nil, cot, fmt.Errorf("no JSON decision found in AI response")
	}

	var decisions []types.PredictionDecision
	if err := json.Unmarshal([]byte(jsonStr), &decisions); err != nil {
		var one types.PredictionDecision
		if err2 := json.Unmarshal([]byte(jsonStr), &one); err2 != nil {
			return nil, cot, fmt.Errorf("parse decision JSON: %w", err)
		}
		decisions = []types.PredictionDecision{one}
	}

	for i := range decisions {
		if err := validateDecision(&decisions[i]); err != nil {
			return decisions, cot, fmt.Errorf("decision[%d]: %w", i, err)
		}
	}
	return decisions, cot, nil
}

// extractJSONArray finds the first balanced [...] block (supports multiple objects).
func extractJSONArray(s string) string {
	start := strings.Index(s, "[")
	if start < 0 {
		return ""
	}
	depth := 0
	inString := false
	escape := false
	for i := start; i < len(s); i++ {
		c := s[i]
		if escape {
			escape = false
			continue
		}
		if c == '\\' && inString {
			escape = true
			continue
		}
		if c == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		switch c {
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}
	return ""
}

func validateDecision(d *types.PredictionDecision) error {
	switch d.Action {
	case types.ActionBuyYes, types.ActionBuyNo, types.ActionSell,
		types.ActionHold, types.ActionWait, types.ActionRedeem:
	default:
		return fmt.Errorf("invalid action %q", d.Action)
	}
	if d.MarketSlug == "" && d.Action != types.ActionWait {
		return fmt.Errorf("market_slug required")
	}
	if d.Action == types.ActionBuyYes || d.Action == types.ActionBuyNo {
		if d.SizeUsd <= 0 {
			return fmt.Errorf("size_usd required for buy")
		}
		if d.LimitPrice <= 0 || d.LimitPrice >= 1 {
			return fmt.Errorf("limit_price must be in (0,1)")
		}
	}
	if d.Action == types.ActionSell {
		if d.SizeUsd <= 0 {
			return fmt.Errorf("size_usd required for sell")
		}
		if d.LimitPrice <= 0 || d.LimitPrice >= 1 {
			return fmt.Errorf("limit_price must be in (0,1)")
		}
	}
	return nil
}

func resolveTokenIDs(decisions []types.PredictionDecision, ctx *Context) {
	slugToTokens := map[string]struct{ yes, no string }{}
	for _, snap := range ctx.CandidateMarkets {
		slugToTokens[snap.Market.Slug] = struct{ yes, no string }{snap.Market.YesTokenID, snap.Market.NoTokenID}
	}
	for i := range decisions {
		d := &decisions[i]
		if d.TokenID != "" {
			continue
		}
		tokens, ok := slugToTokens[d.MarketSlug]
		if !ok {
			continue
		}
		switch d.Action {
		case types.ActionBuyYes:
			if tokens.yes != "" {
				d.TokenID = tokens.yes
			}
		case types.ActionBuyNo:
			if tokens.no != "" {
				d.TokenID = tokens.no
			}
		case types.ActionSell:
			if id := sellTokenFromPositions(d.MarketSlug, ctx.Positions); id != "" {
				d.TokenID = id
			}
		}
	}
}

// sellTokenFromPositions picks the outcome token to sell from open positions.
func sellTokenFromPositions(slug string, positions []types.OutcomePosition) string {
	var held []types.OutcomePosition
	for _, p := range positions {
		if p.MarketSlug == slug && p.Shares > 0 && p.TokenID != "" {
			held = append(held, p)
		}
	}
	switch len(held) {
	case 0:
		return ""
	case 1:
		return held[0].TokenID
	default:
		best := held[0]
		for _, p := range held[1:] {
			if p.Shares > best.Shares {
				best = p
			}
		}
		return best.TokenID
	}
}

// BuildContext assembles trading context from venue + engine config.
func BuildContext(venue types.PredictionVenue, eng *PredictionEngine, callCount, runtimeMin int, recent []types.PredictionDecision) (*Context, error) {
	avail, equity, err := venue.GetCollateral()
	walletOK := err == nil
	if err != nil {
		logger.Warnf("[prediction] wallet/collateral unavailable: %v", err)
		avail, equity = 0, 0
	}
	positions, posErr := venue.GetOutcomePositions()
	if posErr != nil {
		logger.Warnf("[prediction] positions fetch failed: %v", posErr)
		positions = nil
	}
	candidates, err := eng.GetCandidateMarkets()
	if err != nil {
		return nil, err
	}

	spotQuotes := map[string]spot.Quote{}
	if sym := strings.TrimSpace(eng.cfg.Rules.SpotSymbol); sym != "" {
		if q, err := spot.FetchQuote(sym); err == nil {
			spotQuotes[q.Symbol] = q
		} else {
			logger.Warnf("[prediction] spot quote %s: %v", sym, err)
		}
	}

	return &Context{
		CurrentTime:      time.Now().UTC().Format(time.RFC3339),
		RuntimeMinutes:   runtimeMin,
		CallCount:        callCount,
		AvailableUsdc:    avail,
		TotalEquity:      equity,
		WalletAvailable:  walletOK,
		Positions:        positions,
		CandidateMarkets: candidates,
		RecentDecisions:  recent,
		SpotQuotes:       spotQuotes,
	}, nil
}

// Language helper
func langZH(cfg config.StrategyConfig) bool {
	return strings.EqualFold(cfg.Language, "zh") || strings.EqualFold(cfg.Language, "zh-CN")
}
