package bridge

import (
	"encoding/base64"
	"fmt"
	"unsafe"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gorilla/websocket"

	"github.com/icon-project/btp2/chain/icon/client"
	"github.com/icon-project/btp2/common/codec"
	"github.com/icon-project/btp2/common/intconv"
	"github.com/icon-project/btp2/common/link"
	"github.com/icon-project/btp2/common/log"
	"github.com/icon-project/btp2/common/types"
)

type receiveStatus struct {
	height int64
	seq    int64
	rp     *ReceiptProof
}

func (r *receiveStatus) Height() int64 {
	return r.height
}

func (r *receiveStatus) Seq() int64 {
	return r.seq
}

func (r *receiveStatus) ReceiptProof() *ReceiptProof {
	return r.rp
}

type bridge struct {
	l           log.Logger
	src         link.ChainConfig
	dst         types.BtpAddress
	c           *client.Client
	nid         int64
	rsc         chan interface{}
	rss         []*receiveStatus
	rs          *receiveStatus
	startHeight int64
}

func newReceiveStatus(height, rxSeq int64, sn int64, msgs []string, next types.BtpAddress) (*receiveStatus, error) {
	evts := make([]*Event, 0)
	seq := sn
	for _, msg := range msgs {
		if sn > rxSeq {
			evt, err := messageToEvent(next, msg, sn)
			if err != nil {
				return nil, err
			}
			evts = append(evts, evt)
		}
		sn++
	}

	rp := &ReceiptProof{
		Index:  0,
		Events: evts,
		Height: height,
	}

	return &receiveStatus{
		height: height,
		seq:    seq,
		rp:     rp,
	}, nil
}

func newBridge(src link.ChainConfig, dst types.BtpAddress, endpoint string, baseDir string, l log.Logger) (*bridge, error) {
	c := &bridge{
		src: src,
		dst: dst,
		l:   l,
		rsc: make(chan interface{}),
		rss: make([]*receiveStatus, 0),
		rs:  &receiveStatus{},
	}
	c.c = client.NewClient(endpoint, l)
	return c, nil
}

func (b *bridge) getNetworkId() error {
	if b.nid == 0 {
		nid, err := b.c.GetBTPLinkNetworkId(b.src.GetAddress(), b.dst)
		if err != nil {
			return err
		}
		b.nid = nid
	}

	return nil
}

func (b *bridge) Start(bls *types.BMCLinkStatus) (<-chan interface{}, error) {
	if err := b.getNetworkId(); err != nil {
		return nil, err
	}

	if err := b.setStartHeight(); err != nil {
		return nil, err
	}

	go func() {
		err := b.monitoring(bls)
		b.l.Debugf("Unknown monitoring error occurred  (err : %v)", err)
		b.rsc <- err
	}()

	return b.rsc, nil
}

func (b *bridge) Stop() {
	close(b.rsc)
}

func (b *bridge) GetStatus() (link.ReceiveStatus, error) {
	return b.rss[len(b.rss)-1], nil
}

func (b *bridge) GetHeightForSeq(seq int64) int64 {
	rs := b.getReceiveStatusForSequence(seq)
	if rs != nil {
		return rs.height
	} else {
		return 0
	}
}

func (b *bridge) BuildBlockUpdate(bls *types.BMCLinkStatus, limit int64) ([]link.BlockUpdate, error) {
	b.l.Debugf("Build BlockUpdate (height=%d, rxSeq=%d)", bls.Verifier.Height, bls.RxSeq)
	bus := make([]link.BlockUpdate, 0)
	rs := b.nextReceiveStatus(bls)
	bu := newBlockUpdate(bls, rs.Height())
	bus = append(bus, bu)
	return bus, nil
}

func (b *bridge) BuildBlockProof(bls *types.BMCLinkStatus, height int64) (link.BlockProof, error) {
	return nil, nil
}

func (b *bridge) BuildMessageProof(bls *types.BMCLinkStatus, limit int64) (link.MessageProof, error) {
	b.l.Debugf("Build BuildMessageProof (height=%d, rxSeq=%d)", bls.Verifier.Height, bls.RxSeq)
	var rmSize int
	rs := b.getReceiveStatusForSequence(bls.RxSeq + 1)
	if rs == nil {
		return nil, nil
	}
	//offset := int64(rs.seq - bls.RxSeq)
	messageCnt := len(rs.ReceiptProof().Events)
	offset := bls.RxSeq - (rs.Seq() - int64(messageCnt))
	trp := &ReceiptProof{
		Index:  rs.ReceiptProof().Index,
		Events: make([]*Event, 0),
		Height: rs.Height(),
	}

	for i := offset; i < int64(messageCnt); i++ {
		size := sizeOfEvent(rs.ReceiptProof().Events[i])
		if limit < int64(rmSize+size) {
			return newMessageProof(bls, bls.RxSeq+i, trp)
		}
		trp.Events = append(trp.Events, rs.ReceiptProof().Events[i])
		rmSize += size
	}

	//last event
	return newMessageProof(bls, bls.RxSeq+int64(messageCnt), trp)

}

