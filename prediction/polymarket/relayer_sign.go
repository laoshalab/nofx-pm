package polymarket

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

type relayerSignatureParams struct {
	GasPrice       string `json:"gasPrice"`
	Operation      string `json:"operation,omitempty"`
	SafeTxnGas     string `json:"safeTxnGas,omitempty"`
	BaseGas        string `json:"baseGas,omitempty"`
	GasToken       string `json:"gasToken,omitempty"`
	RefundReceiver string `json:"refundReceiver,omitempty"`
	GasLimit       string `json:"gasLimit,omitempty"`
	RelayerFee     string `json:"relayerFee,omitempty"`
	RelayHub       string `json:"relayHub,omitempty"`
	Relay          string `json:"relay,omitempty"`
}

type relayerSubmitRequest struct {
	From            string                 `json:"from"`
	To              string                 `json:"to"`
	ProxyWallet     string                 `json:"proxyWallet"`
	Data            string                 `json:"data"`
	Nonce           string                 `json:"nonce"`
	Signature       string                 `json:"signature"`
	SignatureParams relayerSignatureParams `json:"signatureParams"`
	Type            string                 `json:"type"`
	Metadata        string                 `json:"metadata"`
}

func eip712Digest(typedData apitypes.TypedData) (common.Hash, error) {
	domainSeparator, err := typedData.HashStruct("EIP712Domain", typedData.Domain.Map())
	if err != nil {
		return common.Hash{}, err
	}
	typedDataHash, err := typedData.HashStruct(typedData.PrimaryType, typedData.Message)
	if err != nil {
		return common.Hash{}, err
	}
	raw := []byte{0x19, 0x01}
	raw = append(raw, domainSeparator...)
	raw = append(raw, typedDataHash...)
	return crypto.Keccak256Hash(raw), nil
}

func signPersonalDigest(key *ecdsa.PrivateKey, digest common.Hash) (string, error) {
	prefixed := accounts.TextHash(digest.Bytes())
	sig, err := crypto.Sign(prefixed, key)
	if err != nil {
		return "", err
	}
	if sig[64] < 27 {
		sig[64] += 27
	}
	return "0x" + common.Bytes2Hex(sig), nil
}

func splitAndPackSig(sigHex string) (string, error) {
	sigHex = strings.TrimPrefix(strings.TrimSpace(sigHex), "0x")
	if len(sigHex) != 130 {
		return "", fmt.Errorf("invalid signature length")
	}
	sigV, err := strconv.ParseInt(sigHex[128:130], 16, 64)
	if err != nil {
		return "", err
	}
	switch sigV {
	case 0, 1:
		sigV += 31
	case 27, 28:
		sigV += 4
	default:
		return "", fmt.Errorf("invalid signature v=%d", sigV)
	}
	sigHex = sigHex[:128] + fmt.Sprintf("%02x", sigV)

	rBytes, err := hex.DecodeString(sigHex[0:64])
	if err != nil {
		return "", err
	}
	sBytes, err := hex.DecodeString(sigHex[64:128])
	if err != nil {
		return "", err
	}
	vByte := byte(sigV)

	packed := make([]byte, 65)
	copy(packed[0:32], rBytes)
	copy(packed[32:64], sBytes)
	packed[64] = vByte
	return "0x" + common.Bytes2Hex(packed), nil
}

