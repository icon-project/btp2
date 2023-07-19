package link

import (
	"github.com/icon-project/btp2/common/config"
	"github.com/icon-project/btp2/common/log"
	"github.com/icon-project/btp2/common/types"
)

type RelayConfig struct {
	Direction         string               `json:"direction"`
	config.FileConfig `json:",squash"`     //instead of `mapstructure:",squash"`
	LogLevel          string               `json:"log_level"`
	ConsoleLevel      string               `json:"console_level"`
	LogForwarder      *log.ForwarderConfig `json:"log_forwarder,omitempty"`
	LogWriter         *log.WriterConfig    `json:"log_writer,omitempty"`
}

type ChainConfig interface {
	GetAddress() types.BtpAddress
	GetMode() string
	GetNetworkID() string
}

type ChainsConfigs struct {
	Src map[string]interface{} `json:"src"`
	Dst map[string]interface{} `json:"dst"`
}

type Config struct {
	RelayConfig   `json:"relay_config"`
	ChainsConfigs `json:"chains_config"` //instead of `mapstructure:",squash"`
}
