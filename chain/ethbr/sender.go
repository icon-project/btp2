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

package ethbr

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/core/types"

	"github.com/icon-project/btp2/chain"
	"github.com/icon-project/btp2/common/link"
	btpTypes "github.com/icon-project/btp2/common/types"

	"github.com/icon-project/btp2/chain/ethbr/binding"
	"github.com/icon-project/btp2/chain/ethbr/client"
	"github.com/icon-project/btp2/common/errors"
	"github.com/icon-project/btp2/common/log"
	"github.com/icon-project/btp2/common/wallet"
)

const (
	txMaxDataSize                 = 524288 //512 * 1024 // 512kB
	txOverheadScale               = 0.37   //base64 encoding overhead 0.36, rlp and other fields 0.01
	DefaultGetRelayResultInterval = time.Second
	MaxQueueSize                  = 100
)

var (
	txSizeLimit = int(math.Ceil(txMaxDataSize / (1 + txOverheadScale))) //TODO xcall의 2k 사이즈에 맞게 다시 설정한다.
)

type Queue struct {
	values []*relayMessageTx
}

type relayMessageTx struct {
	id     string
	txHash []byte
}

func NewQueue() *Queue {
	queue := &Queue{}
	return queue
}

func (q *Queue) enqueue(id string, txHash []byte) error {
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

func (q *Queue) dequeue(id string) {
	for i, rm := range q.values {
		if rm.id == id {
			q.values = q.values[i+1:]
			break
		}
	}
}

func (q *Queue) isEmpty() bool {
	return len(q.values) == 0
}

func (q *Queue) len() int {
	return len(q.values)
}

type sender struct {
	c       *client.Client
	srcAddr btpTypes.BtpAddress
	dstCfg  chain.BaseConfig
	w       btpTypes.Wallet
	l       log.Logger
	opt     struct {
	}
	bmc                *binding.BMC
	rr                 chan *btpTypes.RelayResult
	isFoundOffsetBySeq bool
	queue              *Queue
}

func newSender(srcAddr btpTypes.BtpAddress, dstCfg link.ChainConfig, w btpTypes.Wallet, endpoint string, opt map[string]interface{}, l log.Logger) btpTypes.Sender {
	s := &sender{
		srcAddr: srcAddr,
		dstCfg:  dstCfg.(chain.BaseConfig),
		w:       w,
		l:       l,
		rr:      make(chan *btpTypes.RelayResult),
		queue:   NewQueue(),
	}

	b, err := json.Marshal(opt)
	if err != nil {
		l.Panicf("fail to marshal opt:%#v err:%+v", opt, err)
	}
	if err = json.Unmarshal(b, &s.opt); err != nil {
		l.Panicf("fail to unmarshal opt:%#v err:%+v", opt, err)
	}

	s.c = client.NewClient(endpoint, l)

	s.bmc, _ = binding.NewBMC(client.HexToAddress(s.dstCfg.Address.ContractAddress()), s.c.GetEthClient())

	return s
}

func (s *sender) Start() (<-chan *btpTypes.RelayResult, error) {
	return s.rr, nil
}

func (s *sender) Stop() {
	close(s.rr)
}
func (s *sender) GetStatus() (*btpTypes.BMCLinkStatus, error) {
	var status binding.TypesLinkStatus
	status, err := s.bmc.GetStatus(nil, s.srcAddr.String())
	if err != nil {
		s.l.Errorf("Error retrieving relay status from BMC")
		return nil, err
	}

	ls := &btpTypes.BMCLinkStatus{}
	ls.TxSeq = status.TxSeq.Int64()
	ls.RxSeq = status.RxSeq.Int64()
	ls.Verifier.Height = status.Verifier.Height.Int64()
	ls.Verifier.Extra = status.Verifier.Extra

	return ls, nil
}

func (s *sender) Relay(rm btpTypes.RelayMessage) (string, error) {
	//check send queue
	if MaxQueueSize <= s.queue.len() {
		return "", errors.InvalidStateError.New("pending queue full")
	}

	thp, err := s._relay(rm)
	if err != nil {
		return "", err
	}

	s.queue.enqueue(rm.Id(), thp.Hash.Bytes())
	go s.result(rm.Id(), thp)
	return rm.Id(), nil
}

func (s *sender) result(id string, txh *client.TransactionHashParam) {
	_, err := s.GetResult(txh)
	s.queue.dequeue(id)

	if err != nil {
		s.l.Debugf("result fail rm id : %d , txHash : %v", id, txh.Hash)

		if ec, ok := errors.CoderOf(err); ok {
			s.rr <- &btpTypes.RelayResult{
				Id:        id,
				Err:       ec.ErrorCode(),
				Finalized: true,
			}
		}
	} else {
		s.l.Debugf("result success rm id : %s , txHash : %v", id, txh.Hash)
		s.rr <- &btpTypes.RelayResult{
			Id:        id,
			Err:       -1,
			Finalized: true,
		}
	}

}

func (s *sender) GetResult(txh *client.TransactionHashParam) (*types.Receipt, error) {
	for {
		_, pending, err := s.c.GetTransaction(txh.Hash)
		if err != nil {
			return nil, err
		}
		if pending {
			<-time.After(DefaultGetRelayResultInterval)
			continue
		}
		tx, err := s.c.GetTransactionReceipt(txh.Hash)
		if err != nil {
			return nil, err
		}

		if tx.Status == 0 {
			revertMsg, err := s.c.GetRevertMessage(txh.Hash)
			if err != nil {
				return nil, err
			}
			msgs := strings.Split(revertMsg, ":")
			if len(msgs) > 2 {
				codeMsg := strings.Split(msgs[1], " ")
				code, err := strconv.Atoi(codeMsg[len(codeMsg)-1])
				if err != nil {
					return nil, err
				}
				return tx, client.NewRevertError(code)
			} else {
				return nil, client.NewRevertError(25)
			}

		}
		return tx, nil
	}
}

func (s *sender) GetPreference() btpTypes.Preference {
	p := btpTypes.Preference{
		TxSizeLimit:       int64(txSizeLimit),
		MarginForLimit:    int64(0),
		LatestResult:      false,
		FilledBlockUpdate: false,
	}

	return p
}

func (s *sender) _relay(rm btpTypes.RelayMessage) (*client.TransactionHashParam, error) {

	t, err := s.c.NewTransactOpts(s.w.(*wallet.EvmWallet).Skey)
	if err != nil {
		return nil, err
	}

	var tx *types.Transaction

	tx, err = s.bmc.HandleRelayMessage(t, s.srcAddr.String(), rm.Bytes()[:])
	if err != nil {
		s.l.Errorf("handleRelayMessage error: %s, rm id:%s ", err.Error(), rm.Id())
		return nil, err
	}
	txh := tx.Hash()
	return &client.TransactionHashParam{txh}, nil
}
