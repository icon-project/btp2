package ethbr

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"sort"
	"unsafe"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/icon-project/btp2/chain/ethbr/binding"
	"github.com/icon-project/btp2/chain/ethbr/client"
	"github.com/icon-project/btp2/common/codec"
	"github.com/icon-project/btp2/common/errors"
	"github.com/icon-project/btp2/common/link"
	"github.com/icon-project/btp2/common/log"
	btpTypes "github.com/icon-project/btp2/common/types"
)

type receiveStatus struct {
	height   int64
	startSeq int64
	lastSeq  int64
	rps      []*client.ReceiptProof
}

func (r *receiveStatus) Height() int64 {
	return r.height
}

func (r *receiveStatus) Seq() int64 {
	return r.lastSeq
}

func (r *receiveStatus) StartSeq() int64 {
	return r.startSeq
}

func (r *receiveStatus) LastSeq() int64 {
	return r.lastSeq
}

func newReceiveStatus(height, startSeq, lastSeq int64, rps []*client.ReceiptProof) (*receiveStatus, error) {
	return &receiveStatus{
		height:   height,
		startSeq: startSeq,
		lastSeq:  lastSeq,
		rps:      rps,
	}, nil
}

const (
	EPOCH               = 200
	EventSignature      = "Message(string,uint256,bytes)"
	EventIndexSignature = 0
	EventIndexNext      = 1
	EventIndexSequence  = 2
)

type ethbr struct {
	l           log.Logger
	src         btpTypes.BtpAddress
	dst         btpTypes.BtpAddress
	c           *client.Client
	nid         int64
	rsc         chan interface{}
	rss         []*receiveStatus
	seq         int64
	startHeight int64
	opt         struct {
		StartHeight int64
	}
}

func newEthBridge(src, dst btpTypes.BtpAddress, endpoint string, l log.Logger, opt map[string]interface{}) *ethbr {
	c := &ethbr{
		src: src,
		dst: dst,
		l:   l,
		rsc: make(chan interface{}),
		rss: make([]*receiveStatus, 0),
	}
	c.c = client.NewClient(endpoint, l)
	b, err := json.Marshal(opt)
	if err != nil {
		l.Panicf("fail to marshal opt:%#v err:%+v", opt, err)
	}

	if err = json.Unmarshal(b, &c.opt); err != nil {
		l.Panicf("fail to unmarshal opt:%#v err:%+v", opt, err)
	}
	return c
}

func (e *ethbr) Start(bls *btpTypes.BMCLinkStatus) (<-chan interface{}, error) {
	go func() {
		err := e.Monitoring(bls)
		e.l.Debugf("Unknown monitoring error occurred  (err : %v)", err)
		e.rsc <- err
	}()

	return e.rsc, nil
}

func (e *ethbr) Stop() {
	close(e.rsc)
}

func (e *ethbr) GetStatus() (link.ReceiveStatus, error) {
	return e.rss[len(e.rss)-1], nil
}

func (e *ethbr) BuildBlockUpdate(bls *btpTypes.BMCLinkStatus, limit int64) ([]link.BlockUpdate, error) {
	bus := make([]link.BlockUpdate, 0)
	rs := e.nextReceiveStatus(bls)
	if rs == nil {
		return nil, errors.IllegalArgumentError.New("No blockUpdate available to create.")
	}

	bu := NewBlockUpdate(bls, rs.Height())
	bus = append(bus, bu)
	return bus, nil
}

func (e *ethbr) BuildBlockProof(bls *btpTypes.BMCLinkStatus, height int64) (link.BlockProof, error) {
	return nil, nil
}

