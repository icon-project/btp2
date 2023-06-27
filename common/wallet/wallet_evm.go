package wallet

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type EvmWallet struct {
	Skey *ecdsa.PrivateKey
	Pkey *ecdsa.PublicKey
}

func (w *EvmWallet) Address() string {
	pubBytes := w.PublicKey()
	return common.BytesToAddress(crypto.Keccak256(pubBytes[1:])[12:]).Hex()
}

func (w *EvmWallet) Sign(data []byte) ([]byte, error) {
	return crypto.Sign(data, w.Skey)
}

func (w *EvmWallet) PublicKey() []byte {
	return crypto.FromECDSAPub(w.Pkey)
}

func (w *EvmWallet) ECDH(pubKey []byte) ([]byte, error) {
	//TODO: Not implemented yet
	return nil, nil
}

func NewEvmWalletFromPrivateKey(sk *ecdsa.PrivateKey) (*EvmWallet, error) {
	return &EvmWallet{
		Skey: sk,
		Pkey: &sk.PublicKey,
	}, nil
}
