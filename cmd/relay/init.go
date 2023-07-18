package main

import (
	"github.com/icon-project/btp2/chain/ethbr"
	"github.com/icon-project/btp2/chain/icon/bridge"
	"github.com/icon-project/btp2/chain/icon/btp2"
)

func init() {
	bridge.RegisterIconBridge()
	btp2.RegisterIconBtp2()
	ethbr.RegisterIconEthBr()
}
