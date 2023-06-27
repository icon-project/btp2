package link

import (
	"fmt"
	stdlog "log"
	"path"

	"github.com/icon-project/btp2/chain"
	"github.com/icon-project/btp2/chain/ethbr"
	"github.com/icon-project/btp2/chain/icon"
	"github.com/icon-project/btp2/common/config"
	"github.com/icon-project/btp2/common/log"
	"github.com/icon-project/btp2/common/types"
)

const (
	BothDirection    = "both"
	FrontDirection   = "front"
	ReverseDirection = "reverse"

	ICON    = "icon"
	ETH     = "eth"
	ETH2    = "eth2"
	BSC     = "bsc"
	HARDHAT = "hardhat"
)

type LinkFactory struct {
	link types.Link
	l    log.Logger
}

func (l *LinkFactory) GetLogger() log.Logger {
	return l.l
}

func (l *LinkFactory) Start(sender types.Sender) error {
	linkErrCh := make(chan error)
	go func() {
		err := l.link.Start(sender)
		select {
		case linkErrCh <- err:
		default:
		}
	}()

	return nil
}

//func (l *LinkFactory) Start(srcCfg chain.BaseConfig, dstCfg chain.BaseConfig) error {
//	linkErrCh := make(chan error)
//	go func() {
//		s, err := NewSender(srcCfg, dstCfg, l.l)
//		if err != nil {
//			linkErrCh <- err
//		}
//
//		err = l.link.Start(s)
//		if err != nil {
//			linkErrCh <- err
//		}
//		select {
//		case linkErrCh <- err:
//		default:
//		}
//	}()
//
//	return nil
//}

func NewLinkFactory(cfg *Config, modLevels map[string]string) ([]*LinkFactory, error) {
	linkFactorys := make([]*LinkFactory, 0)
	switch cfg.Direction {
	case FrontDirection:
		lf, err := newLinkFactory(cfg.Src, cfg.Dst, cfg.LogConfig, cfg.FileConfig, modLevels)
		if err != nil {
			return nil, err
		}

		linkFactorys = append(linkFactorys, lf)

	case ReverseDirection:
		lf, err := newLinkFactory(cfg.Dst, cfg.Src, cfg.LogConfig, cfg.FileConfig, modLevels)
		if err != nil {
			return nil, err
		}
		linkFactorys = append(linkFactorys, lf)

	case BothDirection:
		srcLf, err := newLinkFactory(cfg.Src, cfg.Dst, cfg.LogConfig, cfg.FileConfig, modLevels)
		if err != nil {
			return nil, err
		}

		linkFactorys = append(linkFactorys, srcLf)

		dstLf, err := newLinkFactory(cfg.Dst, cfg.Src, cfg.LogConfig, cfg.FileConfig, modLevels)
		if err != nil {
			return nil, err
		}
		linkFactorys = append(linkFactorys, dstLf)

	default:
		return nil, fmt.Errorf("Not supported direction:%s", cfg.Direction)
	}

	return linkFactorys, nil
}

func newLinkFactory(srcCfg chain.BaseConfig, dstCfg chain.BaseConfig, lc LogConfig, fc config.FileConfig, modLevels map[string]string) (*LinkFactory, error) {
	var lk types.Link
	l := setLogger(srcCfg, dstCfg, lc, fc, modLevels)
	l.Debugln(fc.FilePath, fc.BaseDir)
	if fc.BaseDir == "" {
		fc.BaseDir = path.Join(".", ".relay", srcCfg.Address.NetworkAddress())
	}

	r := newReceiver(srcCfg, dstCfg, l)
	lk = NewLink(srcCfg, dstCfg, r, l)
	lf := &LinkFactory{
		link: lk,
		l:    l,
	}
	return lf, nil
}

func NewLink(srcCfg, dstCfg chain.BaseConfig, r Receiver, l log.Logger) types.Link {
	link := &Link{
		l:      l.WithFields(log.Fields{log.FieldKeyChain: fmt.Sprintf("%s", dstCfg.Address.NetworkID())}),
		srcCfg: srcCfg,
		dstCfg: dstCfg,
		r:      r,
		rms:    make([]*relayMessage, 0),
		rss:    make([]ReceiveStatus, 0),
		rmi: &relayMessageItem{
			rmis: make([][]RelayMessageItem, 0),
			size: 0,
		},
		blsChannel: make(chan *types.BMCLinkStatus),
		relayState: RUNNING,
	}
	link.rmi.rmis = append(link.rmi.rmis, make([]RelayMessageItem, 0))
	return link
}

func newReceiver(srcCfg chain.BaseConfig, dstCfg chain.BaseConfig, l log.Logger) Receiver {
	var receiver Receiver

	switch srcCfg.Address.BlockChain() {
	case ICON:
		receiver = icon.NewReceiver(srcCfg, dstCfg, l)
	case ETH:
		fallthrough
	case ETH2:
		fallthrough
	case BSC:
		fallthrough
	case HARDHAT:
		receiver = ethbr.NewReceiver(srcCfg, dstCfg, l)
	default:
		l.Fatalf("Not supported for chain:%s", srcCfg.Address.BlockChain())
		return nil
	}
	return receiver
}

func NewSender(srcCfg, dstCfg chain.BaseConfig, l log.Logger) (types.Sender, error) {
	var sender types.Sender
	var err error
	switch srcCfg.Address.BlockChain() {
	case ICON:
		sender, err = icon.NewSender(srcCfg.Address, dstCfg, l)
		if err != nil {
			return nil, err
		}
	case ETH:
		fallthrough
	case ETH2:
		fallthrough
	case BSC:
		fallthrough
	case HARDHAT:
		sender, err = ethbr.NewSender(srcCfg.Address, dstCfg, l)
		if err != nil {
			return nil, err
		}
	default:
		l.Fatalf("Not supported for chain:%s", srcCfg.Address.BlockChain())
	}

	return sender, nil
}

func setLogger(srcCfg, dstCfg chain.BaseConfig, lc LogConfig, fc config.FileConfig, modLevels map[string]string) log.Logger {
	l := log.WithFields(log.Fields{log.FieldKeyWallet: srcCfg.Address.NetworkID() + "2" + dstCfg.Address.NetworkAddress()})
	log.SetGlobalLogger(l)
	stdlog.SetOutput(l.WriterLevel(log.WarnLevel))
	if lc.LogWriter != nil {
		if lc.LogWriter.Filename == "" {
			log.Debugln("LogWriterConfig filename is empty string, will be ignore")
		} else {
			var lwCfg log.WriterConfig
			lwCfg = *lc.LogWriter
			lwCfg.Filename = fc.ResolveAbsolute(lwCfg.Filename)
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
