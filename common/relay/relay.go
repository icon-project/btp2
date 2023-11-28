package relay

import (
	"encoding/json"
	"fmt"
	stdlog "log"

	"github.com/icon-project/btp2/common/link"
	"github.com/icon-project/btp2/common/log"
	"github.com/icon-project/btp2/common/types"
)

const (
	BothDirection    = "both"
	FrontDirection   = "front"
	ReverseDirection = "reverse"
)

type linkFactory struct {
	link   types.Link
	sender types.Sender
}
type Relay struct {
	lfs []*linkFactory
}

func NewRelay(cfg *Config, modLevels map[string]string) (*Relay, error) {

	r := &Relay{
		lfs: make([]*linkFactory, 0),
	}
	switch cfg.Direction {
	case FrontDirection:
		lf, err := newLinkFactory(cfg.Src, cfg.Dst, cfg.RelayConfig, modLevels)
		if err != nil {
			return nil, err
		}

		r.lfs = append(r.lfs, lf)
	case ReverseDirection:
		lf, err := newLinkFactory(cfg.Dst, cfg.Src, cfg.RelayConfig, modLevels)
		if err != nil {
			return nil, err
		}

		r.lfs = append(r.lfs, lf)

	case BothDirection:
		frontLf, err := newLinkFactory(cfg.Src, cfg.Dst, cfg.RelayConfig, modLevels)
		if err != nil {
			return nil, err
		}

		r.lfs = append(r.lfs, frontLf)

		reverseLf, err := newLinkFactory(cfg.Dst, cfg.Src, cfg.RelayConfig, modLevels)
		if err != nil {
			return nil, err
		}

		r.lfs = append(r.lfs, reverseLf)
	}

	return r, nil
}

func newLinkFactory(srcRaw, dstRaw json.RawMessage, relayCfg RelayConfig, modLevels map[string]string) (*linkFactory, error) {
	logger, err := setLogger(srcRaw, relayCfg, modLevels)
	if err != nil {
		return nil, err
	}

	l, err := link.CreateLink(srcRaw, dstRaw, relayCfg.BaseDir, logger)
	if err != nil {
		return nil, err
	}

	s, err := link.CreateSender(srcRaw, dstRaw, relayCfg.BaseDir, logger)
	if err != nil {
		return nil, err
	}

	return &linkFactory{link: l, sender: s}, nil
}

func (r *Relay) Start() error {
	linkErrCh := make(chan error)
	for _, lf := range r.lfs {
		if err := start(lf.link, lf.sender, linkErrCh); err != nil {
			return err
		}
	}

	for {
		select {
		case err := <-linkErrCh:
			if err != nil {
				log.GlobalLogger().Debugln("Relay error :", err)
				return err
			}
		}
	}
}

func start(link types.Link, sender types.Sender, errCh chan error) error {
	if err := link.Start(sender, errCh); err != nil {
		return err
	}
	return nil
}

func setLogger(srcRaw json.RawMessage, lc RelayConfig, modLevels map[string]string) (log.Logger, error) {
	var srcCfgCommon link.ChainConfigCommon
	if err := json.Unmarshal(srcRaw, &srcCfgCommon); err != nil {
		return nil, err
	}

	l := log.WithFields(log.Fields{log.FieldKeyChain: fmt.Sprintf("%s", srcCfgCommon.GetAddress().NetworkID())})
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

	return l, nil
}
