package link

import (
	"encoding/json"

	"github.com/icon-project/btp2/common/types"
)

type ChainConfig interface {
	GetAddress() types.BtpAddress
	GetType() string
}

type ChainConfigCommon struct {
	Type    string           `json:"type"`
	Address types.BtpAddress `json:"address"`
}

func (c *ChainConfigCommon) GetAddress() types.BtpAddress {
	return c.Address
}

func (c *ChainConfigCommon) GetType() string {
	return c.Type
}

type ChainsConfigs struct {
	Src json.RawMessage `json:"src"`
	Dst json.RawMessage `json:"dst"`
}
