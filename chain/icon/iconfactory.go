package icon

//
//import (
//	"encoding/json"
//	"fmt"
//	"os"
//
//	"github.com/icon-project/btp2/chain"
//	"github.com/icon-project/btp2/chain/icon/bridge"
//	"github.com/icon-project/btp2/chain/icon/btp2"
//	"github.com/icon-project/btp2/common/link"
//	"github.com/icon-project/btp2/common/log"
//	"github.com/icon-project/btp2/common/types"
//	"github.com/icon-project/btp2/common/wallet"
//)
//
//const (
//	BRIDGE    = "bridge"
//	TRUSTLESS = "trustless"
//)
//
//// func NewReceiver(srcCfg, dstCfg chain.BaseConfig, l log.Logger) link.Receiver {
//func NewReceiver(srcCfg, dstCfg interface{}, l log.Logger) link.Receiver {
//	src := srcCfg.(chain.BaseConfig)
//	dst := dstCfg.(chain.BaseConfig)
//	var receiver link.Receiver
//
//	src.Name
//	switch srcCfg.RelayMode {
//	case BRIDGE:
//		receiver = bridge.NewBridge(srcCfg.Address, dstCfg.Address, srcCfg.Endpoint, l)
//	case TRUSTLESS:
//		receiver = btp2.NewBTP2(srcCfg.Address, dstCfg.Address, srcCfg.Endpoint, l)
//	default:
//		l.Panicf("Not supported for relay mod:%s", srcCfg.RelayMode)
//	}
//	return receiver
//}
//
//func NewSender(srcAddr types.BtpAddress, cfg chain.BaseConfig, l log.Logger) (types.Sender, error) {
//	//TODO refactoring
//	w, err := Wallet(cfg.KeyStorePass, cfg.KeySecret, cfg.KeyStoreData)
//	if err != nil {
//		return nil, err
//	}
//
//	return newSender(srcAddr, cfg.Address, w, cfg.Endpoint, cfg.Options, l), nil
//}
//
//func Wallet(passwd, secret string, keyStore json.RawMessage) (types.Wallet, error) {
//	pw, err := resolvePassword(secret, passwd)
//	if err != nil {
//		return nil, err
//	}
//	return wallet.DecryptKeyStore(keyStore, pw)
//}
//
//func resolvePassword(keySecret, keyStorePass string) ([]byte, error) {
//	if keySecret != "" {
//		return os.ReadFile(keySecret)
//	} else {
//		if keyStorePass != "" {
//			return []byte(keyStorePass), nil
//		} else {
//			//TODO add error message
//			return nil, fmt.Errorf("")
//		}
//	}
//}
