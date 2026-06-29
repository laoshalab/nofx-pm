package sim

const (
	FillModelLimitCross = "limit_cross"
	FillModelMidInstant = "mid_instant"
)

// Config controls simulated trading behavior.
type Config struct {
	InitialBalanceUsd float64 `json:"initial_balance_usd"`
	FillModel         string  `json:"fill_model"`
	SlippageBps       int     `json:"slippage_bps"`
}

func DefaultConfig() Config {
	return Config{
		InitialBalanceUsd: 10_000,
		FillModel:         FillModelLimitCross,
	}
}
