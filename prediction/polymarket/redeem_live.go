package polymarket

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	predtypes "nofx/prediction/types"
)

func (c *Client) liveRedeemReady() bool {
	if c.cfg.PreviewMode || strings.TrimSpace(c.cfg.PrivateKey) == "" {
		return false
	}
	if c.needsRelayer() {
		return newRelayerClient(c.cfg).hasCreds()
	}
	return true
}

// LiveRedeemConfigured reports whether the server can attempt live redeem
// (Builder relayer for proxy/Safe, or Polygon RPC for direct EOA).
func LiveRedeemConfigured(cfg Config) bool {
	cfg = MergeEnvConfig(cfg)
	if newRelayerClient(cfg).hasCreds() {
		return true
	}
	return strings.TrimSpace(cfg.PolygonRPCURL) != "" || defaultPolygonRPCURL != ""
}

func (c *Client) needsRelayer() bool {
	if strings.TrimSpace(c.cfg.ProxyAddress) != "" {
		return true
	}
	return c.cfg.SignatureType == 1 || c.cfg.SignatureType == 2
}

func (c *Client) redeemConditionLive(conditionID string, negRisk bool) (*predtypes.RedeemResult, error) {
	w, err := loadWallet(c.cfg.PrivateKey)
	if err != nil {
		return nil, err
	}
	calls, err := c.buildRedeemCalls(conditionID, negRisk)
	if err != nil {
		return nil, err
	}
	if c.needsRelayer() {
		txHash, err := c.redeemViaRelayer(w, calls, conditionID)
		if err != nil {
			return nil, err
		}
		return &predtypes.RedeemResult{
			ConditionID: conditionID,
			Status:      "success",
			TxHash:      txHash,
		}, nil
	}
	txHash, err := c.redeemViaEOA(w, calls)
	if err != nil {
		return nil, err
	}
	return &predtypes.RedeemResult{
		ConditionID: conditionID,
		Status:      "success",
		TxHash:      txHash,
	}, nil
}

func (c *Client) buildRedeemCalls(conditionID string, negRisk bool) ([]chainCall, error) {
	redeem, err := buildRedeemCall(conditionID, negRisk, c.cfg.RedeemUSDCe)
	if err != nil {
		return nil, err
	}
	if c.cfg.RedeemUSDCe {
		return []chainCall{redeem}, nil
	}
	funder, err := c.funderAddr()
	if err != nil {
		return nil, err
	}
	adapter := redeem.To
	ok, err := c.isAdapterApproved(funder, adapter)
	if err != nil {
		return nil, err
	}
	if ok {
		return []chainCall{redeem}, nil
	}
	approve, err := buildApproveCall(adapter)
	if err != nil {
		return nil, err
	}
	return []chainCall{approve, redeem}, nil
}

func (c *Client) funderAddr() (common.Address, error) {
	if addr := strings.TrimSpace(c.cfg.ProxyAddress); addr != "" {
		return common.HexToAddress(addr), nil
	}
	return c.signerAddress()
}

func (c *Client) polygonRPC() string {
	if strings.TrimSpace(c.cfg.PolygonRPCURL) != "" {
		return c.cfg.PolygonRPCURL
	}
	return defaultPolygonRPCURL
}

