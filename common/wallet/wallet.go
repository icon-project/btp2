package wallet

import (
	"github.com/icon-project/btp2/common"
	"github.com/icon-project/btp2/common/crypto"
)

type softwareWallet struct {
	skey *crypto.PrivateKey
	pkey *crypto.PublicKey
}

func (w *softwareWallet) Address() string {
	return common.NewAccountAddressFromPublicKey(w.pkey).String()
}

func (w *softwareWallet) Sign(data interface{}) ([]byte, error) {
	sig, err := crypto.NewSignature(data.([]byte), w.skey)
	if err != nil {
		return nil, err
	}
	return sig.SerializeRSV()
}

func (w *softwareWallet) PublicKey() []byte {
	return w.pkey.SerializeCompressed()
}

func (w *softwareWallet) PrivateKey() interface{} {
	return w.skey.Bytes()
}

func (w *softwareWallet) ECDH(pubKey []byte) ([]byte, error) {
	pkey, err := crypto.ParsePublicKey(pubKey)
	if err != nil {
		return nil, err
	}
	return w.skey.ECDH(pkey), nil
}

func New() *softwareWallet {
	sk, pk := crypto.GenerateKeyPair()
	return &softwareWallet{
		skey: sk,
		pkey: pk,
	}
}

func NewIcxWalletFromPrivateKey(sk *crypto.PrivateKey) (*softwareWallet, error) {
	pk := sk.PublicKey()
	return &softwareWallet{
		skey: sk,
		pkey: pk,
	}, nil
}