func (b *bridge) BuildRelayMessage(rmis []link.RelayMessageItem) ([]byte, error) {
	//delete blockUpdate and only mp append
	for _, rmi := range rmis {
		if rmi.Type() == link.TypeMessageProof {
			mp := rmi.(*MessageProof)
			b.l.Debugf("BuildRelayMessage (height:%d, data:%s ", mp.nextBls.Verifier.Height,
				base64.URLEncoding.EncodeToString(mp.Bytes()))

			return mp.Bytes(), nil
		}
	}
	return nil, nil
}

func (b *bridge) FinalizedStatus(blsc <-chan *types.BMCLinkStatus) {
	go func() {
		for {
			select {
			case bls := <-blsc:
				b.clearReceiveStatus(bls)
			}
		}
	}()
}

func (b *bridge) monitoring(bls *types.BMCLinkStatus) error {
	if bls.Verifier.Height < 1 {
		return fmt.Errorf("cannot catchup from zero height")
	}

	req := &client.BTPRequest{
		Height:    client.NewHexInt(bls.Verifier.Height),
		NetworkID: client.NewHexInt(b.nid),
		ProofFlag: client.NewHexInt(0),
	}

	onErr := func(conn *websocket.Conn, err error) {
		b.l.Debugf("onError %s err:%+v", conn.LocalAddr().String(), err)
		b.c.CloseMonitor(conn)
		//Restart Monitoring
		ls := &types.BMCLinkStatus{}
		ls.RxSeq = b.rs.Seq()
		ls.Verifier.Height = b.rs.Height()
		b.l.Debugf("Restart Monitoring")
		b.monitoring(ls)
	}
	onConn := func(conn *websocket.Conn) {
		b.l.Debugf("ReceiveLoop monitorBTP2Block height:%d seq:%d networkId:%d connected %s",
			bls.Verifier.Height, bls.RxSeq, b.nid, conn.LocalAddr().String())
	}

	err := b.monitorBTP2Block(req, bls, onConn, onErr)
	if err != nil {
		return err
	}
	return nil
}

func (b *bridge) monitorBTP2Block(req *client.BTPRequest, bls *types.BMCLinkStatus, scb func(conn *websocket.Conn), errCb func(*websocket.Conn, error)) error {
	offset, err := b.c.GetBTPLinkOffset(b.src.GetAddress(), b.dst)
	if err != nil {
		return err
	}
	//BMC.seq starts with 1 and BTPBlock.FirstMessageSN starts with 0
	offset += 1

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
			msgs, err := b.c.GetBTPMessage(bh.MainHeight, b.nid)
			if err != nil {
				return err
			}
			sn := offset + bh.UpdateNumber>>1

			rs, err := newReceiveStatus(bh.MainHeight, bls.RxSeq, sn, msgs, b.dst)
			if err != nil {
				return err
			}
			b.rs = rs
			b.rss = append(b.rss, rs)
			b.l.Debugf("monitor info : Height:%d  UpdateNumber:%d  MessageCnt:%d ", bh.MainHeight, bh.UpdateNumber, len(msgs))

			b.rsc <- rs
		}
		return nil
	}, scb, errCb)
}

func (b *bridge) nextReceiveStatus(bls *types.BMCLinkStatus) *receiveStatus {
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

func (b *bridge) clearReceiveStatus(bls *types.BMCLinkStatus) {
	for i, rs := range b.rss {
		if rs.Height() <= bls.Verifier.Height && rs.Seq() <= bls.RxSeq {
			b.l.Debugf("clear receive data (height:%d, seq:%d) ", bls.Verifier.Height, bls.RxSeq)
			b.rss = b.rss[i+1:]
			return
		}
	}
}

func (b *bridge) getReceiveStatusForSequence(seq int64) *receiveStatus {
	for _, rs := range b.rss {
		if rs.Seq() <= seq && seq <= rs.Seq() {
			return rs
		}
	}
	return nil
}

func (b *bridge) setStartHeight() error {
	p := &client.BTPNetworkInfoParam{Id: client.HexInt(intconv.FormatInt(b.nid))}
	ni, err := b.c.GetBTPNetworkInfo(p)
	if err != nil {
		return err
	}
	sh, err := ni.StartHeight.Value()
	b.startHeight = sh + 1
	return nil
}

func sizeOfEvent(rp *Event) int {
	return int(unsafe.Sizeof(rp))
}

func messageToEvent(next types.BtpAddress, msg string, seq int64) (*Event, error) {
	b, err := base64.StdEncoding.DecodeString(msg)
	if err != nil {
		return nil, err
	}
	evt := &Event{
		Next:     crypto.Keccak256Hash([]byte(next.String())).Bytes(),
		Sequence: seq,
		Message:  b,
	}
	return evt, nil
}
