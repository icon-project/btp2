package wallet

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/ecies"
)

type EvmWallet struct {
	Skey *ecdsa.PrivateKey
	Pkey *ecdsa.PublicKey
}

func (w *EvmWallet) Address() string {
	pubBytes := w.PublicKey()
	return common.BytesToAddress(crypto.Keccak256(pubBytes[1:])[12:]).Hex()
}

func (w *EvmWallet) Sign(data interface{}) ([]byte, error) {
	return crypto.Sign(data.([]byte), w.Skey)
}

func (w *EvmWallet) PublicKey() []byte {
	return crypto.FromECDSAPub(w.Pkey)
}

func (w *EvmWallet) PrivateKey() interface{} {
	return crypto.FromECDSA(w.Skey)
}

func (w *EvmWallet) ECDH(pubKey []byte) ([]byte, error) {
	pri := ecies.ImportECDSA(w.Skey)
	pub := ecies.ImportECDSAPublic(w.Pkey)
	return pri.GenerateShared(pub, pub.Params.KeyLen, pub.Params.KeyLen)
}

func NewEvmWalletFromPrivateKey(sk *ecdsa.PrivateKey) (*EvmWallet, error) {
	return &EvmWallet{
		Skey: sk,
		Pkey: &sk.PublicKey,
	}, nil
}
