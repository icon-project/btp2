package ethbr

import (
	"encoding/json"
	"os"

	"github.com/icon-project/btp2/chain"
	"github.com/icon-project/btp2/common/link"
	"github.com/icon-project/btp2/common/log"
	"github.com/icon-project/btp2/common/types"
	"github.com/icon-project/btp2/common/wallet"
)

const TYPE = "eth-bridge-solidity"

func RegisterEthBridge() {
	link.RegisterFactory(&link.Factory{
		Type:             TYPE,
		ParseChainConfig: ParseChainConfig,
		NewLink:          NewLink,
		NewReceiver:      NewReceiver,
		NewSender:        NewSender,
	})
}

func ParseChainConfig(raw json.RawMessage) (link.ChainConfig, error) {
	cfg := chain.BaseConfig{}

	jsonbody, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(jsonbody, &cfg); err != nil {
		return nil, err
	}

	//TODO add check
	if cfg.Type == TYPE {
		return cfg, nil
	}

	return nil, nil
}

func NewLink(srcCfg link.ChainConfig, dstAddr types.BtpAddress, baseDir string, l log.Logger) (types.Link, error) {
	src := srcCfg.(chain.BaseConfig)

	r, err := newEthBridge(srcCfg, dstAddr, src.Endpoint, l, baseDir, src.Options)
	if err != nil {
		return nil, err
	}

	link := link.NewLink(srcCfg, r, l)
	return link, nil
}

func NewReceiver(srcCfg link.ChainConfig, dstAddr types.BtpAddress, baseDir string, l log.Logger) (link.Receiver, error) {
	src := srcCfg.(chain.BaseConfig)

	return newEthBridge(srcCfg, dstAddr, src.Endpoint, l, baseDir, src.Options)
}

func NewSender(srcAddr types.BtpAddress, dstCfg link.ChainConfig, l log.Logger) (types.Sender, error) {
	dst := dstCfg.(chain.BaseConfig)
	w, err := newWallet(dst.KeyStorePass, dst.KeySecret, dst.KeyStoreData)
	if err != nil {
		return nil, err
	}

	return newSender(srcAddr, dst, w, dst.Endpoint, dst.Options, l), nil
}

func newWallet(passwd, secret string, keyStore json.RawMessage) (types.Wallet, error) {
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
		}
	}
	return nil, nil
}
