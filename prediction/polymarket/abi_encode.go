package polymarket

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

var (
	redeem4ArgABI = mustParseABI(`[{"name":"redeemPositions","type":"function","inputs":[{"name":"collateralToken","type":"address"},{"name":"parentCollectionId","type":"bytes32"},{"name":"conditionId","type":"bytes32"},{"name":"indexSets","type":"uint256[]"}],"outputs":[]}]`)
	approveABI    = mustParseABI(`[{"name":"setApprovalForAll","type":"function","inputs":[{"name":"operator","type":"address"},{"name":"approved","type":"bool"}],"outputs":[]}]`)
	isApprovedABI = mustParseABI(`[{"name":"isApprovedForAll","type":"function","inputs":[{"name":"account","type":"address"},{"name":"operator","type":"address"}],"outputs":[{"name":"","type":"bool"}],"stateMutability":"view","type":"function"}]`)
	proxyABI      = mustParseABI(`[{"name":"proxy","type":"function","inputs":[{"components":[{"name":"typeCode","type":"uint8"},{"name":"to","type":"address"},{"name":"value","type":"uint256"},{"name":"data","type":"bytes"}],"name":"calls","type":"tuple[]"}],"outputs":[{"name":"returnValues","type":"bytes[]"}],"stateMutability":"payable","type":"function"}]`)
)

func mustParseABI(raw string) abi.ABI {
	parsed, err := abi.JSON(strings.NewReader(raw))
	if err != nil {
		panic(err)
	}
	return parsed
}

func parseConditionID(raw string) (common.Hash, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "0x")
	if len(raw) != 64 {
		return common.Hash{}, fmt.Errorf("invalid condition_id length")
	}
	b, err := hex.DecodeString(raw)
	if err != nil {
		return common.Hash{}, fmt.Errorf("invalid condition_id: %w", err)
	}
	var out common.Hash
	copy(out[:], b)
	return out, nil
}

type chainCall struct {
	To   common.Address
	Data []byte
}

func encodeRedeem4Arg(conditionID common.Hash) ([]byte, error) {
	parent := common.Hash{}
	indexSets := []*big.Int{big.NewInt(1), big.NewInt(2)}
	return redeem4ArgABI.Pack("redeemPositions", usdcAddr, parent, conditionID, indexSets)
}

func encodeSetApprovalForAll(operator common.Address, approved bool) ([]byte, error) {
	return approveABI.Pack("setApprovalForAll", operator, approved)
}

func encodeIsApprovedForAll(account, operator common.Address) ([]byte, error) {
	return isApprovedABI.Pack("isApprovedForAll", account, operator)
}

func redeemTarget(negRisk, usdce bool) common.Address {
	if usdce {
		if negRisk {
			return negRiskAdapterAddr
		}
		return ctfAddr
	}
	if negRisk {
		return negRiskCollateralAddr
	}
	return ctfCollateralAddr
}

func buildRedeemCall(conditionID string, negRisk, usdce bool) (chainCall, error) {
	cid, err := parseConditionID(conditionID)
	if err != nil {
		return chainCall{}, err
	}
	data, err := encodeRedeem4Arg(cid)
	if err != nil {
		return chainCall{}, err
	}
	return chainCall{To: redeemTarget(negRisk, usdce), Data: data}, nil
}

func buildApproveCall(adapter common.Address) (chainCall, error) {
	data, err := encodeSetApprovalForAll(adapter, true)
	if err != nil {
		return chainCall{}, err
	}
	return chainCall{To: ctfAddr, Data: data}, nil
}

func encodeProxyCalls(calls []chainCall) ([]byte, error) {
	type proxyCall struct {
		TypeCode uint8
		To       common.Address
		Value    *big.Int
		Data     []byte
	}
	wrapped := make([]proxyCall, len(calls))
	for i, c := range calls {
		wrapped[i] = proxyCall{
			TypeCode: 1,
			To:       c.To,
			Value:    big.NewInt(0),
			Data:     c.Data,
		}
	}
	return proxyABI.Pack("proxy", wrapped)
}