func buildSafeSubmitRequest(
	key *ecdsa.PrivateKey,
	owner common.Address,
	safe common.Address,
	call chainCall,
	chainID int64,
	nonce string,
	metadata string,
) (relayerSubmitRequest, error) {
	zero := common.Address{}.Hex()
	typedData := apitypes.TypedData{
		Types: apitypes.Types{
			"EIP712Domain": {
				{Name: "chainId", Type: "uint256"},
				{Name: "verifyingContract", Type: "address"},
			},
			"SafeTx": {
				{Name: "to", Type: "address"},
				{Name: "value", Type: "uint256"},
				{Name: "data", Type: "bytes"},
				{Name: "operation", Type: "uint8"},
				{Name: "safeTxGas", Type: "uint256"},
				{Name: "baseGas", Type: "uint256"},
				{Name: "gasPrice", Type: "uint256"},
				{Name: "gasToken", Type: "address"},
				{Name: "refundReceiver", Type: "address"},
				{Name: "nonce", Type: "uint256"},
			},
		},
		PrimaryType: "SafeTx",
		Domain: apitypes.TypedDataDomain{
			ChainId:           math.NewHexOrDecimal256(chainID),
			VerifyingContract: safe.Hex(),
		},
		Message: apitypes.TypedDataMessage{
			"to":             call.To.Hex(),
			"value":          "0",
			"data":           "0x" + common.Bytes2Hex(call.Data),
			"operation":      "0",
			"safeTxGas":      "0",
			"baseGas":        "0",
			"gasPrice":       "0",
			"gasToken":       zero,
			"refundReceiver": zero,
			"nonce":          nonce,
		},
	}
	digest, err := eip712Digest(typedData)
	if err != nil {
		return relayerSubmitRequest{}, err
	}
	sig, err := signPersonalDigest(key, digest)
	if err != nil {
		return relayerSubmitRequest{}, err
	}
	packed, err := splitAndPackSig(sig)
	if err != nil {
		return relayerSubmitRequest{}, err
	}
	return relayerSubmitRequest{
		From:        owner.Hex(),
		To:          call.To.Hex(),
		ProxyWallet: safe.Hex(),
		Data:        "0x" + common.Bytes2Hex(call.Data),
		Nonce:       nonce,
		Signature:   packed,
		SignatureParams: relayerSignatureParams{
			GasPrice:       "0",
			Operation:      "0",
			SafeTxnGas:     "0",
			BaseGas:        "0",
			GasToken:       zero,
			RefundReceiver: zero,
		},
		Type:     relayerTxTypeSafe,
		Metadata: metadata,
	}, nil
}

func buildProxySubmitRequest(
	key *ecdsa.PrivateKey,
	owner common.Address,
	proxy common.Address,
	calls []chainCall,
	relay common.Address,
	nonce string,
	metadata string,
) (relayerSubmitRequest, error) {
	data, err := encodeProxyCalls(calls)
	if err != nil {
		return relayerSubmitRequest{}, err
	}
	gasLimit := big.NewInt(10_000_000)
	txFee := big.NewInt(0)
	gasPrice := big.NewInt(0)

	prefix := []byte("rlx:")
	parts := [][]byte{
		prefix,
		owner.Bytes(),
		proxyFactoryAddr.Bytes(),
		data,
		encodeU256(txFee.Uint64()),
		encodeU256(gasPrice.Uint64()),
		encodeU256(gasLimit.Uint64()),
		encodeU256(parseUint64(nonce)),
		relayHubAddr.Bytes(),
		relay.Bytes(),
	}
	var concat []byte
	for _, p := range parts {
		concat = append(concat, p...)
	}
	digest := crypto.Keccak256Hash(concat)
	sig, err := signPersonalDigest(key, digest)
	if err != nil {
		return relayerSubmitRequest{}, err
	}
	return relayerSubmitRequest{
		From:        owner.Hex(),
		To:          proxyFactoryAddr.Hex(),
		ProxyWallet: proxy.Hex(),
		Data:        "0x" + common.Bytes2Hex(data),
		Nonce:       nonce,
		Signature:   sig,
		SignatureParams: relayerSignatureParams{
			GasPrice:   "0",
			GasLimit:   gasLimit.String(),
			RelayerFee: "0",
			RelayHub:   relayHubAddr.Hex(),
			Relay:      relay.Hex(),
		},
		Type:     relayerTxTypeProxy,
		Metadata: metadata,
	}, nil
}

func parseUint64(s string) uint64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	if strings.HasPrefix(s, "0x") {
		v, _ := strconv.ParseUint(strings.TrimPrefix(s, "0x"), 16, 64)
		return v
	}
	v, _ := strconv.ParseUint(s, 10, 64)
	return v
}
