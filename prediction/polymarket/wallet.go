package polymarket

import (
	"encoding/json"
	"fmt"
	"net/url"
	"nofx/prediction/types"
	"strings"
)

type dataPosition struct {
	Asset       string  `json:"asset"`
	ConditionID string  `json:"conditionId"`
	Size        float64 `json:"size"`
	AvgPrice    float64 `json:"avgPrice"`
	Title       string  `json:"title"`
	Slug        string  `json:"slug"`
	Outcome     string  `json:"outcome"`
	CurPrice    float64 `json:"curPrice"`
}

// GetCollateral returns available USDC and total equity via Data API.
func (c *Client) GetCollateral() (availableUsdc, totalEquity float64, err error) {
	user, err := c.funderHex()
	if err != nil {
		return 0, 0, err
	}
	u := fmt.Sprintf("%s/value?user=%s", c.cfg.DataURL, url.QueryEscape(user))
	body, err := c.getJSON(u)
	if err != nil {
		return 0, 0, err
	}
	var resp []struct {
		Value float64 `json:"value"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		var single struct {
			Value float64 `json:"value"`
		}
		if err2 := json.Unmarshal(body, &single); err2 != nil {
			return 0, 0, err
		}
		totalEquity = single.Value
	} else {
		for _, row := range resp {
			totalEquity += row.Value
		}
	}

	positionValue := c.positionNotional(user)
	availableUsdc = totalEquity - positionValue
	if availableUsdc < 0 {
		availableUsdc = 0
	}
	return availableUsdc, totalEquity, nil
}

func (c *Client) positionNotional(user string) float64 {
	u := fmt.Sprintf("%s/positions?user=%s", c.cfg.DataURL, url.QueryEscape(user))
	body, err := c.getJSON(u)
	if err != nil {
		return 0
	}
	var rows []dataPosition
	if err := json.Unmarshal(body, &rows); err != nil {
		return 0
	}
	var total float64
	for _, row := range rows {
		if row.Size <= 0 {
			continue
		}
		price := row.CurPrice
		if price <= 0 {
			price = row.AvgPrice
		}
		if price <= 0 {
			continue
		}
		total += row.Size * price
	}
	return total
}

// GetOutcomePositions lists open YES/NO positions for the configured wallet.
func (c *Client) GetOutcomePositions() ([]types.OutcomePosition, error) {
	user, err := c.funderHex()
	if err != nil {
		return nil, err
	}
	u := fmt.Sprintf("%s/positions?user=%s", c.cfg.DataURL, url.QueryEscape(user))
	body, err := c.getJSON(u)
	if err != nil {
		return nil, err
	}
	var rows []dataPosition
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, err
	}
	out := make([]types.OutcomePosition, 0, len(rows))
	for _, row := range rows {
		if row.Size <= 0 {
			continue
		}
		outcome := strings.ToUpper(strings.TrimSpace(row.Outcome))
		if outcome == "" {
			outcome = "YES"
		}
		pos := types.OutcomePosition{
			TokenID:    row.Asset,
			MarketSlug: row.Slug,
			Outcome:    outcome,
			Shares:     row.Size,
			AvgCost:    row.AvgPrice,
			MidPrice:   row.CurPrice,
		}
		if pos.MidPrice == 0 && pos.TokenID != "" {
			pos.MidPrice, _ = c.GetMidPrice(pos.TokenID)
		}
		out = append(out, pos)
	}
	return out, nil
}

func (c *Client) funderHex() (string, error) {
	if addr := strings.TrimSpace(c.cfg.ProxyAddress); addr != "" {
		return addr, nil
	}
	signer, err := c.signerAddress()
	if err != nil {
		return "", err
	}
	return signer.Hex(), nil
}
