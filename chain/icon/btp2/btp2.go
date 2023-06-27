package btp2

import (
	"encoding/base64"
	"fmt"
	"time"

	"github.com/gorilla/websocket"

	"github.com/icon-project/btp2/chain/icon/client"
	"github.com/icon-project/btp2/common/codec"
	"github.com/icon-project/btp2/common/errors"
	"github.com/icon-project/btp2/common/intconv"
	"github.com/icon-project/btp2/common/jsonrpc"
	"github.com/icon-project/btp2/common/link"
	"github.com/icon-project/btp2/common/log"
	"github.com/icon-project/btp2/common/mbt"
	"github.com/icon-project/btp2/common/types"
)

const (
	DefaultGetBtpMessageInterval = time.Second
)

type receiveStatus struct {
	height int64
	seq    int64
}

func (r *receiveStatus) Height() int64 {
	return r.height
}

func (r *receiveStatus) Seq() int64 {
	return r.seq
}

func newReceiveStatus(height, seq int64) (*receiveStatus, error) {
	return &receiveStatus{
		height: height,
		seq:    seq,
	}, nil

}

type btp2 struct {
	l           log.Logger
	src         types.BtpAddress
	dst         types.BtpAddress
	c           *client.Client
	nid         int64
	rsc         chan interface{}
	rss         []*receiveStatus
	rs          *receiveStatus
	seq         int64
	startHeight int64
}

func NewBTP2(src, dst types.BtpAddress, endpoint string, l log.Logger) *btp2 {
	c := &btp2{
		src: src,
		dst: dst,
		l:   l,
		rsc: make(chan interface{}),
		rss: make([]*receiveStatus, 0),
		rs:  &receiveStatus{},
	}
	c.c = client.NewClient(endpoint, l)
	return c
}

func (b *btp2) getNetworkId() error {
	if b.nid == 0 {
		nid, err := b.c.GetBTPLinkNetworkId(b.src, b.dst)
		if err != nil {
			return err
		}
		b.nid = nid
	}

	return nil
}
func (b *btp2) getBtpMessage(height int64) ([]string, error) {
	for {
		pr := &client.BTPBlockParam{Height: client.HexInt(intconv.FormatInt(height)), NetworkId: client.HexInt(intconv.FormatInt(b.nid))}
		mgs, err := b.c.GetBTPMessage(pr)
		if err != nil {
			if je, ok := err.(*jsonrpc.Error); ok {
				switch je.Code {
				case client.JsonrpcErrorCodeNotFound:
					<-time.After(DefaultGetBtpMessageInterval)
					continue
				default:
					return nil, err
				}
			}
		}
		return mgs, nil
	}
}

func (b *btp2) getBtpHeader(height int64) ([]byte, []byte, error) {
	pr := &client.BTPBlockParam{Height: client.HexInt(intconv.FormatInt(height)), NetworkId: client.HexInt(intconv.FormatInt(b.nid))}
	hB64, err := b.c.GetBTPHeader(pr)
	if err != nil {
		return nil, nil, err
	}

	h, err := base64.StdEncoding.DecodeString(hB64)
	if err != nil {
		return nil, nil, err
	}

	pB64, err := b.c.GetBTPProof(pr)
	if err != nil {
		return nil, nil, err
	}
	p, err := base64.StdEncoding.DecodeString(pB64)
	if err != nil {
		return nil, nil, err
	}

	return h, p, nil
}

func (b *btp2) Start(bls *types.BMCLinkStatus) (<-chan interface{}, error) {
	if err := b.getNetworkId(); err != nil {
		return nil, err
	}

	if err := b.setStartHeight(); err != nil {
		return nil, err
	}

	go func() {
		err := b.Monitoring(bls)
		b.l.Debugf("Unknown monitoring error occurred  (err : %v)", err)
		b.rsc <- err
	}()

	return b.rsc, nil
}

func (b *btp2) Stop() {
	close(b.rsc)
}

func (b *btp2) GetStatus() (link.ReceiveStatus, error) {
	return b.rss[len(b.rss)-1], nil
}

