// Polymarket CLI tools for nofx-prediction (M0).
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"nofx/prediction/polymarket"
	"nofx/prediction/types"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}
	switch os.Args[1] {
	case "inspect":
		runInspect(os.Args[2:])
	case "midpoint":
		runMidpoint(os.Args[2:])
	case "search":
		runSearch(os.Args[2:])
	case "book":
		runBook(os.Args[2:])
	case "order":
		runOrder(os.Args[2:])
	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `nofx-prediction polymarket CLI

Usage:
  go run ./cmd/polymarket inspect  -slug <market_slug>
  go run ./cmd/polymarket midpoint   -token <clob_token_id>
  go run ./cmd/polymarket book       -token <clob_token_id> [-json]
  go run ./cmd/polymarket search     [-tag crypto] [-keyword "up or down"] [-limit 10]
  go run ./cmd/polymarket order      -token <id> -side buy|sell -price 0.55 -size 10 [-preview]
`)
}

func runInspect(args []string) {
	fs := flag.NewFlagSet("inspect", flag.ExitOnError)
	slug := fs.String("slug", "", "Polymarket market slug")
	_ = fs.Parse(args)
	if *slug == "" {
		fmt.Fprintln(os.Stderr, "missing -slug")
		os.Exit(1)
	}
	client := polymarket.NewClient(polymarket.DefaultConfig())
	m, err := client.GetMarketBySlug(*slug)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	printMarket(client, m)
}

func runMidpoint(args []string) {
	fs := flag.NewFlagSet("midpoint", flag.ExitOnError)
	token := fs.String("token", "", "CLOB token id")
	_ = fs.Parse(args)
	if *token == "" {
		fmt.Fprintln(os.Stderr, "missing -token")
		os.Exit(1)
	}
	client := polymarket.NewClient(polymarket.DefaultConfig())
	mid, err := client.GetMidPrice(*token)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("token_id: %s\nmid: %.4f\n", *token, mid)
}

func runSearch(args []string) {
	fs := flag.NewFlagSet("search", flag.ExitOnError)
	tag := fs.String("tag", "crypto", "Gamma tag filter")
	keyword := fs.String("keyword", "", "Keyword in question/slug")
	limit := fs.Int("limit", 10, "Max results")
	_ = fs.Parse(args)
	filter := types.MarketFilter{Tag: *tag, Limit: *limit}
	if k := strings.TrimSpace(*keyword); k != "" {
		filter.Keywords = []string{k}
	}
	client := polymarket.NewClient(polymarket.DefaultConfig())
	markets, err := client.SearchMarkets(filter)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("found %d markets (tag=%s)\n\n", len(markets), *tag)
	for i, m := range markets {
		fmt.Printf("%d. %s\n   slug: %s\n", i+1, m.Question, m.Slug)
	}
}

func printMarket(client *polymarket.Client, m *types.Market) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(m)
	if m.YesTokenID != "" {
		mid, err := client.GetMidPrice(m.YesTokenID)
		if err == nil {
			fmt.Printf("\nYES mid: %.4f\n", mid)
		}
	}
	if m.NoTokenID != "" {
		mid, err := client.GetMidPrice(m.NoTokenID)
		if err == nil {
			fmt.Printf("NO  mid: %.4f\n", mid)
		}
	}
}

func runBook(args []string) {
	fs := flag.NewFlagSet("book", flag.ExitOnError)
	token := fs.String("token", "", "CLOB token id")
	asJSON := fs.Bool("json", false, "JSON output")
	_ = fs.Parse(args)
	if *token == "" {
		fmt.Fprintln(os.Stderr, "missing -token")
		os.Exit(1)
	}
	client := polymarket.NewClient(polymarket.DefaultConfig())
	book, err := client.GetOrderBook(*token)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(book)
		return
	}
	fmt.Printf("token: %s mid: %.4f\n", book.TokenID, book.Mid)
	fmt.Printf("bids: %d levels | asks: %d levels\n", len(book.Bids), len(book.Asks))
	if len(book.Bids) > 0 {
		fmt.Printf("best bid: %.4f x %.2f\n", book.Bids[0].Price, book.Bids[0].Size)
	}
	if len(book.Asks) > 0 {
		fmt.Printf("best ask: %.4f x %.2f\n", book.Asks[0].Price, book.Asks[0].Size)
	}
}

func runOrder(args []string) {
	fs := flag.NewFlagSet("order", flag.ExitOnError)
	token := fs.String("token", "", "CLOB token id")
	side := fs.String("side", "buy", "buy or sell")
	price := fs.Float64("price", 0, "Limit price 0-1")
	size := fs.Float64("size", 0, "Share size")
	preview := fs.Bool("preview", true, "Preview only (default true)")
	privKey := fs.String("key", os.Getenv("POLYMARKET_PRIVATE_KEY"), "Private key hex")
	negRisk := fs.Bool("neg-risk", false, "Negative risk market")
	_ = fs.Parse(args)
	if *token == "" || *price <= 0 || *size <= 0 {
		fmt.Fprintln(os.Stderr, "usage: -token -price -size required")
		os.Exit(1)
	}
	if *privKey == "" {
		fmt.Fprintln(os.Stderr, "missing -key or POLYMARKET_PRIVATE_KEY")
		os.Exit(1)
	}
	cfg := polymarket.DefaultConfig()
	cfg.PrivateKey = *privKey
	cfg.PreviewMode = *preview
	client := polymarket.NewClient(cfg)
	sideStr := "BUY"
	if strings.EqualFold(*side, "sell") {
		sideStr = "SELL"
	}
	result, err := client.PlaceLimitOrder(types.LimitOrderReq{
		TokenID: *token,
		Side:    sideStr,
		Price:   *price,
		Size:    *size,
		NegRisk: *negRisk,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(result)
}