func (e *ethbr) BuildMessageProof(bls *btpTypes.BMCLinkStatus, limit int64) (link.MessageProof, error) {
	var rmSize int
	seq := bls.RxSeq + 1
	rps := make([]*client.ReceiptProof, 0)
	rs := e.GetReceiveStatusForSequence(seq)
	if rs == nil {
		return nil, nil
	}

	eventCnt := rs.lastSeq - (rs.startSeq - 1)
	e.l.Debugf("OnBlockOfSrc eventCnt:%d rxSeq:%d", eventCnt, rs.Seq())
	if eventCnt > 0 {
		for _, rp := range rs.rps {
			trp := &client.ReceiptProof{
				Index:  rp.Index,
				Events: make([]*client.Event, 0),
				Height: rp.Height,
			}
			for _, event := range rp.Events {
				if event.Sequence.Int64() == seq {
					rps = append(rps, trp)
					size := sizeOfEvent(event)

					if (int(limit) < rmSize+size) && rmSize > 0 {
						return NewMessageProof(bls, bls.RxSeq+1, seq-1, rps)
					}

					trp.Events = append(trp.Events, event)
					seq = event.Sequence.Int64()
					rmSize += size

					seq = seq + 1
				}
			}

			//last event
			if int(limit) < rmSize {
				return NewMessageProof(bls, bls.RxSeq+1, seq-1, rps)
			}

			//remove last receipt if empty
			if len(trp.Events) == 0 {
				rps = rps[:len(rps)-1]
			}
		}
		return NewMessageProof(bls, bls.RxSeq+1, seq-1, rps)
	}
	return nil, nil
}

func (e *ethbr) GetHeightForSeq(seq int64) int64 {
	rs := e.GetReceiveStatusForSequence(seq)
	if rs != nil {
		return rs.height
	} else {
		return 0
	}
}

func (e *ethbr) BuildRelayMessage(rmis []link.RelayMessageItem) ([]byte, error) {
	//delete blockUpdate and only mp append
	for _, rmi := range rmis {
		if rmi.Type() == link.TypeMessageProof {
			mp := rmi.(*MessageProof)
			e.l.Debugf("BuildRelayMessage height:%d data:%s ", mp.nextBls.Verifier.Height,
				base64.URLEncoding.EncodeToString(mp.Bytes()))
			return mp.Bytes(), nil
		}
	}
	return nil, nil
}

func (e *ethbr) FinalizedStatus(blsc <-chan *btpTypes.BMCLinkStatus) {
	go func() {
		for {
			select {
			case bls := <-blsc:
				e.clearReceiveStatus(bls)
			}
		}
	}()
}

func (e *ethbr) nextReceiveStatus(bls *btpTypes.BMCLinkStatus) *receiveStatus {
	for i, rs := range e.rss {
		if bls.Verifier.Height <= rs.Height() {
			if bls.Verifier.Height == rs.Height() {
				return e.rss[i+1]
			}
			return e.rss[i]
		}
	}
	return nil
}

func (e *ethbr) clearReceiveStatus(bls *btpTypes.BMCLinkStatus) {
	for i, rs := range e.rss {
		if rs.Height() <= bls.Verifier.Height && rs.Seq() <= bls.RxSeq {
			e.l.Debugf("clear receive data (height:%d, seq:%d) ", bls.Verifier.Height, bls.RxSeq)
			e.rss = e.rss[i+1:]
			return
		}
	}
}

