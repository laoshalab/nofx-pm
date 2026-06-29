package polymarket

import (
	"encoding/json"
	"fmt"
	"math/big"
	cryptorand "crypto/rand"
	"encoding/binary"
	"nofx/prediction/types"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

type postOrderBody struct {
	OrderType string    `json:"orderType"`
	DeferExec bool      `json:"deferExec"`
	Owner     string    `json:"owner"`
	Order     orderWire `json:"order"`
}

type orderWire struct {
	Salt          string `json:"salt"`
	Maker         string `json:"maker"`
	Signer        string `json:"signer"`
	TokenID       string `json:"tokenId"`
	MakerAmount   string `json:"makerAmount"`
	TakerAmount   string `json:"takerAmount"`
	Side          string `json:"side"`
	SignatureType int    `json:"signatureType"`
	Timestamp     string `json:"timestamp"`
	Metadata      string `json:"metadata"`
	Builder       string `json:"builder"`
	Signature     string `json:"signature"`
}

// PlaceLimitOrder signs and submits a CLOB limit order (PreviewMode skips POST).
func (c *Client) PlaceLimitOrder(req types.LimitOrderReq) (*types.OrderResult, error) {
	if req.TokenID == "" {
		return nil, fmt.Errorf("empty token_id")
	}
	if req.Price <= 0 || req.Size <= 0 {
		return nil, fmt.Errorf("invalid price/size")
	}
	if c.cfg.PrivateKey == "" {
		return nil, fmt.Errorf("polymarket: private key required to place orders")
	}

	tokenID, ok := new(big.Int).SetString(req.TokenID, 10)
	if !ok {
		return nil, fmt.Errorf("invalid token_id: %s", req.TokenID)
	}

	w, err := loadWallet(c.cfg.PrivateKey)
	if err != nil {
		return nil, err
	}
	meta, err := c.GetBookMeta(req.TokenID)
	if err != nil {
		return nil, err
	}
	negRisk := req.NegRisk || meta.NegRisk
	tick, _ := strconv.ParseFloat(meta.TickSize, 64)
	price := roundToTick(req.Price, tick)

	maker := c.makerAddress(w.address)
	signer := w.address
	if c.cfg.ProxyAddress != "" {
		maker = common.HexToAddress(c.cfg.ProxyAddress)
	}
	sigType := c.cfg.SignatureTypeForOrder()
	makerAmt, takerAmt, sideNum := buildOrderAmounts(req.Side, price, req.Size)

	salt, err := randomOrderSalt()
	if err != nil {
		return nil, fmt.Errorf("order salt: %w", err)
	}
	tsMs := time.Now().UnixMilli()

	so := signedOrder{
		Salt:          new(big.Int).SetUint64(salt),
		Maker:         maker,
		Signer:        signer,
		TokenID:       tokenID,
		MakerAmount:   makerAmt,
		TakerAmount:   takerAmt,
		Side:          sideNum,
		SignatureType: uint8(sigType),
		Timestamp:     new(big.Int).SetInt64(tsMs),
	}
	orderSig, err := signOrderV2(w, c.cfg.ChainID, negRisk, so)
	if err != nil {
		return nil, fmt.Errorf("sign order: %w", err)
	}

	sideStr := "BUY"
	if sideNum == 1 {
		sideStr = "SELL"
	}
	wire := orderWire{
		Salt:          strconv.FormatUint(salt, 10),
		Maker:         maker.Hex(),
		Signer:        signer.Hex(),
		TokenID:       req.TokenID,
		MakerAmount:   makerAmt.String(),
		TakerAmount:   takerAmt.String(),
		Side:          sideStr,
		SignatureType: sigType,
		Timestamp:     strconv.FormatInt(tsMs, 10),
		Metadata:      zeroBytes32,
		Builder:       zeroBytes32,
		Signature:     orderSig,
	}

	result := &types.OrderResult{Status: "preview"}
	if c.cfg.PreviewMode {
		wireJSON, _ := json.Marshal(wire)
		result.OrderID = fmt.Sprintf("preview-%d", tsMs)
		result.PreviewWire = string(wireJSON)
		return result, nil
	}

	if err := c.ensureCredentials(); err != nil {
		return nil, err
	}
	payload := postOrderBody{
		OrderType: "GTC",
		DeferExec: false,
		Owner:     c.creds.APIKey,
		Order:     wire,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	respBody, err := c.doRequest("POST", "/order", body, true)
	if err != nil {
		return nil, err
	}
	var resp struct {
		OrderID string `json:"orderID"`
		Status  string `json:"status"`
		Error   string `json:"errorMsg"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, err
	}
	result.OrderID = resp.OrderID
	result.Status = resp.Status
	result.Error = resp.Error
	return result, nil
}

// CancelOrder cancels an open CLOB order.
func (c *Client) CancelOrder(orderID string) error {
	if orderID == "" {
		return fmt.Errorf("empty order_id")
	}
	if strings.HasPrefix(orderID, "preview-") || c.cfg.PreviewMode {
		return nil
	}
	_, err := c.doRequest("DELETE", "/order/"+orderID, nil, true)
	return err
}

// GetOrderStatus polls CLOB for order fill state.
func (c *Client) GetOrderStatus(orderID string) (*types.OrderStatus, error) {
	if orderID == "" {
		return nil, fmt.Errorf("empty order_id")
	}
	if strings.HasPrefix(orderID, "preview-") {
		return &types.OrderStatus{OrderID: orderID, Status: "preview", Terminal: true}, nil
	}
	body, err := c.doRequest("GET", "/order/"+orderID, nil, true)
	if err != nil {
		return nil, err
	}
	var raw struct {
		ID           string `json:"id"`
		SizeMatched  string `json:"size_matched"`
		OriginalSize string `json:"original_size"`
		Status       string `json:"status"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	matched, _ := strconv.ParseFloat(raw.SizeMatched, 64)
	orig, _ := strconv.ParseFloat(raw.OriginalSize, 64)
	st := strings.ToLower(raw.Status)
	terminal := st == "matched" || st == "cancelled" || st == "canceled" || st == "expired"
	return &types.OrderStatus{
		OrderID:      raw.ID,
		SizeMatched:  matched,
		OriginalSize: orig,
		Status:       raw.Status,
		Terminal:     terminal,
	}, nil
}

// ListOpenOrders returns authenticated user's open CLOB orders.
func (c *Client) ListOpenOrders() ([]types.VenueOrder, error) {
	if c.cfg.PrivateKey == "" {
		return nil, fmt.Errorf("polymarket: private key required")
	}
	if err := c.ensureCredentials(); err != nil {
		return nil, err
	}
	body, err := c.doRequest("GET", "/data/orders", nil, true)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data []struct {
			ID           string `json:"id"`
			Status       string `json:"status"`
			AssetID      string `json:"asset_id"`
			Market       string `json:"market"`
			Side         string `json:"side"`
			Price        string `json:"price"`
			OriginalSize string `json:"original_size"`
			SizeMatched  string `json:"size_matched"`
			CreatedAt    string `json:"created_at"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	out := make([]types.VenueOrder, 0, len(resp.Data))
	for _, row := range resp.Data {
		price, _ := strconv.ParseFloat(row.Price, 64)
		orig, _ := strconv.ParseFloat(row.OriginalSize, 64)
		matched, _ := strconv.ParseFloat(row.SizeMatched, 64)
		out = append(out, types.VenueOrder{
			OrderID:      row.ID,
			TokenID:      row.AssetID,
			MarketID:     row.Market,
			Side:         row.Side,
			Price:        price,
			OriginalSize: orig,
			SizeMatched:  matched,
			Status:       row.Status,
			CreatedAt:    row.CreatedAt,
		})
	}
	return out, nil
}

const zeroBytes32 = "0x0000000000000000000000000000000000000000000000000000000000000000"

func randomOrderSalt() (uint64, error) {
	var b [8]byte
	if _, err := cryptorand.Read(b[:]); err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint64(b[:]), nil
}

var _ types.PredictionVenue = (*Client)(nil)
