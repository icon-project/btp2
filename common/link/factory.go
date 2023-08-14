package link

import (
	"encoding/json"
	"fmt"

	"github.com/icon-project/btp2/common/errors"
	"github.com/icon-project/btp2/common/log"
	"github.com/icon-project/btp2/common/types"
)

type Factory struct {
	Type             string
	ParseChainConfig func(raw json.RawMessage) (ChainConfig, error)
	NewReceiver      func(srcCfg ChainConfig, dstAddr types.BtpAddress, baseDir string, l log.Logger) (Receiver, error)
	NewLink          func(srcCfg ChainConfig, dstAddr types.BtpAddress, baseDir string, l log.Logger) (types.Link, error)
	NewSender        func(srcAddr types.BtpAddress, dstCfg ChainConfig, baseDir string, l log.Logger) (types.Sender, error)
}

var factories = map[string]*Factory{}

func RegisterFactory(f *Factory) {
	if f.NewLink == nil && f.NewReceiver == nil {
		panic("NewLink and NewReader are empty")
	}
	if _, ok := factories[f.Type]; ok {
		panic(fmt.Sprintf("Duplicate factory registration for %s", f.Type))
	}
	factories[f.Type] = f
}

func CreateLink(srcRaw, dstRaw json.RawMessage, baseDir string, l log.Logger) (types.Link, error) {

	var srcCfgCommon ChainConfigCommon
	if err := json.Unmarshal(srcRaw, &srcCfgCommon); err != nil {
		return nil, err
	}
	srcType := srcCfgCommon.GetType()
	if len(srcType) == 0 {
		return nil, errors.IllegalArgumentError.New("empty type in source config")

	}
	//Sender
	var dstCfgCommon ChainConfigCommon
	if err := json.Unmarshal(dstRaw, &dstCfgCommon); err != nil {
		return nil, err
	}

	if f, ok := factories[srcType]; ok {
		srcCfg, err := f.ParseChainConfig(srcRaw)
		if err != nil {
			return nil, err
		}
		if f.NewLink != nil {
			return f.NewLink(srcCfg, dstCfgCommon.GetAddress(), baseDir, l)
		} else {
			receiver, err := f.NewReceiver(srcCfg, dstCfgCommon.GetAddress(), baseDir, l)
			if err != nil {
				return nil, err
			}
			return NewLink(srcCfg, receiver, l), nil
		}

	} else {
		return nil, errors.NotFoundError.Errorf("UnknownSourceType(type=%s)", srcType)
	}
}

func CreateSender(srcRaw, dstRaw json.RawMessage, baseDir string, l log.Logger) (types.Sender, error) {
	var srcCfgCommon ChainConfigCommon
	if err := json.Unmarshal(srcRaw, &srcCfgCommon); err != nil {
		return nil, err
	}
	var dstCfgCommon ChainConfigCommon
	if err := json.Unmarshal(dstRaw, &dstCfgCommon); err != nil {
		return nil, err
	}
	dstType := dstCfgCommon.GetType()
	if len(dstType) == 0 {
		return nil, errors.IllegalArgumentError.New("empty type in dst config")
	}

	if f, ok := factories[dstType]; ok {
		dstCfg, err := f.ParseChainConfig(dstRaw)
		if err != nil {
			return nil, err
		}

		return f.NewSender(srcCfgCommon.GetAddress(), dstCfg, baseDir, l)
	} else {
		return nil, errors.NotFoundError.Errorf("UnknownSourceType(type=%s)", dstType)
	}
}