func (c *Client) isAdapterApproved(account, adapter common.Address) (bool, error) {
	client, err := ethclient.Dial(c.polygonRPC())
	if err != nil {
		return false, err
	}
	defer client.Close()
	data, err := encodeIsApprovedForAll(account, adapter)
	if err != nil {
		return false, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	out, err := client.CallContract(ctx, ethereum.CallMsg{
		To:   &ctfAddr,
		Data: data,
	}, nil)
	if err != nil {
		return false, err
	}
	vals, err := isApprovedABI.Unpack("isApprovedForAll", out)
	if err != nil || len(vals) == 0 {
		return false, err
	}
	approved, _ := vals[0].(bool)
	return approved, nil
}

func (c *Client) redeemViaEOA(w *wallet, calls []chainCall) (string, error) {
	client, err := ethclient.Dial(c.polygonRPC())
	if err != nil {
		return "", err
	}
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	var lastHash string
	for _, call := range calls {
		hash, err := sendEOATx(ctx, client, w.privateKey, c.cfg.ChainID, call.To, call.Data)
		if err != nil {
			return lastHash, err
		}
		lastHash = hash
		if _, err := waitMined(ctx, client, common.HexToHash(hash)); err != nil {
			return hash, err
		}
	}
	return lastHash, nil
}

func sendEOATx(ctx context.Context, client *ethclient.Client, key *ecdsa.PrivateKey, chainID int64, to common.Address, data []byte) (string, error) {
	from := crypto.PubkeyToAddress(key.PublicKey)
	nonce, err := client.PendingNonceAt(ctx, from)
	if err != nil {
		return "", err
	}
	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return "", err
	}
	gasLimit, err := client.EstimateGas(ctx, ethereum.CallMsg{
		From: from,
		To:   &to,
		Data: data,
	})
	if err != nil {
		gasLimit = 500_000
	}
	tx := types.NewTransaction(nonce, to, big.NewInt(0), gasLimit, gasPrice, data)
	signer := types.NewLondonSigner(big.NewInt(chainID))
	signed, err := types.SignTx(tx, signer, key)
	if err != nil {
		return "", err
	}
	if err := client.SendTransaction(ctx, signed); err != nil {
		return "", err
	}
	return signed.Hash().Hex(), nil
}

func waitMined(ctx context.Context, client *ethclient.Client, hash common.Hash) (*types.Receipt, error) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		receipt, err := client.TransactionReceipt(ctx, hash)
		if err == nil {
			if receipt.Status == types.ReceiptStatusFailed {
				return receipt, fmt.Errorf("transaction reverted")
			}
			return receipt, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}

func (c *Client) redeemViaRelayer(w *wallet, calls []chainCall, conditionID string) (string, error) {
	rl := newRelayerClient(c.cfg)
	if !rl.hasCreds() {
		return "", fmt.Errorf("polymarket: POLYMARKET_BUILDER_* credentials required for proxy/safe redeem")
	}
	owner := w.address
	metadata := "nofx redeem " + conditionID

	if c.cfg.SignatureType == 1 {
		proxy := deriveProxyAddress(owner)
		if addr := strings.TrimSpace(c.cfg.ProxyAddress); addr != "" {
			proxy = common.HexToAddress(addr)
		}
		payload, err := rl.getRelayPayload(owner.Hex(), relayerTxTypeProxy)
		if err != nil {
			return "", err
		}
		req, err := buildProxySubmitRequest(w.privateKey, owner, proxy, calls, common.HexToAddress(payload.Address), payload.Nonce, metadata)
		if err != nil {
			return "", err
		}
		submitted, err := rl.submit(req)
		if err != nil {
			return "", err
		}
		return rl.waitConfirmed(submitted.TransactionID, 60)
	}

	safe := deriveSafeAddress(owner)
	if addr := strings.TrimSpace(c.cfg.ProxyAddress); addr != "" {
		safe = common.HexToAddress(addr)
	}
	var lastHash string
	for _, call := range calls {
		nonce, err := rl.getNonce(owner.Hex(), relayerTxTypeSafe)
		if err != nil {
			return lastHash, err
		}
		req, err := buildSafeSubmitRequest(w.privateKey, owner, safe, call, c.cfg.ChainID, nonce, metadata)
		if err != nil {
			return lastHash, err
		}
		submitted, err := rl.submit(req)
		if err != nil {
			return lastHash, err
		}
		hash, err := rl.waitConfirmed(submitted.TransactionID, 60)
		if err != nil {
			return hash, err
		}
		lastHash = hash
	}
	return lastHash, nil
}
