package link

import (
	"fmt"
	stdlog "log"

	"github.com/icon-project/btp2/common/config"
	"github.com/icon-project/btp2/common/log"
	"github.com/icon-project/btp2/common/types"
)

type Factory struct {
	GetChainConfig func(dict map[string]interface{}) (ChainConfig, error)
	CheckConfig    func(cfg ChainConfig) bool
	NewReceiver    func(srcCfg, dstCfg ChainConfig, fileCfg config.FileConfig, l log.Logger) (Receiver, error)
	NewSender      func(srcCfg, dstCfg ChainConfig, l log.Logger) (types.Sender, error)
}

var factories []*Factory

func RegisterFactory(f *Factory) {
	if f != nil {
		factories = append(factories, f)
	}
}

func ComposeLink(srcCfg, dstCfg map[string]interface{}, relayCfg RelayConfig, modLevels map[string]string) (types.Link, types.Sender, error) {
	var err error

	var srcChainCfg ChainConfig
	var dstChainCfg ChainConfig

	var srcFactory *Factory
	var dstFactory *Factory

	//Receiver
	for _, factory := range factories {
		srcChainCfg, err = factory.GetChainConfig(srcCfg)
		if err != nil {
			return nil, nil, err
		}
		if factory.CheckConfig(srcChainCfg) {
			srcFactory = factory
			break
		}
	}

	if srcChainCfg == nil {
		return nil, nil, fmt.Errorf("not supported source chain")
	}

	//Sender
	for _, factory := range factories {
		dstChainCfg, err = factory.GetChainConfig(dstCfg)
		if err != nil {
			return nil, nil, err
		}
		if factory.CheckConfig(dstChainCfg) {
			dstFactory = factory
			break
		}
	}

	if dstChainCfg == nil {
		return nil, nil, fmt.Errorf("not supported destination chain")
	}

	logger := setLogger(srcChainCfg, relayCfg, modLevels)
	logger.Debugln(relayCfg.FileConfig.FilePath, relayCfg.FileConfig.BaseDir)

	//new receiver
	r, err := srcFactory.NewReceiver(srcChainCfg, dstChainCfg, relayCfg.FileConfig, logger)
	if err != nil {
		return nil, nil, err
	}

	//new sender
	s, err := dstFactory.NewSender(srcChainCfg, dstChainCfg, logger)
	if err != nil {
		return nil, nil, err
	}

	l := NewLink(srcChainCfg, dstChainCfg, r, logger)
	return l, s, nil
}

func Start(link types.Link, sender types.Sender, errCh chan error) error {
	go func() {
		err := link.Start(sender)
		select {
		case errCh <- err:
		default:
		}
	}()

	return nil
}

func setLogger(srcCfg ChainConfig, lc RelayConfig, modLevels map[string]string) log.Logger {
	l := log.WithFields(log.Fields{log.FieldKeyChain: fmt.Sprintf("%s", srcCfg.GetNetworkID())})
	log.SetGlobalLogger(l)
	stdlog.SetOutput(l.WriterLevel(log.WarnLevel))
	if lc.LogWriter != nil {
		if lc.LogWriter.Filename == "" {
			log.Debugln("LogWriterConfig filename is empty string, will be ignore")
		} else {
			var lwCfg log.WriterConfig
			lwCfg = *lc.LogWriter
			lwCfg.Filename = lc.ResolveAbsolute(lwCfg.Filename)
			w, err := log.NewWriter(&lwCfg)
			if err != nil {
				log.Panicf("Fail to make writer err=%+v", err)
			}
			err = l.SetFileWriter(w)
			if err != nil {
				log.Panicf("Fail to set file l err=%+v", err)
			}
		}
	}

	if lv, err := log.ParseLevel(lc.LogLevel); err != nil {
		log.Panicf("Invalid log_level=%s", lc.LogLevel)
	} else {
		l.SetLevel(lv)
	}
	if lv, err := log.ParseLevel(lc.ConsoleLevel); err != nil {
		log.Panicf("Invalid console_level=%s", lc.ConsoleLevel)
	} else {
		l.SetConsoleLevel(lv)
	}

	for mod, lvStr := range modLevels {
		if lv, err := log.ParseLevel(lvStr); err != nil {
			log.Panicf("Invalid mod_level mod=%s level=%s", mod, lvStr)
		} else {
			l.SetModuleLevel(mod, lv)
		}
	}

	if lc.LogForwarder != nil {
		if lc.LogForwarder.Vendor == "" && lc.LogForwarder.Address == "" {
			log.Debugln("LogForwarderConfig vendor and address is empty string, will be ignore")
		} else {
			if err := log.AddForwarder(lc.LogForwarder); err != nil {
				log.Fatalf("Invalid log_forwarder err:%+v", err)
			}
		}
	}

	return l
}
