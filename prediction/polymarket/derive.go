package polymarket

import (
	"encoding/binary"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func deriveCreate2Address(factory common.Address, salt, bytecodeHash common.Hash) common.Address {
	data := append([]byte{0xff}, factory.Bytes()...)
	data = append(data, salt.Bytes()...)
	data = append(data, bytecodeHash.Bytes()...)
	hash := crypto.Keccak256(data)
	return common.BytesToAddress(hash[12:])
}

// deriveSafeAddress computes the Polymarket Gnosis Safe for an EOA owner.
func deriveSafeAddress(owner common.Address) common.Address {
	salt := crypto.Keccak256(encodeABIAddress(owner))
	return deriveCreate2Address(safeFactoryAddr, common.BytesToHash(salt), safeInitCodeHash)
}

// deriveProxyAddress computes the Polymarket proxy wallet for an EOA owner.
func deriveProxyAddress(owner common.Address) common.Address {
	salt := crypto.Keccak256(owner.Bytes())
	return deriveCreate2Address(proxyFactoryAddr, common.BytesToHash(salt), proxyInitCodeHash)
}

func encodeABIAddress(addr common.Address) []byte {
	out := make([]byte, 32)
	copy(out[12:], addr.Bytes())
	return out
}

func encodeU256(v uint64) []byte {
	out := make([]byte, 32)
	binary.BigEndian.PutUint64(out[24:], v)
	return out
}
