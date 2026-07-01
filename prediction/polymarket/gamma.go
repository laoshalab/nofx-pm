package polymarket

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"nofx/prediction/types"
	"strconv"
	"strings"
	"time"
)

// Client implements Polymarket Gamma + CLOB access.
type Client struct {
	cfg   Config
	http  *http.Client
	creds *apiCredentials
}

func NewClient(cfg Config) *Client {
	if cfg.GammaURL == "" {
		cfg = DefaultConfig()
	}
	c := &Client{cfg: cfg}
	c.ensureHTTP()
	return c
}

func (c *Client) VenueID() string { return "polymarket" }

// GetMarketBySlug fetches a single market from Gamma API.
func (c *Client) GetMarketBySlug(slug string) (*types.Market, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return nil, fmt.Errorf("empty slug")
	}
	u := fmt.Sprintf("%s/markets?slug=%s", c.cfg.GammaURL, url.QueryEscape(slug))
	body, err := c.getJSON(u)
	if err != nil {
		return nil, err
	}
	var rows []gammaMarket
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("market not found: %s", slug)
	}
	return rows[0].toMarket(), nil
}

// SearchMarkets lists markets by tag (e.g. crypto).
func (c *Client) SearchMarkets(filter types.MarketFilter) ([]types.Market, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	q := url.Values{}
	q.Set("limit", strconv.Itoa(limit))
	if filter.Closed != nil {
		q.Set("closed", strconv.FormatBool(*filter.Closed))
	} else {
		q.Set("closed", "false")
	}
	if filter.Tag != "" {
		q.Set("tag", filter.Tag)
	}
	u := fmt.Sprintf("%s/markets?%s", c.cfg.GammaURL, q.Encode())
	body, err := c.getJSON(u)
	if err != nil {
		return nil, err
	}
	var rows []gammaMarket
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, err
	}
	out := make([]types.Market, 0, len(rows))
	for _, row := range rows {
		m := row.toMarket()
		if !matchKeywords(m, filter.Keywords) {
			continue
		}
		out = append(out, *m)
	}
	return out, nil
}

func matchKeywords(m *types.Market, keywords []string) bool {
	if len(keywords) == 0 {
		return true
	}
	hay := strings.ToLower(m.Question + " " + m.Slug)
	for _, k := range keywords {
		if strings.Contains(hay, strings.ToLower(k)) {
			return true
		}
	}
	return false
}


type gammaMarket struct {
	ID            string  `json:"id"`
	Slug          string  `json:"slug"`
	Question      string  `json:"question"`
	ConditionID   string  `json:"conditionId"`
	EndDate       string  `json:"endDate"`
	Closed        bool    `json:"closed"`
	NegRisk       bool    `json:"negRisk"`
	ClobTokenIds  string  `json:"clobTokenIds"`
	OutcomePrices string  `json:"outcomePrices"`
	Volume24hr    float64 `json:"volume24hr"`
	Liquidity     float64 `json:"liquidityNum"`
}

func (g gammaMarket) toMarket() *types.Market {
	yes, no := parseTokenIDs(g.ClobTokenIds)
	var end time.Time
	if g.EndDate != "" {
		end, _ = time.Parse(time.RFC3339, g.EndDate)
	}
	return &types.Market{
		Venue:         "polymarket",
		ID:            g.ID,
		Slug:          g.Slug,
		Question:      g.Question,
		ConditionID:   g.ConditionID,
		EndDate:       end,
		Closed:        g.Closed,
		NegRisk:       g.NegRisk,
		YesTokenID:    yes,
		NoTokenID:     no,
		Volume24h:     g.Volume24hr,
		Liquidity:     g.Liquidity,
		OutcomePrices: parseOutcomePrices(g.OutcomePrices),
	}
}

func parseOutcomePrices(raw string) []float64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var parts []string
	if strings.HasPrefix(raw, "[") {
		if err := json.Unmarshal([]byte(raw), &parts); err != nil {
			return nil
		}
	} else {
		parts = strings.Split(raw, ",")
	}
	out := make([]float64, 0, len(parts))
	for _, p := range parts {
		p = strings.Trim(p, `" `)
		if p == "" {
			continue
		}
		v, err := strconv.ParseFloat(p, 64)
		if err != nil {
			continue
		}
		out = append(out, v)
	}
	return out
}

func parseTokenIDs(raw string) (yes, no string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", ""
	}
	var ids []string
	if strings.HasPrefix(raw, "[") {
		_ = json.Unmarshal([]byte(raw), &ids)
	} else {
		ids = strings.Split(raw, ",")
	}
	if len(ids) > 0 {
		yes = strings.Trim(ids[0], `" `)
	}
	if len(ids) > 1 {
		no = strings.Trim(ids[1], `" `)
	}
	return yes, no
}
