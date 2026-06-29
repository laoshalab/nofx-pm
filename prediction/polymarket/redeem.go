package polymarket

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"nofx/prediction/types"
)

// FindRedeemablePositions lists held positions in closed/resolved markets.
func (c *Client) FindRedeemablePositions() ([]types.RedeemablePosition, error) {
	if items, err := c.getDataRedeemablePositions(); err == nil && len(items) > 0 {
		return items, nil
	}
	return c.findRedeemableViaGamma()
}

func (c *Client) findRedeemableViaGamma() ([]types.RedeemablePosition, error) {
	positions, err := c.GetOutcomePositions()
	if err != nil {
		return nil, err
	}
	out := make([]types.RedeemablePosition, 0)
	seen := map[string]bool{}
	for _, p := range positions {
		if p.Shares <= 0 || p.MarketSlug == "" {
			continue
		}
		m, err := c.GetMarketBySlug(p.MarketSlug)
		if err != nil || !m.Closed {
			continue
		}
		if m.ConditionID == "" || seen[m.ConditionID] {
			continue
		}
		seen[m.ConditionID] = true
		out = append(out, types.RedeemablePosition{
			ConditionID: m.ConditionID,
			MarketSlug:  m.Slug,
			Outcome:     p.Outcome,
			Shares:      p.Shares,
			NegRisk:     m.NegRisk,
		})
	}
	return out, nil
}

// getDataRedeemablePositions uses Data API redeemable=true (faster for live wallets).
func (c *Client) getDataRedeemablePositions() ([]types.RedeemablePosition, error) {
	user, err := c.funderHex()
	if err != nil {
		return nil, err
	}
	u := fmt.Sprintf("%s/positions?user=%s&redeemable=true&sizeThreshold=0", c.cfg.DataURL, url.QueryEscape(user))
	body, err := c.getJSON(u)
	if err != nil {
		return nil, err
	}
	var rows []dataPosition
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, err
	}
	out := make([]types.RedeemablePosition, 0)
	seen := map[string]bool{}
	for _, row := range rows {
		if row.Size <= 0 {
			continue
		}
		cid := strings.TrimSpace(row.ConditionID)
		negRisk := false
		if row.Slug != "" {
			if m, err := c.GetMarketBySlug(row.Slug); err == nil && m != nil {
				negRisk = m.NegRisk
				if cid == "" {
					cid = m.ConditionID
				}
			}
		}
		if cid == "" || seen[cid] {
			continue
		}
		seen[cid] = true
		outcome := strings.ToUpper(strings.TrimSpace(row.Outcome))
		if outcome == "" {
			outcome = "YES"
		}
		out = append(out, types.RedeemablePosition{
			ConditionID: cid,
			MarketSlug:  row.Slug,
			Outcome:     outcome,
			Shares:      row.Size,
			NegRisk:     negRisk,
		})
	}
	return out, nil
}

// RedeemCondition claims collateral for a resolved market condition.
// Preview logs only; Live uses relayer (Safe/Proxy) or direct EOA on-chain redeem.
func (c *Client) RedeemCondition(conditionID string, negRisk bool) (*types.RedeemResult, error) {
	conditionID = strings.TrimSpace(conditionID)
	if conditionID == "" {
		return nil, fmt.Errorf("empty condition_id")
	}
	result := &types.RedeemResult{
		ConditionID: conditionID,
		Status:      "preview",
	}
	if c.cfg.PreviewMode {
		return result, nil
	}
	if c.cfg.PrivateKey == "" {
		return nil, fmt.Errorf("polymarket: private key required for redeem")
	}
	if !c.liveRedeemReady() {
		return nil, fmt.Errorf("polymarket: live redeem not configured — set POLYMARKET_BUILDER_* and POLYGON_RPC_URL")
	}
	return c.redeemConditionLive(conditionID, negRisk)
}
