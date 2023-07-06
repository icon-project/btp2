package ethbr

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/icon-project/btp2/chain"
	"github.com/icon-project/btp2/common/log"
	"github.com/icon-project/btp2/common/types"
	"github.com/icon-project/btp2/common/wallet"
)

func NewReceiver(srcCfg, dstCfg chain.BaseConfig, l log.Logger) *ethbr {
	receiver := newEthBridge(srcCfg.Address, dstCfg.Address, srcCfg.Endpoint, l, srcCfg.Options)
	return receiver
}

func NewSender(srcAddr types.BtpAddress, cfg chain.BaseConfig, l log.Logger) (types.Sender, error) {
	//TODO refactoring
	w, err := Wallet(cfg.KeyStorePass, cfg.KeySecret, cfg.KeyStoreData)
	if err != nil {
		return nil, err
	}

	return newSender(srcAddr, cfg.Address, w, cfg.Endpoint, cfg.Options, l), nil
}

func Wallet(passwd, secret string, keyStore json.RawMessage) (types.Wallet, error) {
	pw, err := resolvePassword(secret, passwd)
	if err != nil {
		return nil, err
	}
	return wallet.DecryptKeyStore(keyStore, pw)
}

func resolvePassword(keySecret, keyStorePass string) ([]byte, error) {
	if keySecret != "" {
		return os.ReadFile(keySecret)
	} else {
		if keyStorePass != "" {
			return []byte(keyStorePass), nil
		} else {
			//TODO
			return nil, fmt.Errorf("")
		}
	}
}
