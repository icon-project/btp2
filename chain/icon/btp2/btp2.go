package btp2

import (
	"encoding/base64"
	"fmt"
	"math/big"
	"path/filepath"

	"github.com/gorilla/websocket"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"

	"github.com/icon-project/btp2/chain/icon/client"
	"github.com/icon-project/btp2/common/codec"
	"github.com/icon-project/btp2/common/errors"
	"github.com/icon-project/btp2/common/intconv"
	"github.com/icon-project/btp2/common/link"
	"github.com/icon-project/btp2/common/log"
	"github.com/icon-project/btp2/common/mbt"
	"github.com/icon-project/btp2/common/types"
)

const (
	DefaultProgressInterval = 50
	ReceiveBlockPrefix      = "H|"
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
	src         link.ChainConfig
	dst         types.BtpAddress
	c           *client.Client
	db          *leveldb.DB
	nid         int64
	rsc         chan interface{}
	rss         []*receiveStatus
	seq         int64
	startHeight int64
}

func newBTP2(src link.ChainConfig, dst types.BtpAddress, endpoint string, baseDir string, l log.Logger) (*btp2, error) {
	c := &btp2{
		src: src,
		dst: dst,
		l:   l,
		rsc: make(chan interface{}),
		rss: make([]*receiveStatus, 0),
	}
	c.c = client.NewClient(endpoint, l)
	err := c.prepareDatabase(baseDir)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (b *btp2) getFirstHeightForReceiveBlock() int64 {
	iter := b.db.NewIterator(util.BytesPrefix([]byte(ReceiveBlockPrefix)), nil)
	if iter.First() {
		return new(big.Int).SetBytes(iter.Key()[len([]byte(ReceiveBlockPrefix)):]).Int64()
	}
	return 0
}

func (b *btp2) addReceiveBlock(height int64, data []byte) error {
	h := big.NewInt(height)
	key := append([]byte(ReceiveBlockPrefix), h.Bytes()...)
	return b.db.Put(key, data, nil)
}

func (b *btp2) removeAllReceiveBlock() error {
	b.l.Debugf("removeAllReceiveBlock")
	iter := b.db.NewIterator(util.BytesPrefix([]byte(ReceiveBlockPrefix)), nil)
	if iter.Next() {
		b.l.Debugf("Delete height stored in database (height : %d)",
			new(big.Int).SetBytes(iter.Key()[len([]byte(ReceiveBlockPrefix)):]).Int64())
		if err := b.db.Delete(iter.Key(), nil); err != nil {
			return err
		}
	}
	return nil
}

func (b *btp2) removeReceiveBlockByHeight(height int64) error {
	b.l.Debugf("removeReceiveBlockByHeight (height : %d) ", height)
	h := big.NewInt(height)
	key := append([]byte(ReceiveBlockPrefix), h.Bytes()...)
	iter := b.db.NewIterator(util.BytesPrefix([]byte(ReceiveBlockPrefix)), nil)
	if iter.Seek(key) {
		for iter.Prev() {
			b.l.Debugf("Delete height stored in database (height : %d)",
				new(big.Int).SetBytes(iter.Key()[len([]byte(ReceiveBlockPrefix)):]).Int64())
			if err := b.db.Delete(iter.Key(), nil); err != nil {
				return err
			}
		}
	}
	return b.db.Delete(key, nil)
	return nil
}

func (b *btp2) setLastReceiveHeight(height int64) error {
	bytesArray := big.NewInt(height).Bytes()
	return b.db.Put([]byte("LastReceiveHeight"), bytesArray, nil)
}

func (b *btp2) getLastReceiveHeight() (int64, error) {
	if has, err := b.db.Has([]byte("LastReceiveHeight"), nil); err != nil {
		return 0, err
	} else if has {
		bytesArray, err := b.db.Get([]byte("LastReceiveHeight"), nil)
		if err != nil {
			return 0, err
		}
		return new(big.Int).SetBytes(bytesArray).Int64(), nil
	}
	return 0, nil
}

func (b *btp2) deleteAllDatabase() error {
	b.l.Debugf("deleteAllDatabase")
	iter := b.db.NewIterator(nil, nil)
	for iter.Next() {
		b.db.Delete(iter.Key(), nil)
	}
	return nil
}

func (b *btp2) prepareDatabase(baseDir string) error {
	var err error
	dbDir := filepath.Join(baseDir, b.src.GetAddress().NetworkAddress())
	b.l.Debugln("open database", dbDir)
	b.db, err = leveldb.OpenFile(dbDir, nil)
	if err != nil {
		return errors.Wrap(err, "fail to open database")
	}
	defer func() {
		if err != nil {
			b.db.Close()
		}
	}()
	return nil
}

func (b *btp2) getNetworkId() error {
	if b.nid == 0 {
		nid, err := b.c.GetBTPLinkNetworkId(b.src.GetAddress(), b.dst)
		if err != nil {
			return err
		}
		b.nid = nid
	}

	return nil
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
		err := b.monitoring(bls)
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
	rs := b.getReceiveStatusForSequence(seq)
	if rs != nil {
		return rs.height
	} else {
		return 0
	}
}

func (b *btp2) BuildBlockUpdate(bls *types.BMCLinkStatus, limit int64) ([]link.BlockUpdate, error) {
	b.l.Debugf("Build BlockUpdate (height:%d, rxSeq:%d)", bls.Verifier.Height, bls.RxSeq)
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
	b.l.Debugf("Build BuildMessageProof (height:%d, rxSeq:%d)", bls.Verifier.Height, bls.RxSeq)
	rs := b.getReceiveStatusForHeight(bls.Verifier.Height)

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

		b.l.Debugf("BuildRelayMessage (type:%d, len:%d)", rmi.Type(), rmi.Len())
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
				b.removeReceiveBlockByHeight(bls.Verifier.Height)
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
	msgs, err := b.c.GetBTPMessage(height, b.nid)
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

func (b *btp2) monitoring(bls *types.BMCLinkStatus) error {
	var height int64

	if bls.Verifier.Height < 1 {
		return fmt.Errorf("cannot catchup from zero height")
	}

	if bls.RxSeq < 1 {
		if err := b.deleteAllDatabase(); err != nil {
			return err
		}
	}

	lastHeight, err := b.getLastReceiveHeight()
	if err != nil {
		return err
	}

	fb := b.getFirstHeightForReceiveBlock()
	if bls.Verifier.Height > lastHeight {
		b.deleteAllDatabase()
		height = bls.Verifier.Height
	} else {
		if fb != 0 && bls.Verifier.Height > fb {
			height = bls.Verifier.Height
		} else if fb > bls.Verifier.Height {
			height = fb - 1
		} else {
			height = lastHeight
		}
		if err := b.removeAllReceiveBlock(); err != nil {
			return err
		}
	}

	req := &client.BTPRequest{
		Height:           client.NewHexInt(height + 1),
		NetworkID:        client.NewHexInt(b.nid),
		ProofFlag:        client.NewHexInt(0),
		ProgressInterval: client.NewHexInt(int64(DefaultProgressInterval)),
	}

	onErr := func(conn *websocket.Conn, err error) {
		b.l.Debugf("onError %s err:%+v", conn.LocalAddr().String(), err)
		b.c.CloseMonitor(conn)
	}
	onConn := func(conn *websocket.Conn) {
		b.l.Debugf("ReceiveLoop monitorBTP2Block height:%d seq:%d networkId:%d connected %s",
			height, bls.RxSeq, b.nid, conn.LocalAddr().String())
	}

	err = b.monitorBTP2Block(req, bls, onConn, onErr)
	if err != nil {
		return err
	}
	return nil
}

func (b *btp2) monitorBTP2Block(req *client.BTPRequest, bls *types.BMCLinkStatus, scb func(conn *websocket.Conn), errCb func(*websocket.Conn, error)) error {
	vs := &client.VerifierStatus{}
	_, err := codec.RLP.UnmarshalFromBytes(bls.Verifier.Extra, vs)
	if err != nil {
		return err
	}

	if bls.RxSeq != 0 {
		b.seq = bls.RxSeq + vs.SequenceOffset
	}

	return b.c.MonitorBTP(req, func(conn *websocket.Conn, v *client.BTPNotification) error {
		h, err := base64.StdEncoding.DecodeString(v.Header)
		if err != nil {
			return err
		}

		if v.Progress.Value != 0 {
			if err := b.setLastReceiveHeight(v.Progress.Value); err != nil {
				return err
			}
		}

		if len(v.Header) == 0 {
			return nil
		}

		bh := &client.BTPBlockHeader{}
		if _, err = codec.RLP.UnmarshalFromBytes(h, bh); err != nil {
			return err
		}

		if bh.MainHeight != b.startHeight {

			if bh.MessageCount != 0 {
				msgs, err := b.c.GetBTPMessage(bh.MainHeight, b.nid)
				if err != nil {
					return err
				}

				messageSN := bh.UpdateNumber >> 1
				if messageSN != b.seq {
					return fmt.Errorf("invalid seq (UpdateNumber:%d, Seq:%d)", bh.UpdateNumber, b.seq)
				}
				b.seq += int64(len(msgs))
			}

			if err := b.addReceiveBlock(bh.MainHeight, h); err != nil {
				return err
			}

			rs, err := newReceiveStatus(bh.MainHeight, b.seq)
			if err != nil {
				return err
			}
			b.rss = append(b.rss, rs)
			b.l.Debugf("monitor info : Height:%d  UpdateNumber:%d  MessageCnt:%d  Seq:%d ", bh.MainHeight, bh.UpdateNumber, bh.MessageCount, b.seq)
			b.rsc <- rs
		}

		return nil
	}, scb, errCb)
}

func (b *btp2) getReceiveStatusForSequence(seq int64) *receiveStatus {
	for _, rs := range b.rss {
		if rs.Seq() <= seq && seq <= rs.Seq() {
			return rs
		}
	}
	return nil
}

func (b *btp2) getReceiveStatusForHeight(height int64) *receiveStatus {
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
