// Prediction CLI — run one AI cycle or inspect runtime (M2).
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"nofx/mcp"
	predcfg "nofx/prediction/config"
	"nofx/prediction/engine"
	"nofx/prediction/polymarket"
	"nofx/prediction/sim"
	"nofx/prediction/trader"
	"nofx/prediction/types"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}
	switch os.Args[1] {
	case "cycle":
		runCycle(os.Args[2:])
	case "markets":
		runMarkets(os.Args[2:])
	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `nofx-prediction CLI (M2)

Usage:
  go run ./cmd/prediction cycle   -provider deepseek -key $KEY [-slug btc-market] [-preview|-sim|-live]
  go run ./cmd/prediction markets [-tag crypto] [-limit 10]

Simulation (no private key):
  go run ./cmd/prediction cycle -provider deepseek -key $KEY -sim -balance 10000 -tag crypto
`)
}

func runMarkets(args []string) {
	fs := flag.NewFlagSet("markets", flag.ExitOnError)
	tag := fs.String("tag", "crypto", "Gamma tag")
	limit := fs.Int("limit", 10, "Max markets")
	_ = fs.Parse(args)

	client := polymarket.NewClient(polymarket.DefaultConfig())
	cfg := predcfg.DefaultStrategyConfig()
	cfg.MarketSource.Tag = *tag
	cfg.MarketSource.Limit = *limit
	eng := engine.NewPredictionEngine(cfg, client)
	snaps, err := eng.GetCandidateMarkets()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(snaps)
}

func runCycle(args []string) {
	fs := flag.NewFlagSet("cycle", flag.ExitOnError)
	provider := fs.String("provider", "deepseek", "AI provider")
	apiKey := fs.String("key", os.Getenv("DEEPSEEK_API_KEY"), "AI API key")
	slug := fs.String("slug", "", "Static market slug (optional)")
	tag := fs.String("tag", "crypto", "Tag search when no slug")
	preview := fs.Bool("preview", false, "Preview mode (sign only, no live orders)")
	simMode := fs.Bool("sim", false, "Simulation mode (virtual USDC, live market data)")
	live := fs.Bool("live", false, "Live Polymarket orders (requires private key)")
	balance := fs.Float64("balance", 10000, "Initial virtual USDC for -sim")
	_ = fs.Parse(args)

	if *apiKey == "" {
		fmt.Fprintln(os.Stderr, "missing -key or DEEPSEEK_API_KEY")
		os.Exit(1)
	}

	modeCount := 0
	if *preview {
		modeCount++
	}
	if *simMode {
		modeCount++
	}
	if *live {
		modeCount++
	}
	if modeCount == 0 {
		*simMode = true
	} else if modeCount > 1 {
		fmt.Fprintln(os.Stderr, "use only one of -preview, -sim, or -live")
		os.Exit(1)
	}

	var venue types.PredictionVenue
	strat := predcfg.DefaultStrategyConfig()
	if s := strings.TrimSpace(*slug); s != "" {
		strat.MarketSource.Type = predcfg.MarketSourceStatic
		strat.StaticSlugs = []string{s}
	} else {
		strat.MarketSource.Tag = *tag
	}

	switch {
	case *simMode:
		simCfg := sim.DefaultConfig()
		simCfg.InitialBalanceUsd = *balance
		pub := polymarket.NewClient(polymarket.DefaultConfig())
		venue = sim.NewVenue(pub, simCfg, sim.NewLedger(simCfg.InitialBalanceUsd))
		strat.Risk.PreviewMode = false
	case *live:
		pmCfg := polymarket.DefaultConfig()
		pmCfg.PreviewMode = false
		pmCfg.PrivateKey = os.Getenv("POLYMARKET_PRIVATE_KEY")
		venue = polymarket.NewClient(pmCfg)
		strat.Risk.PreviewMode = false
	default:
		pmCfg := polymarket.DefaultConfig()
		pmCfg.PreviewMode = true
		pmCfg.PrivateKey = os.Getenv("POLYMARKET_PRIVATE_KEY")
		venue = polymarket.NewClient(pmCfg)
		strat.Risk.PreviewMode = true
	}

	ptCfg := predcfg.TraderConfig{
		ID:       "cli",
		Name:     "CLI Prediction",
		AIModel:  *provider,
		APIKey:   *apiKey,
		Strategy: strat,
	}
	ai := mcp.NewAIClientByProvider(*provider)
	pt, err := trader.NewPredictionTrader(ptCfg, venue, ai)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	decision, err := pt.RunOnce()
	if err != nil {
		fmt.Fprintf(os.Stderr, "cycle error: %v\n", err)
		if decision != nil {
			fmt.Fprintf(os.Stderr, "raw response:\n%s\n", decision.RawResponse)
		}
		os.Exit(1)
	}

	avail, equity, _ := venue.GetCollateral()
	positions, _ := venue.GetOutcomePositions()

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	out := map[string]interface{}{
		"cot":            decision.CoTTrace,
		"decisions":      decision.Decisions,
		"duration_ms":    decision.AIRequestDurationMs,
		"available_usdc": avail,
		"total_equity":   equity,
		"positions":      positions,
	}
	_ = enc.Encode(out)
}
