package relay

import (
	"github.com/icon-project/btp2/common/config"
	"github.com/icon-project/btp2/common/link"
	"github.com/icon-project/btp2/common/log"
)

type RelayConfig struct {
	Direction         string               `json:"direction"`
	config.FileConfig `json:",squash"`     //instead of `mapstructure:",squash"`
	LogLevel          string               `json:"log_level"`
	ConsoleLevel      string               `json:"console_level"`
	LogForwarder      *log.ForwarderConfig `json:"log_forwarder,omitempty"`
	LogWriter         *log.WriterConfig    `json:"log_writer,omitempty"`
}

type Config struct {
	RelayConfig        `json:"relay_config"`
	link.ChainsConfigs `json:"chains_config"` //instead of `mapstructure:",squash"`
}
