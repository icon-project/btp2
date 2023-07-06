package linkfactory

import (
	"github.com/icon-project/btp2/chain"
	"github.com/icon-project/btp2/common/config"
	"github.com/icon-project/btp2/common/log"
)

type LinkConfig struct {
	config.FileConfig `json:",squash"` //instead of `mapstructure:",squash"`
	Src               chain.BaseConfig `json:"src"`
	Dst               chain.BaseConfig `json:"dst"`
	Direction         string           `json:"direction"` //front, reverse, both
}

type LogConfig struct {
	LogLevel     string               `json:"log_level"`
	ConsoleLevel string               `json:"console_level"`
	LogForwarder *log.ForwarderConfig `json:"log_forwarder,omitempty"`
	LogWriter    *log.WriterConfig    `json:"log_writer,omitempty"`
}

type Config struct {
	LinkConfig `json:",squash"` //instead of `mapstructure:",squash"`
	LogConfig  `json:",squash"` //instead of `mapstructure:",squash"`
}
