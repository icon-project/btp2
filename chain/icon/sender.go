/*
* Copyright 2021 ICON Foundation
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*
*     http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
 */

package icon

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/icon-project/btp2/chain"
	"github.com/icon-project/btp2/chain/icon/client"
	"github.com/icon-project/btp2/common/errors"
	"github.com/icon-project/btp2/common/jsonrpc"
	"github.com/icon-project/btp2/common/link"
	"github.com/icon-project/btp2/common/log"
	"github.com/icon-project/btp2/common/types"
)

const (
	txMaxDataSize                 = 524288 //512 * 1024 // 512kB
	txOverheadScale               = 0.37   //base64 encoding overhead 0.36, rlp and other fields 0.01
	DefaultGetRelayResultInterval = time.Second
	DefaultRelayReSendInterval    = time.Second
	DefaultStepLimit              = 0x9502f900 //maxStepLimit(invoke), refer https://www.icondev.io/docs/step-estimation
	MaxQueueSize                  = 100
	TransactionResultRetryLimit   = 5
)

var (
	txSizeLimit = int(math.Ceil(txMaxDataSize / (1 + txOverheadScale)))
)

type queue struct {
	values []*relayMessageTx
}

type relayMessageTx struct {
	id     string
	txHash []byte
}

func newQueue() *queue {
	queue := &queue{}
	return queue
}

func (q *queue) enqueue(id string, txHash []byte) error {
	if MaxQueueSize <= len(q.values) {
		return fmt.Errorf("queue full")
	}
	q.values = append(q.values,
		&relayMessageTx{
			id:     id,
			txHash: txHash,
		})
	return nil
}

func (q *queue) dequeue(id string) {
	for i, rm := range q.values {
		if rm.id == id {
			q.values = q.values[i+1:]
			break
		}
	}
}

func (q *queue) isEmpty() bool {
	return len(q.values) == 0
}

func (q *queue) len() int {
	return len(q.values)
}

type sender struct {
	c       *client.Client
	srcAddr types.BtpAddress
	dstCfg  chain.BaseConfig
	w       types.Wallet
	l       log.Logger
	opt     struct {
		StepLimit int64
	}
	rr                 chan *types.RelayResult
	isFoundOffsetBySeq bool
	queue              *queue
}

func NewSender(srcAddr types.BtpAddress, dstCfg link.ChainConfig, w types.Wallet, endpoint string, opt map[string]interface{}, l log.Logger) types.Sender {
	s := &sender{
		srcAddr: srcAddr,
		dstCfg:  dstCfg.(chain.BaseConfig),
		w:       w,
		l:       l,
		rr:      make(chan *types.RelayResult),
		queue:   newQueue(),
	}
	b, err := json.Marshal(opt)
	if err != nil {
		l.Panicf("fail to marshal opt:%#v err:%+v", opt, err)
	}
	if err = json.Unmarshal(b, &s.opt); err != nil {
		l.Panicf("fail to unmarshal opt:%#v err:%+v", opt, err)
	}
	if s.opt.StepLimit <= 0 {
		s.opt.StepLimit = DefaultStepLimit
	}
	s.c = client.NewClient(endpoint, l)
	return s
}

func (s *sender) Start() (<-chan *types.RelayResult, error) {
	return s.rr, nil
}

func (s *sender) Stop() {
	close(s.rr)
}

func (s *sender) Relay(rm types.RelayMessage) (string, error) {
	//check send queue
	if MaxQueueSize <= s.queue.len() {
		return "", errors.InvalidStateError.New("pending queue full")
	}
	s.l.Debugf("_relay src address:%s, rm id:%s, rm msg:%s", s.srcAddr.String(), rm.Id(), hex.EncodeToString(rm.Bytes()[:]))

	thp, err := s._relay(rm)
	if err != nil {
		return "", err
	}

	b, err := thp.Hash.Value()
	if err != nil {
		return "", err
	}

	s.queue.enqueue(rm.Id(), b)

	go s.result(rm.Id(), thp)
	return rm.Id(), nil
}

func (s *sender) _relay(rm types.RelayMessage) (*client.TransactionHashParam, error) {
	msg := rm.Bytes()
	idx := len(msg) / txSizeLimit

	if idx == 0 {
		rmp := &client.BMCRelayMethodParams{
			Prev:     s.srcAddr.String(),
			Messages: base64.URLEncoding.EncodeToString(msg),
		}
		return s.sendTransaction(s.newTransactionParam(client.BMCRelayMethod, rmp))
	} else {
		thp, err := s.sendFragment(msg[:txSizeLimit], idx*-1)
		if err != nil {
			return nil, err
		}
		msg = msg[txSizeLimit:]
		for idx--; idx > 0; idx-- {
			if thp, err = s.sendFragment(msg[:txSizeLimit], idx); err != nil {
				return thp, err
			}
			msg = msg[txSizeLimit:]
		}
		if thp, err = s.sendFragment(msg[:], idx); err != nil {
			return nil, err
		}
		return thp, err
	}
}

func (s *sender) result(id string, txh *client.TransactionHashParam) {
	_, err := s.GetResult(txh)
	s.queue.dequeue(id)

	if err != nil {
		s.l.Debugf("result fail rm id : %s , txHash : %v", id, txh.Hash)

		if ec, ok := errors.CoderOf(err); ok {
			s.rr <- &types.RelayResult{
				Id:        id,
				Err:       ec.ErrorCode(),
				Finalized: true,
			}
		}
	} else {
		s.l.Debugf("result success rm id : %s , txHash : %v", id, txh.Hash)
		s.rr <- &types.RelayResult{
			Id:        id,
			Err:       -1,
			Finalized: true,
		}
	}
}

