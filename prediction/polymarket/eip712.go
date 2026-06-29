package polymarket

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

const clobAuthMessage = "This message attests that I control the given wallet"

type wallet struct {
	privateKey *ecdsa.PrivateKey
	address    common.Address
}

func loadWallet(privateKeyHex string) (*wallet, error) {
	privateKeyHex = strings.TrimPrefix(strings.TrimSpace(privateKeyHex), "0x")
	key, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}
	return &wallet{
		privateKey: key,
		address:    crypto.PubkeyToAddress(key.PublicKey),
	}, nil
}

func (c *Client) signerAddress() (common.Address, error) {
	if c.cfg.PrivateKey == "" {
		return common.Address{}, fmt.Errorf("polymarket: private key not configured")
	}
	w, err := loadWallet(c.cfg.PrivateKey)
	if err != nil {
		return common.Address{}, err
	}
	return w.address, nil
}

func (c *Client) makerAddress(signer common.Address) common.Address {
	if addr := strings.TrimSpace(c.cfg.ProxyAddress); addr != "" {
		return common.HexToAddress(addr)
	}
	return signer
}

func signClobAuth(w *wallet, chainID int64, timestamp string, nonce uint64) (string, error) {
	typedData := apitypes.TypedData{
		Types: apitypes.Types{
			"EIP712Domain": {
				{Name: "name", Type: "string"},
				{Name: "version", Type: "string"},
				{Name: "chainId", Type: "uint256"},
			},
			"ClobAuth": {
				{Name: "address", Type: "address"},
				{Name: "timestamp", Type: "string"},
				{Name: "nonce", Type: "uint256"},
				{Name: "message", Type: "string"},
			},
		},
		PrimaryType: "ClobAuth",
		Domain: apitypes.TypedDataDomain{
			Name:    "ClobAuthDomain",
			Version: "1",
			ChainId: math.NewHexOrDecimal256(chainID),
		},
		Message: apitypes.TypedDataMessage{
			"address":   w.address.Hex(),
			"timestamp": timestamp,
			"nonce":     fmt.Sprintf("%d", nonce),
			"message":   clobAuthMessage,
		},
	}
	return signTypedData(w.privateKey, typedData)
}

type signedOrder struct {
	Salt          *big.Int
	Maker         common.Address
	Signer        common.Address
	TokenID       *big.Int
	MakerAmount   *big.Int
	TakerAmount   *big.Int
	Side          uint8
	SignatureType uint8
	Timestamp     *big.Int
	Metadata      [32]byte
	Builder       [32]byte
	Signature     string
}

func signOrderV2(w *wallet, chainID int64, negRisk bool, order signedOrder) (string, error) {
	contract := ExchangeV2Standard
	if negRisk {
		contract = ExchangeV2NegRisk
	}
	typedData := apitypes.TypedData{
		Types: apitypes.Types{
			"EIP712Domain": {
				{Name: "name", Type: "string"},
				{Name: "version", Type: "string"},
				{Name: "chainId", Type: "uint256"},
				{Name: "verifyingContract", Type: "address"},
			},
			"Order": {
				{Name: "salt", Type: "uint256"},
				{Name: "maker", Type: "address"},
				{Name: "signer", Type: "address"},
				{Name: "tokenId", Type: "uint256"},
				{Name: "makerAmount", Type: "uint256"},
				{Name: "takerAmount", Type: "uint256"},
				{Name: "side", Type: "uint8"},
				{Name: "signatureType", Type: "uint8"},
				{Name: "timestamp", Type: "uint256"},
				{Name: "metadata", Type: "bytes32"},
				{Name: "builder", Type: "bytes32"},
			},
		},
		PrimaryType: "Order",
		Domain: apitypes.TypedDataDomain{
			Name:              "Polymarket CTF Exchange",
			Version:           "2",
			ChainId:           math.NewHexOrDecimal256(chainID),
			VerifyingContract: contract,
		},
		Message: apitypes.TypedDataMessage{
			"salt":          order.Salt,
			"maker":         order.Maker.Hex(),
			"signer":        order.Signer.Hex(),
			"tokenId":       order.TokenID,
			"makerAmount":   order.MakerAmount,
			"takerAmount":   order.TakerAmount,
			"side":          math.NewHexOrDecimal256(int64(order.Side)),
			"signatureType": math.NewHexOrDecimal256(int64(order.SignatureType)),
			"timestamp":     order.Timestamp,
			"metadata":      "0x" + common.Bytes2Hex(order.Metadata[:]),
			"builder":       "0x" + common.Bytes2Hex(order.Builder[:]),
		},
	}
	return signTypedData(w.privateKey, typedData)
}

func signTypedData(key *ecdsa.PrivateKey, typedData apitypes.TypedData) (string, error) {
	domainSeparator, err := typedData.HashStruct("EIP712Domain", typedData.Domain.Map())
	if err != nil {
		return "", err
	}
	typedDataHash, err := typedData.HashStruct(typedData.PrimaryType, typedData.Message)
	if err != nil {
		return "", err
	}
	raw := []byte{0x19, 0x01}
	raw = append(raw, domainSeparator...)
	raw = append(raw, typedDataHash...)
	hash := crypto.Keccak256Hash(raw)
	sig, err := crypto.Sign(hash.Bytes(), key)
	if err != nil {
		return "", err
	}
	if sig[64] < 27 {
		sig[64] += 27
	}
	return "0x" + common.Bytes2Hex(sig), nil
}

// usdcAmount converts USD notional to 6-decimal integer string for CLOB.
func usdcAmount(usd float64) *big.Int {
	v := new(big.Float).Mul(big.NewFloat(usd), big.NewFloat(1_000_000))
	i, _ := v.Int(nil)
	return i
}

// shareAmount converts share count to 6-decimal integer.
func shareAmount(shares float64) *big.Int {
	v := new(big.Float).Mul(big.NewFloat(shares), big.NewFloat(1_000_000))
	i, _ := v.Int(nil)
	return i
}

func buildOrderAmounts(side string, price, shares float64) (makerAmt, takerAmt *big.Int, sideNum uint8) {
	if strings.EqualFold(side, "SELL") {
		sideNum = 1
		// maker gives outcome shares, receives USDC
		return shareAmount(shares), usdcAmount(price*shares), sideNum
	}
	sideNum = 0
	// maker gives USDC, receives outcome shares
	return usdcAmount(price * shares), shareAmount(shares), sideNum
}
