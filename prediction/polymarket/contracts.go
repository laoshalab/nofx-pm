package polymarket

import "github.com/ethereum/go-ethereum/common"

// Polygon mainnet Polymarket / CTF contracts.
const (
	usdcAddress                = "0x2791Bca1f2de4661ED88A30C99A7a9449Aa84174"
	ctfAddress                 = "0x4D97DCd97eC945f40cF65F87097ACe5EA0476045"
	negRiskAdapterAddress      = "0xd91E80cF2E7be2e162c6513ceD06f1dD0dA35296"
	ctfCollateralAdapter       = "0xAdA100Db00Ca00073811820692005400218FcE1f"
	negRiskCollateralAdapter   = "0xadA2005600Dec949baf300f4C6120000bDB6eAab"
	proxyFactoryAddress        = "0xaB45c5A4B0c941a2F231C04C3f49182e1A254052"
	relayHubAddress            = "0xD216153c06E857cD7f72665E0aF1d7D82172F494"
	safeFactoryAddress         = "0xaacFeEa03eb1561C4e67d661e40682Bd20E3541b"
	defaultRelayerURL          = "https://relayer-v2.polymarket.com"
	defaultPolygonRPCURL       = "https://polygon-rpc.com"
)

var (
	usdcAddr              = common.HexToAddress(usdcAddress)
	ctfAddr               = common.HexToAddress(ctfAddress)
	negRiskAdapterAddr    = common.HexToAddress(negRiskAdapterAddress)
	ctfCollateralAddr     = common.HexToAddress(ctfCollateralAdapter)
	negRiskCollateralAddr = common.HexToAddress(negRiskCollateralAdapter)
	proxyFactoryAddr      = common.HexToAddress(proxyFactoryAddress)
	relayHubAddr          = common.HexToAddress(relayHubAddress)
	safeFactoryAddr       = common.HexToAddress(safeFactoryAddress)

	safeInitCodeHash  = common.HexToHash("0x2bce2127ff07fb632d16c8347c4ebf501f4841168bed00d9e6ef715ddb6fcecf")
	proxyInitCodeHash = common.HexToHash("0xd21df8dc65880a8606f09fe0ce3df9b8869287ab0b058be05aa9e8af6330a00b")
)

const (
	relayerTxTypeSafe  = "SAFE"
	relayerTxTypeProxy = "PROXY"
)