func (b *btp2) GetHeightForSeq(seq int64) int64 {
	rs := b.GetReceiveStatusForSequence(seq)
	if rs != nil {
		return rs.height
	} else {
		return 0
	}
}

func (b *btp2) BuildBlockUpdate(bls *types.BMCLinkStatus, limit int64) ([]link.BlockUpdate, error) {
	bus := make([]link.BlockUpdate, 0)
	rs := b.nextReceiveStatus(bls)
	if rs == nil {
		return nil, errors.IllegalArgumentError.New("No blockUpdate available to create.")
	}

	h, p, err := b.getBtpHeader(rs.Height())
	if err != nil {
		return nil, err
	}
	bh := &client.BTPBlockHeader{}
	if _, err := codec.RLP.UnmarshalFromBytes(h, bh); err != nil {
		return nil, err
	}
	bbu := &client.BTPBlockUpdate{BTPBlockHeader: h, BTPBlockProof: p}

	if limit < int64(len(codec.RLP.MustMarshalToBytes(bbu))) {
		return bus, nil
	}

	bu := NewBlockUpdate(bls, bh.MainHeight, bbu)
	bus = append(bus, bu)
	return bus, nil
}

func (b *btp2) BuildBlockProof(bls *types.BMCLinkStatus, height int64) (link.BlockProof, error) {
	return nil, nil
}

func (b *btp2) BuildMessageProof(bls *types.BMCLinkStatus, limit int64) (link.MessageProof, error) {
	rs := b.GetReceiveStatusForHeight(bls.Verifier.Height)

	if rs == nil {
		return nil, nil
	}

	mbt, err := b.getMessage(bls.Verifier.Height)
	if err != nil {
		return nil, err
	}

	messageCnt := int64(mbt.Len())
	offset := bls.RxSeq - (rs.Seq() - messageCnt)
	if (bls.RxSeq - rs.seq) == 0 {
		return nil, nil
	}
	if messageCnt > 0 {
		for i := offset + 1; i < messageCnt; i++ {
			p, err := mbt.Proof(int(offset+1), int(i))
			if err != nil {
				return nil, err
			}

			if limit < int64(len(codec.RLP.MustMarshalToBytes(p))) {
				mp := NewMessageProof(bls, bls.RxSeq+i, *p)
				return mp, nil
			}
		}
	}

	p, err := mbt.Proof(int(offset+1), int(messageCnt))
	if err != nil {
		return nil, err
	}
	mp := NewMessageProof(bls, bls.RxSeq+messageCnt, *p)
	return mp, nil
}

func (b *btp2) BuildRelayMessage(rmis []link.RelayMessageItem) ([]byte, error) {
	bm := &BTPRelayMessage{
		Messages: make([]*TypePrefixedMessage, 0),
	}

	for _, rmi := range rmis {
		tpm, err := NewTypePrefixedMessage(rmi)
		if err != nil {
			return nil, err
		}

		bm.Messages = append(bm.Messages, tpm)
	}

	rb, err := codec.RLP.MarshalToBytes(bm)
	if err != nil {
		return nil, err
	}

	return rb, nil
}

func (b *btp2) FinalizedStatus(blsc <-chan *types.BMCLinkStatus) {
	go func() {
		for {
			select {
			case bls := <-blsc:
				b.clearReceiveStatus(bls)
			}
		}
	}()
}

func (b *btp2) nextReceiveStatus(bls *types.BMCLinkStatus) *receiveStatus {
	for i, rs := range b.rss {
		if bls.Verifier.Height <= rs.Height() {
			if bls.Verifier.Height == rs.Height() {
				return b.rss[i+1]
			}
			return b.rss[i]
		}
	}
	return nil
}

func (b *btp2) clearReceiveStatus(bls *types.BMCLinkStatus) {
	for i, rs := range b.rss {
		if rs.Height() <= bls.Verifier.Height && rs.Seq() <= bls.RxSeq {
			b.l.Debugf("clear receive data (height:%d, seq:%d) ", bls.Verifier.Height, bls.RxSeq)
			b.rss = b.rss[i+1:]
			return
		}
	}
}

