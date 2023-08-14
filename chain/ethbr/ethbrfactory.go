package ethbr

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/icon-project/btp2/chain"
	"github.com/icon-project/btp2/common/link"
	"github.com/icon-project/btp2/common/log"
	"github.com/icon-project/btp2/common/types"
	"github.com/icon-project/btp2/common/wallet"
)

const TYPE = "eth-bridge"

func RegisterEthBridge() {
	link.RegisterFactory(&link.Factory{
		Type:             TYPE,
		ParseChainConfig: ParseChainConfig,
		NewReceiver:      NewReceiver,
		NewSender:        NewSender,
	})
}

func ParseChainConfig(raw json.RawMessage) (link.ChainConfig, error) {
	cfg := chain.BaseConfig{}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, err
	}
	if cfg.Type != TYPE {
		return nil, fmt.Errorf("invalid type (type:%s)", cfg.Type)
	}
	return cfg, nil
}

func NewReceiver(srcCfg link.ChainConfig, dstAddr types.BtpAddress, baseDir string, l log.Logger) (link.Receiver, error) {
	src := srcCfg.(chain.BaseConfig)

	return newEthBridge(srcCfg, dstAddr, src.Endpoint, l, baseDir, src.Options)
}

func NewSender(srcAddr types.BtpAddress, dstCfg link.ChainConfig, baseDir string, l log.Logger) (types.Sender, error) {
	dst := dstCfg.(chain.BaseConfig)
	w, err := newWallet(dst.KeyStorePass, dst.KeySecret, dst.KeyStore)
	if err != nil {
		return nil, err
	}

	return newSender(srcAddr, dst, w, dst.Endpoint, dst.Options, l), nil
}

func newWallet(passwd, secret string, keyStorePath string) (types.Wallet, error) {
	if keyStore, err := os.ReadFile(keyStorePath); err != nil {
		return nil, fmt.Errorf("fail to open KeyStore file path=%s", keyStorePath)
	} else {
		pw, err := resolvePassword(secret, passwd)
		if err != nil {
			return nil, err
		}
		return wallet.DecryptKeyStore(keyStore, pw)
	}
}

func resolvePassword(keySecret, keyStorePass string) ([]byte, error) {
	if keySecret != "" {
		return os.ReadFile(keySecret)
	} else {
		if keyStorePass != "" {
			return []byte(keyStorePass), nil
		}
	}
	return nil, nil
}
