package relay

import (
	"github.com/icon-project/btp2/common/link"
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
	lfs []linkFactory
}

// TODO rename / new go file??
func NewRelay(cfg *link.Config, modLevels map[string]string) (*Relay, error) {

	r := &Relay{
		lfs: make([]linkFactory, 0),
	}
	switch cfg.Direction {
	case FrontDirection:
		l, s, err := link.NewLinkFactory(cfg.Src, cfg.Dst, cfg.RelayConfig, modLevels)
		if err != nil {
			return nil, err
		}
		r.lfs = append(r.lfs, linkFactory{link: l, sender: s})
	case ReverseDirection:
		l, s, err := link.NewLinkFactory(cfg.Dst, cfg.Src, cfg.RelayConfig, modLevels)
		if err != nil {
			return nil, err
		}
		r.lfs = append(r.lfs, linkFactory{link: l, sender: s})
	case BothDirection:
		frontL, frontS, err := link.NewLinkFactory(cfg.Src, cfg.Dst, cfg.RelayConfig, modLevels)
		if err != nil {
			return nil, err
		}
		r.lfs = append(r.lfs, linkFactory{link: frontL, sender: frontS})

		reverseL, reverseS, err := link.NewLinkFactory(cfg.Dst, cfg.Src, cfg.RelayConfig, modLevels)
		if err != nil {
			return nil, err
		}
		r.lfs = append(r.lfs, linkFactory{link: reverseL, sender: reverseS})
	}

	return r, nil
}

func (r *Relay) Start() error {
	linkErrCh := make(chan error)

	for _, lf := range r.lfs {
		if err := link.Start(lf.link, lf.sender, linkErrCh); err != nil {
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