func (b *btp2) getMessage(height int64) (*mbt.MerkleBinaryTree, error) {
	msgs, err := b.getBtpMessage(height)
	if err != nil {
		return nil, err
	}
	result := make([][]byte, 0)
	for _, mg := range msgs {
		m, err := base64.StdEncoding.DecodeString(mg)
		if err != nil {
			return nil, err
		}
		result = append(result, m)
	}

	mt, err := mbt.NewMerkleBinaryTree(mbt.HashFuncByUID("eth"), result)
	if err != nil {
		return nil, err
	}
	return mt, nil
}

func (b *btp2) Monitoring(bls *types.BMCLinkStatus) error {
	if bls.Verifier.Height < 1 {
		return fmt.Errorf("cannot catchup from zero height")
	}

	req := &client.BTPRequest{
		Height:    client.NewHexInt(bls.Verifier.Height + 1),
		NetworkID: client.NewHexInt(b.nid),
		ProofFlag: client.NewHexInt(0),
	}

	onErr := func(conn *websocket.Conn, err error) {
		b.l.Debugf("onError %s err:%+v", conn.LocalAddr().String(), err)
		b.c.CloseMonitor(conn)
		//Restart Monitoring
		ls := &types.BMCLinkStatus{}
		ls.TxSeq = b.rs.Seq()
		ls.Verifier.Height = b.rs.Height()
		b.l.Debugf("Restart Monitoring")
		b.Monitoring(ls)
	}
	onConn := func(conn *websocket.Conn) {
		b.l.Debugf("ReceiveLoop monitorBTP2Block height:%d seq:%d networkId:%d connected %s",
			bls.Verifier.Height, bls.TxSeq, b.nid, conn.LocalAddr().String())
	}

	err := b.monitorBTP2Block(req, bls, onConn, onErr)
	if err != nil {
		return err
	}
	return nil
}

func (b *btp2) monitorBTP2Block(req *client.BTPRequest, bls *types.BMCLinkStatus, scb func(conn *websocket.Conn), errCb func(*websocket.Conn, error)) error {
	if bls.RxSeq != 0 {
		b.seq = bls.RxSeq
	}

	if b.rs.Height() == 0 {
		b.rs.height = bls.Verifier.Height
		b.rs.seq = bls.RxSeq
	}

	return b.c.MonitorBTP(req, func(conn *websocket.Conn, v *client.BTPNotification) error {
		h, err := base64.StdEncoding.DecodeString(v.Header)
		if err != nil {
			return err
		}

		bh := &client.BTPBlockHeader{}
		if _, err = codec.RLP.UnmarshalFromBytes(h, bh); err != nil {
			return err
		}

		if bh.MainHeight != b.startHeight {
			msgs, err := b.getBtpMessage(bh.MainHeight)
			if err != nil {
				return err
			}

			b.seq += int64(len(msgs))
			rs, err := newReceiveStatus(bh.MainHeight, b.seq)
			if err != nil {
				return err
			}
			b.rs = rs
			b.rss = append(b.rss, rs)
			b.l.Debugf("monitor info : Height:%d  UpdateNumber:%d  MessageCnt:%d  Seq:%d ", bh.MainHeight, bh.UpdateNumber, len(msgs), b.seq)
			b.rsc <- rs
		}
		return nil
	}, scb, errCb)
}

func (b *btp2) GetReceiveStatusForSequence(seq int64) *receiveStatus {
	for _, rs := range b.rss {
		if rs.Seq() <= seq && seq <= rs.Seq() { //TODO
			return rs
		}
	}
	return nil
}

func (b *btp2) GetReceiveStatusForHeight(height int64) *receiveStatus {
	for _, rs := range b.rss {
		if rs.Height() == height {
			return rs
		}
	}
	return nil
}

func (b *btp2) setStartHeight() error {
	p := &client.BTPNetworkInfoParam{Id: client.HexInt(intconv.FormatInt(b.nid))}
	ni, err := b.c.GetBTPNetworkInfo(p)
	if err != nil {
		return err
	}
	sh, err := ni.StartHeight.Value()
	b.startHeight = sh + 1
	return nil
}
