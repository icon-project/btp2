package linkfactory

import (
	"fmt"
	stdlog "log"
	"path"

	"github.com/icon-project/btp2/chain"
	"github.com/icon-project/btp2/chain/ethbr"
	"github.com/icon-project/btp2/chain/icon"
	"github.com/icon-project/btp2/common/config"
	"github.com/icon-project/btp2/common/link"
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

type LinkInfo struct {
	link types.Link
	l    log.Logger
}

func (l *LinkInfo) GetLogger() log.Logger {
	return l.l
}

func (l *LinkInfo) Start(sender types.Sender, errCh chan error) error {
	go func() {
		err := l.link.Start(sender)
		select {
		case errCh <- err:
		default:
		}
	}()

	return nil
}

func (l *LinkInfo) Stop() {
	l.link.Stop()
}

func NewLinkInfo(srcCfg chain.BaseConfig, dstCfg chain.BaseConfig, lc LogConfig, fc config.FileConfig, modLevels map[string]string) (*LinkInfo, error) {
	var lk types.Link
	l := setLogger(srcCfg, lc, fc, modLevels)
	l.Debugln(fc.FilePath, fc.BaseDir)
	if fc.BaseDir == "" {
		fc.BaseDir = path.Join(".", ".relay", srcCfg.Address.NetworkAddress())
	}

	r := newReceiver(srcCfg, dstCfg, l)
	lk = link.NewLink(srcCfg, dstCfg, r, l)
	linkInfo := &LinkInfo{
		link: lk,
		l:    l,
	}
	return linkInfo, nil
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

type LinkFactory struct {
	linkInfos []*LinkInfo
	senders   []types.Sender
}

func (l *LinkFactory) Start() error {
	linkErrCh := make(chan error)

	for i, linkInfo := range l.linkInfos {
		if err := linkInfo.Start(l.senders[i], linkErrCh); err != nil {
			return err
		}
	}

	for {
		select {
		case err := <-linkErrCh:
			if err != nil {
				return err
			}
		}
	}
}

func (l *LinkFactory) Stop() {
	for i, linkInfo := range l.linkInfos {
		linkInfo.Stop()
		l.senders[i].Stop()
	}
}

func NewLinkFactory(cfg *Config, modLevels map[string]string) (*LinkFactory, error) {
	lf := &LinkFactory{
		linkInfos: make([]*LinkInfo, 0),
		senders:   make([]types.Sender, 0),
	}

	switch cfg.Direction {
	case FrontDirection:
		linkInfo, err := NewLinkInfo(cfg.Src, cfg.Dst, cfg.LogConfig, cfg.FileConfig, modLevels)
		if err != nil {
			return nil, err
		}
		sender, err := NewSender(cfg.Src, cfg.Dst, linkInfo.GetLogger())
		if err != nil {
			return nil, err
		}
		lf.linkInfos = append(lf.linkInfos, linkInfo)
		lf.senders = append(lf.senders, sender)
	case ReverseDirection:
		linkInfo, err := NewLinkInfo(cfg.Dst, cfg.Src, cfg.LogConfig, cfg.FileConfig, modLevels)
		if err != nil {
			return nil, err
		}
		sender, err := NewSender(cfg.Dst, cfg.Src, linkInfo.GetLogger())
		if err != nil {
			return nil, err
		}
		lf.linkInfos = append(lf.linkInfos, linkInfo)
		lf.senders = append(lf.senders, sender)

	case BothDirection:
		srcLinkInfo, err := NewLinkInfo(cfg.Src, cfg.Dst, cfg.LogConfig, cfg.FileConfig, modLevels)
		if err != nil {
			return nil, err
		}
		srcSender, err := NewSender(cfg.Src, cfg.Dst, srcLinkInfo.GetLogger())
		if err != nil {
			return nil, err
		}
		lf.linkInfos = append(lf.linkInfos, srcLinkInfo)
		lf.senders = append(lf.senders, srcSender)

		dstLinkInfo, err := NewLinkInfo(cfg.Dst, cfg.Src, cfg.LogConfig, cfg.FileConfig, modLevels)
		if err != nil {
			return nil, err
		}
		dstSender, err := NewSender(cfg.Dst, cfg.Src, dstLinkInfo.GetLogger())
		if err != nil {
			return nil, err
		}
		lf.linkInfos = append(lf.linkInfos, dstLinkInfo)
		lf.senders = append(lf.senders, dstSender)

	default:
		return nil, fmt.Errorf("Not supported direction:%s", cfg.Direction)
	}

	return lf, nil
}

func newReceiver(srcCfg chain.BaseConfig, dstCfg chain.BaseConfig, l log.Logger) link.Receiver {
	var receiver link.Receiver

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
	switch dstCfg.Address.BlockChain() {
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

func setLogger(srcCfg chain.BaseConfig, lc LogConfig, fc config.FileConfig, modLevels map[string]string) log.Logger {
	l := log.WithFields(log.Fields{log.FieldKeyChain: fmt.Sprintf("%s", srcCfg.Address.NetworkID())})
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
