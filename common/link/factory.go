package link

import (
	"encoding/json"

	"github.com/icon-project/btp2/common/log"
	"github.com/icon-project/btp2/common/types"
)

type Factory struct {
	Type             string
	ParseChainConfig func(raw json.RawMessage) (ChainConfig, error)
	NewReceiver      func(srcCfg ChainConfig, dstAddr types.BtpAddress, baseDir string, l log.Logger) (Receiver, error)
	NewLink          func(srcCfg ChainConfig, dstAddr types.BtpAddress, baseDir string, l log.Logger) (types.Link, error)
	NewSender        func(srcAddr types.BtpAddress, dstCfg ChainConfig, l log.Logger) (types.Sender, error)
}

var factories []*Factory

func RegisterFactory(f *Factory) {
	if f != nil {
		if checkDuplicates(f) {
			return //TODO err??
		}
		factories = append(factories, f)
	}
}

func checkDuplicates(f *Factory) bool {
	for _, cf := range factories {
		if cf.Type == f.Type {
			return true
		}
	}
	return false
}

func CreateLink(srcRaw, dstRaw json.RawMessage, l log.Logger, baseDir string) (types.Link, error) {

	//Sender
	var dstCfgCommon ChainConfigCommon
	if err := json.Unmarshal(dstRaw, &dstCfgCommon); err != nil {
		return nil, err
	}

	for _, f := range factories {
		srcCfg, err := f.ParseChainConfig(srcRaw)
		if err != nil {
			return nil, err
		}
		if srcCfg != nil {
			link, err := f.NewLink(srcCfg, dstCfgCommon.GetAddress(), baseDir, l)
			if err != nil {
				return nil, err
			}

			if link == nil {
				receiver, err := f.NewReceiver(srcCfg, dstCfgCommon.GetAddress(), baseDir, l)
				if err != nil {
					return nil, err
				}

				link = NewLink(srcCfg, receiver, l)
			}
			return link, nil
		}
	}

	return nil, nil
}

func CreateSender(srcRaw, dstRaw json.RawMessage, l log.Logger) (types.Sender, error) {
	//Sender
	var srcCfgCommon ChainConfigCommon
	if err := json.Unmarshal(srcRaw, &srcCfgCommon); err != nil {
		return nil, err
	}

	for _, f := range factories {
		dstCfg, err := f.ParseChainConfig(dstRaw)
		if err != nil {
			return nil, err
		}
		if dstCfg != nil {
			s, err := f.NewSender(srcCfgCommon.GetAddress(), dstCfg, l)
			if err != nil {
				return nil, err
			}
			return s, nil
		}
	}

	return nil, nil
}