func (e *ethbr) Monitoring(bls *btpTypes.BMCLinkStatus) error {
	var height int64
	fq := &ethereum.FilterQuery{
		Addresses: []common.Address{common.HexToAddress(e.src.ContractAddress())},
		Topics: [][]common.Hash{
			{crypto.Keccak256Hash([]byte(EventSignature))},
		},
	}

	if e.opt.StartHeight > bls.Verifier.Height {
		height = e.opt.StartHeight
	} else {
		height = bls.Verifier.Height
	}

	e.l.Debugf("ReceiveLoop height:%d seq:%d filterQuery[Address:%s,Topic:%s]",
		height, bls.RxSeq, fq.Addresses[0].String(), fq.Topics[0][0].Hex())
	br := &client.BlockRequest{
		Height:      big.NewInt(height),
		FilterQuery: fq,
	}

	if bls.RxSeq != 0 {
		e.seq = bls.RxSeq
	}

	return e.c.MonitorBlock(br,
		func(v *client.BlockNotification) error {

			if len(v.Logs) > 0 {
				var startSeq int64
				var lastSeq int64
				rpsMap := make(map[uint]*client.ReceiptProof)
			EpLoop:
				for _, el := range v.Logs {
					evt, err := logToEvent(&el)
					if err != nil {
						return err
					}

					e.l.Debugf("event[seq:%d] seq:%d dst:%s",
						evt.Sequence, e.seq, e.dst.String())
					if evt.Sequence.Int64() <= e.seq {
						continue EpLoop
					}

					if startSeq == 0 {
						startSeq = evt.Sequence.Int64()
					}
					lastSeq = evt.Sequence.Int64()
					//below statement is unnecessary if 'next' is indexed
					dstHash := crypto.Keccak256Hash([]byte(e.dst.String()))
					if !bytes.Equal(evt.Next, dstHash.Bytes()) {
						continue EpLoop
					}

					rp, ok := rpsMap[el.TxIndex]
					if !ok {
						rp = &client.ReceiptProof{
							Index:  int64(el.TxIndex),
							Events: make([]*client.Event, 0),
							Height: int64(el.BlockNumber),
						}
						rpsMap[el.TxIndex] = rp
					}
					rp.Events = append(rp.Events, evt)
				}
				if len(rpsMap) > 0 {
					rps := make([]*client.ReceiptProof, 0)
					for _, rp := range rpsMap {
						rps = append(rps, rp)
					}
					sort.Slice(rps, func(i int, j int) bool {
						return rps[i].Index < rps[j].Index
					})
					e.seq = lastSeq
					rs, err := newReceiveStatus(v.Height.Int64(), startSeq, lastSeq, rps)
					if err != nil {
						return err
					}

					e.rss = append(e.rss, rs)
					e.l.Debugf("monitor info : Height:%d  RpsCnt:%d LastSeq:%d ",
						v.Height.Int64(), len(rps), e.seq)

					e.rsc <- rs
				}
			}

			return nil
		},
	)
}

func (e *ethbr) newBlockUpdate(v *client.BlockNotification) (*client.BlockUpdate, error) {
	var err error

	bu := &client.BlockUpdate{
		BlockHash: v.Hash.Bytes(),
		Height:    v.Height.Int64(),
	}

	header := client.MakeHeader(v.Header)
	bu.Header, err = codec.RLP.MarshalToBytes(*header)
	if err != nil {
		return nil, err
	}

	encodedHeader, _ := rlp.EncodeToBytes(v.Header)
	if !bytes.Equal(v.Header.Hash().Bytes(), crypto.Keccak256(encodedHeader)) {
		return nil, fmt.Errorf("mismatch block hash with BlockNotification")
	}

	update := &client.EvmBlockUpdate{}
	update.BlockHeader, _ = codec.RLP.MarshalToBytes(*header)
	buf := new(bytes.Buffer)
	encodeSigHeader(buf, v.Header)
	update.EvmHeader = buf.Bytes()

	bu.Proof, err = codec.RLP.MarshalToBytes(update)
	if err != nil {
		return nil, err
	}

	return bu, nil
}

func (e *ethbr) GetReceiveStatusForSequence(seq int64) *receiveStatus {
	for _, rs := range e.rss {
		if rs.startSeq <= seq && seq <= rs.lastSeq {
			return rs
		}
	}
	return nil
}

func (e *ethbr) GetReceiveStatusForHeight(height int64) *receiveStatus {
	for _, rs := range e.rss {
		if rs.Height() == height {
			return rs
		}
	}
	return nil
}

func encodeSigHeader(w io.Writer, header *types.Header) {
	err := rlp.Encode(w, []interface{}{
		big.NewInt(97),
		header.ParentHash,
		header.UncleHash,
		header.Coinbase,
		header.Root,
		header.TxHash,
		header.ReceiptHash,
		header.Bloom,
		header.Difficulty,
		header.Number,
		header.GasLimit,
		header.GasUsed,
		header.Time,
		header.Extra[:len(header.Extra)-65], // Yes, this will panic if extra is too short
		header.MixDigest,
		header.Nonce,
	})

	if err != nil {
		//panic("can't encode: " + err.Error())
	}
}

func logToEvent(el *types.Log) (*client.Event, error) {
	mgs, err := binding.UnpackEventLog(el.Data)
	if err != nil {
		return nil, err
	}
	return &client.Event{
		Next:     el.Topics[1].Bytes(),
		Sequence: el.Topics[2].Big(),
		Message:  mgs.Msg,
	}, nil
}

func sizeOfEvent(rp *client.Event) int {
	return int(unsafe.Sizeof(rp))
}