func (s *sender) GetPreference() types.Preference {
	p := types.Preference{
		TxSizeLimit:       int64(txSizeLimit),
		MarginForLimit:    int64(0),
		LatestResult:      false,
		FilledBlockUpdate: false,
	}

	return p
}

func (s *sender) GetStatus() (*types.BMCLinkStatus, error) {
	p := &client.CallParam{
		FromAddress: client.Address(s.w.Address()),
		ToAddress:   client.Address(s.dstCfg.Address.Account()),
		DataType:    "call",
		Data: client.CallData{
			Method: client.BMCGetStatusMethod,
			Params: client.BMCStatusParams{
				Target: s.srcAddr.String(),
			},
		},
	}
	bs := &client.BMCStatus{}
	err := client.MapError(s.c.Call(p, bs))
	if err != nil {
		return nil, err
	}
	ls := &types.BMCLinkStatus{}
	if ls.TxSeq, err = bs.TxSeq.Value(); err != nil {
		return nil, err
	}
	if ls.RxSeq, err = bs.RxSeq.Value(); err != nil {
		return nil, err
	}
	if ls.Verifier.Height, err = bs.Verifier.Height.Value(); err != nil {
		return nil, err
	}
	if ls.Verifier.Extra, err = bs.Verifier.Extra.Value(); err != nil {
		return nil, err
	}
	return ls, nil
}

func (s *sender) newTransactionParam(method string, params interface{}) *client.TransactionParam {
	p := &client.TransactionParam{
		Version:     client.NewHexInt(client.JsonrpcApiVersion),
		FromAddress: client.Address(s.w.Address()),
		ToAddress:   client.Address(s.dstCfg.Address.Account()),
		NetworkID:   client.HexInt(s.dstCfg.Address.NetworkID()),
		StepLimit:   client.NewHexInt(s.opt.StepLimit),
		DataType:    "call",
		Data: &client.CallData{
			Method: method,
			Params: params,
		},
	}
	return p
}

func (s *sender) sendFragment(msg []byte, idx int) (*client.TransactionHashParam, error) {
	fmp := &client.BMCFragmentMethodParams{
		Prev:     s.srcAddr.String(),
		Messages: base64.URLEncoding.EncodeToString(msg),
		Index:    client.NewHexInt(int64(idx)),
	}
	p := s.newTransactionParam(client.BMCFragmentMethod, fmp)
	return s.sendTransaction(p)
}

func (s *sender) sendTransaction(p *client.TransactionParam) (*client.TransactionHashParam, error) {
	thp := &client.TransactionHashParam{}
SignLoop:
	for {
		if err := s.c.SignTransaction(s.w, p); err != nil {
			return nil, err
		}
	SendLoop:
		for {
			txh, err := s.c.SendTransaction(p)
			if txh != nil {
				thp.Hash = *txh
			}
			if err != nil {
				if je, ok := err.(*jsonrpc.Error); ok {
					switch je.Code {
					case client.JsonrpcErrorCodeTxPoolOverflow:
						<-time.After(DefaultRelayReSendInterval)
						continue SendLoop
					case client.JsonrpcErrorCodeSystem:
						if subEc, err := strconv.ParseInt(je.Message[1:5], 0, 32); err == nil {
							switch subEc {
							case client.DuplicateTransactionError:
								s.l.Debugf("DuplicateTransactionError txh:%v", txh)
								return thp, nil
							case client.ExpiredTransactionError:
								continue SignLoop
							}
						}
					}
				}
				return nil, client.MapError(err)
			}
			return thp, nil
		}
	}
}

func (s *sender) GetResult(txh *client.TransactionHashParam) (*client.TransactionResult, error) {
	var retry = TransactionResultRetryLimit
	for {
		txr, err := s.c.GetTransactionResult(txh)
		if err != nil {
			if je, ok := err.(*jsonrpc.Error); ok {
				switch je.Code {
				case client.JsonrpcErrorCodePending, client.JsonrpcErrorCodeExecuting:
					<-time.After(DefaultGetRelayResultInterval)
					continue
				case client.JsonrpcErrorCodeNotFound:
					if retry == 0 {
						return nil, fmt.Errorf("not found transaction result ( TxHash : %v)", txh.Hash)
					}
					retry--
					<-time.After(DefaultGetRelayResultInterval)
					continue
				}
			}
		}
		return txr, mapErrorWithTransactionResult(txr, err)
	}
}

func mapErrorWithTransactionResult(txr *client.TransactionResult, err error) error {
	err = client.MapError(err)
	if err == nil && txr != nil && txr.Status != client.ResultStatusSuccess {
		fc, _ := txr.Failure.CodeValue.Value()
		if fc < client.ResultStatusFailureCodeRevert || fc > client.ResultStatusFailureCodeEnd {
			err = fmt.Errorf("failure with code:%s, message:%s",
				txr.Failure.CodeValue, txr.Failure.MessageValue)
		} else {
			err = errors.NewRevertError(int(fc - client.ResultStatusFailureCodeRevert))
		}
	}
	return err
}
