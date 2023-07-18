package bridge

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/icon-project/btp2/chain"
	"github.com/icon-project/btp2/chain/icon"
	"github.com/icon-project/btp2/common/config"
	"github.com/icon-project/btp2/common/link"
	"github.com/icon-project/btp2/common/log"
	"github.com/icon-project/btp2/common/types"
	"github.com/icon-project/btp2/common/wallet"
)

func RegisterIconBridge() {
	link.RegisterFactory(&link.Factory{
		GetChainConfig: GetChainConfig,
		CheckConfig:    CheckConfig,
		NewReceiver:    NewReceiver,
		NewSender:      NewSender,
	})
}

func GetChainConfig(dict map[string]interface{}) (link.ChainConfig, error) {
	cfg := chain.BaseConfig{}

	jsonbody, err := json.Marshal(dict)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(jsonbody, &cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func CheckConfig(cfg link.ChainConfig) bool {
	baseCfg, ok := cfg.(chain.BaseConfig)

	if !ok {
		return false
	}

	if baseCfg.ChainId == "icon_bridge" {
		return true
	}

	return false
}

func NewReceiver(srcCfg, dstCfg link.ChainConfig, fileCfg config.FileConfig, l log.Logger) (link.Receiver, error) {
	src := srcCfg.(chain.BaseConfig)

	return newBridge(src, dstCfg.GetAddress(), src.Endpoint, l)
}

func NewSender(srcCfg, dstCfg link.ChainConfig, l log.Logger) (types.Sender, error) {
	dst := dstCfg.(chain.BaseConfig)

	//TODO refactoring
	w, err := newWallet(dst.KeyStorePass, dst.KeySecret, dst.KeyStoreData)
	if err != nil {
		return nil, err
	}

	return icon.NewSender(srcCfg.GetAddress(), dst, w, dst.Endpoint, dst.Options, l), nil
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
		} else {
			//TODO add error message
			return nil, fmt.Errorf("")
		}
	}
}
