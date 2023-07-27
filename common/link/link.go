package link

import (
	"strconv"
	"sync"

	"github.com/icon-project/btp2/common/errors"
	"github.com/icon-project/btp2/common/log"
	"github.com/icon-project/btp2/common/types"
)

type RelayState int

const (
	RUNNING = iota
	PENDING
)

type relayMessage struct {
	id            string
	bls           *types.BMCLinkStatus
	message       []byte
	rmis          []RelayMessageItem
	sendingStatus bool
}

func (r *relayMessage) Id() string {
	return r.id
}

func (r *relayMessage) Bytes() []byte {
	return r.message
}

func (r *relayMessage) Size() int64 {
	return int64(len(r.message))
}

func (r *relayMessage) BMCLinkStatus() *types.BMCLinkStatus {
	return r.bls
}

func (r *relayMessage) RelayMessageItems() []RelayMessageItem {
	return r.rmis
}

type relayMessageItem struct {
	rmis [][]RelayMessageItem
	size int64
}

type Link struct {
	r          Receiver
	s          types.Sender
	l          log.Logger
	mtx        sync.RWMutex
	rmsMtx     sync.RWMutex
	rms        []*relayMessage
	rss        []ReceiveStatus
	rmi        *relayMessageItem
	limitSize  int64
	srcCfg     ChainConfig
	bls        *types.BMCLinkStatus
	blsChannel chan *types.BMCLinkStatus
	relayState RelayState
	p          types.Preference
}

func NewLink(srcCfg ChainConfig, r Receiver, l log.Logger) types.Link {
	link := &Link{
		l:      l,
		srcCfg: srcCfg,
		r:      r,
		rms:    make([]*relayMessage, 0),
		rss:    make([]ReceiveStatus, 0),
		rmi: &relayMessageItem{
			rmis: make([][]RelayMessageItem, 0),
			size: 0,
		},
		blsChannel: make(chan *types.BMCLinkStatus),
		relayState: RUNNING,
	}
	link.rmi.rmis = append(link.rmi.rmis, make([]RelayMessageItem, 0))
	return link
}

func (l *Link) Start(sender types.Sender, errChan chan error) error {
	l.s = sender
	l.p = sender.GetPreference()

	if err := l.startSenderChannel(errChan); err != nil {
		return err
	}

	bls, err := l.s.GetStatus()
	if err != nil {
		return err
	}

	l.bls = bls

	if err := l.startReceiverChannel(errChan); err != nil {
		return err
	}
	l.r.FinalizedStatus(l.blsChannel)

	return nil
}

func (l *Link) Stop() {
	l.s.Stop()
	l.r.Stop()
}

func (l *Link) startReceiverChannel(errCh chan error) error {
	once := new(sync.Once)
	rc, err := l.r.Start(l.bls)
	if err != nil {
		return err
	}
	go func() {
		for {
			select {
			case rsc := <-rc:
				switch t := rsc.(type) {
				case ReceiveStatus:
					rs := t.(ReceiveStatus)
					l.rss = append(l.rss, t)
					l.l.Debugf("ReceiveStatus (height:%d, seq:%d)", rs.Height(), rs.Seq())
					once.Do(func() {
						if err = l.handleUndeliveredRelayMessage(); err != nil {
							errCh <- err
						}

						if err = l.handleRelayMessage(); err != nil {
							errCh <- err
						}
						l.relayState = PENDING
					})

					if l.bls.Verifier.Height < rs.Height() {
						if err = l.handleRelayMessage(); err != nil {
							errCh <- err
						}
					}
				case error:
					errCh <- t
				}
			}
		}
	}()
	return nil
}

func (l *Link) startSenderChannel(errCh chan error) error {
	l.limitSize = l.p.TxSizeLimit - l.p.MarginForLimit
	rcc, err := l.s.Start()
	if err != nil {
		return err
	}
	go func() {
		for {
			select {
			case rc := <-rcc:
				err := l.result(rc)
				errCh <- err
			}
		}
	}()
	return nil
}

func (l *Link) buildRelayMessage() error {
	l.l.Debugf("BuildRelayMessage (bls height:%d, bls rx seq:%d)", l.bls.Verifier.Height, l.bls.RxSeq)
	if len(l.rmi.rmis) == 0 {
		l.resetRelayMessageItem()
	}

	//Get Block
	bus, err := l.buildBlockUpdates(l.bls)
	if err != nil {
		return err
	}

	if len(bus) != 0 {
		for _, bu := range bus {
			l.appendRelayMessageItem(bu)
			if err := bu.UpdateBMCLinkStatus(l.bls); err != nil {
				return err
			}

			mpLen, err := l.buildProof(bu)
			if err != nil {
				return err
			}

			if mpLen == 0 {
				if l.p.FilledBlockUpdate == true {
					if l.isOverLimit(l.rmi.size) {
						if err = l.appendRelayMessage(); err != nil {
							return err
						}
					}
				} else {
					if err = l.appendRelayMessage(); err != nil {
						return err
					}
				}
			} else {
				if err = l.appendRelayMessage(); err != nil {
					return err
				}
			}

		}
	}

	return nil
}

func (l *Link) sendRelayMessage() error {
	for _, rm := range l.rms {
		if rm.sendingStatus == false {
			l.l.Debugf("SendRelayMessage (bls height:%d, bls txSeq:%d, bls rxSeq:%d)",
				rm.bls.Verifier.Height, rm.bls.TxSeq, rm.bls.RxSeq)
			_, err := l.s.Relay(rm)
			if err != nil {
				if errors.InvalidStateError.Equals(err) {
					l.relayState = PENDING
					return nil
				} else {
					return err
				}
			} else {
				rm.sendingStatus = true
			}
		}
	}
	return nil
}

func (l *Link) appendRelayMessage() error {
	for _, rmi := range l.rmi.rmis {
		m, err := l.r.BuildRelayMessage(rmi)
		if err != nil {
			return err
		}

		rm := &relayMessage{
			id:      l.srcCfg.GetAddress().NetworkID() + "_" + strconv.FormatInt(l.bls.Verifier.Height, 16) + "_" + strconv.FormatInt(l.bls.RxSeq, 16),
			bls:     &types.BMCLinkStatus{},
			message: m,
			rmis:    rmi,
		}

		rm.bls.TxSeq = l.bls.TxSeq
		rm.bls.RxSeq = l.bls.RxSeq
		rm.bls.Verifier.Height = l.bls.Verifier.Height
		copy(rm.bls.Verifier.Extra, l.bls.Verifier.Extra)

		rm.sendingStatus = false
		l.rms = append(l.rms, rm)
		l.l.Debugf("AppendRelayMessage (bls height:%d, bls txSeq:%d, bls rxSeq:%d)",
			rm.bls.Verifier.Height, rm.bls.TxSeq, rm.bls.RxSeq)
	}

	l.rmi.rmis = l.rmi.rmis[:0]
	l.resetRelayMessageItem()

	return nil
}

func (l *Link) handleRelayMessage() error {
	l.rmsMtx.Lock()
	defer l.rmsMtx.Unlock()
	if l.relayState == RUNNING {
		if err := l.sendRelayMessage(); err != nil {
			return err
		}

		for true {
			if l.relayState == RUNNING &&
				len(l.rss) != 0 &&
				l.bls.Verifier.Height < l.rss[len(l.rss)-1].Height() {
				l.buildRelayMessage()
				l.sendRelayMessage()
			} else {
				l.l.Debugf("Relay status : %d, ReceiveStatus size: %d", l.relayState, len(l.rss))
				break
			}
		}
	}
	return nil
}

func (l *Link) buildBlockUpdates(bs *types.BMCLinkStatus) ([]BlockUpdate, error) {
	l.l.Debugf("BuildBlockUpdates (bls height:%d, bls rxSeq:%d)", l.bls.Verifier.Height, l.bls.RxSeq)
	for {
		bus, err := l.r.BuildBlockUpdate(bs, l.limitSize-l.rmi.size)
		if err != nil {
			return nil, err
		}
		if len(bus) != 0 {
			return bus, nil
		}
	}
}

func (l *Link) handleUndeliveredRelayMessage() error {
	rs := l.getReceiveStatusForHeight(l.bls.Verifier.Height)
	if rs == nil {
		return nil
	}
	for l.bls.RxSeq < rs.Seq() {
		l.l.Debugf("HandleUndeliveredRelayMessage ReceiveStatus(height : %d, seq : %s), BMCLinkStatus(height : %d, seq : %s)",
			rs.Height(), rs.Seq(), l.bls.Verifier.Height, l.bls.RxSeq)
		_, err := l.buildProof(nil)
		if err != nil {
			return err
		}
	}

	if l.rmi.size > 0 {
		l.appendRelayMessage()
	}
	return nil
}

func (l *Link) buildProof(bu BlockUpdate) (int64, error) {
	l.l.Debugf("BuildProof (bls height:%d, bls rx seq:%d)", l.bls.Verifier.Height, l.bls.RxSeq)
	var mpLen int64
	rs := l.getReceiveStatusForHeight(l.bls.Verifier.Height)
	if rs == nil {
		return 0, nil
	}
	for {
		if rs.Seq() <= l.bls.RxSeq {
			break
		}

		mp, err := l.buildMessageProof()
		if err != nil {
			return 0, err
		}

		if mp == nil || mp.Len() == 0 {
			if len(l.rmi.rmis) != 0 {
				l.appendRelayMessage()
				continue
			} else {
				return 0, nil
			}
		}

		mpLen += mp.Len()
		if l.isOverLimit(l.rmi.size) {
			l.appendRelayMessage()
			bp, err := l.buildBlockProof(l.bls)
			if err != nil {
				return 0, err
			}
			l.appendRelayMessageItem(bp)
		} else {
			if bu == nil || bu.ProofHeight() == -1 {
				bp, err := l.buildBlockProof(l.bls)
				if err != nil {
					return 0, err
				}
				l.appendRelayMessageItem(bp)
			}
		}
		l.appendRelayMessageItem(mp)
	}
	return mpLen, nil
}

func (l *Link) buildMessageProof() (MessageProof, error) {
	mp, err := l.r.BuildMessageProof(l.bls, l.limitSize-l.rmi.size)
	if err != nil {
		return nil, err
	}
	if mp != nil {
		if err := mp.UpdateBMCLinkStatus(l.bls); err != nil {
			return nil, err
		}
	}
	return mp, nil
}

func (l *Link) buildBlockProof(bls *types.BMCLinkStatus) (BlockProof, error) {
	h := l.r.GetHeightForSeq(bls.RxSeq)
	bf, err := l.r.BuildBlockProof(bls, h)
	if err != nil {
		return nil, err
	}

	if bf != nil {
		if err := bf.UpdateBMCLinkStatus(bls); err != nil {
			return nil, err
		}
	}
	return bf, nil
}

func (l *Link) appendRelayMessageItem(rmi RelayMessageItem) {
	l.rmi.rmis[len(l.rmi.rmis)-1] = append(l.rmi.rmis[len(l.rmi.rmis)-1], rmi)
	l.rmi.size += rmi.Len()
}

func (l *Link) getReceiveStatusForHeight(height int64) ReceiveStatus {
	for _, rs := range l.rss {
		if rs.Height() == height {
			return rs
		}
	}
	return nil
}

func (l *Link) removeReceiveStatus(bls *types.BMCLinkStatus) {
	for i, rs := range l.rss {
		if rs.Height() <= bls.Verifier.Height && rs.Seq() <= bls.RxSeq {
			l.rss = l.rss[i+1:]
			break
		}
	}
}

func (l *Link) getRelayMessage(bls *types.BMCLinkStatus) *relayMessage {
	for _, rm := range l.rms {
		if bls.Verifier.Height == rm.bls.Verifier.Height && bls.RxSeq == rm.bls.RxSeq {
			return rm
		}
	}
	return nil
}

func (l *Link) getRelayMessageForId(id string) *relayMessage {
	for _, rm := range l.rms {
		if rm.Id() == id {
			return rm
		}
	}
	return nil
}

func (l *Link) removeRelayMessage(bls *types.BMCLinkStatus) int {
	index := 0
	for index, rm := range l.rms {
		if rm.bls.Verifier.Height <= bls.Verifier.Height && rm.bls.RxSeq <= bls.RxSeq {
			l.rms = l.rms[index+1:]
			break
		}
	}
	return index
}

func (l *Link) removeAllRelayMessage() {
	l.rms = l.rms[:0]
}

func (l *Link) updateBlockProof(id string) error {
	rm := l.getRelayMessageForId(id)

	for _, rmi := range rm.RelayMessageItems() {
		if rmi.Type() == TypeBlockProof {
			h := l.r.GetHeightForSeq(rm.bls.RxSeq)
			bf, err := l.r.BuildBlockProof(rm.bls, h)
			if err != nil {
				return err
			}
			rmi = bf
		}
	}
	return nil
}

func (l *Link) isOverLimit(size int64) bool {
	if l.p.TxSizeLimit < size {
		return true
	}
	return false
}

func (l *Link) resetRelayMessageItem() {
	l.rmi.rmis = append(l.rmi.rmis, make([]RelayMessageItem, 0))
	l.rmi.size = 0
}

func (l *Link) successRelayMessage(id string) error {
	rm := l.getRelayMessageForId(id)
	l.removeRelayMessage(rm.BMCLinkStatus())
	l.removeReceiveStatus(rm.BMCLinkStatus())

	l.relayState = RUNNING

	if err := l.handleRelayMessage(); err != nil {
		return err
	}
	l.blsChannel <- rm.BMCLinkStatus()
	return nil
}

func (l *Link) updateBMCLinkStatus() error {
	bls, err := l.s.GetStatus()
	if err != nil {
		return err
	}
	l.bls = bls
	return nil
}

func (l *Link) result(rr *types.RelayResult) error {
	rm := l.getRelayMessageForId(rr.Id)
	if rm != nil {
		switch rr.Err {
		case errors.SUCCESS:
			if l.p.LatestResult == true {
				l.successRelayMessage(rr.Id)
			} else {
				if rr.Finalized == true {
					l.successRelayMessage(rr.Id)
				}
			}
		case errors.BMVUnknown:
			l.l.Panicf("BMVUnknown Revert : ErrorCoder:%+v", rr.Err)
		case errors.BMVNotVerifiable:
			if rr.Finalized != true {
				l.relayState = PENDING
			} else {
				l.updateBMCLinkStatus()
				l.removeAllRelayMessage()
				l.relayState = RUNNING
				l.handleRelayMessage()
			}
		case errors.BMVAlreadyVerified:
			if rr.Finalized != true {
				l.relayState = PENDING
			} else {
				l.updateBMCLinkStatus()
				l.relayState = RUNNING
				index := l.removeRelayMessage(l.bls)
				if index == 0 {
					l.removeAllRelayMessage()
				} else {
					if l.rms[index].sendingStatus == false {
						if err := l.handleRelayMessage(); err != nil {
							return err
						}
					}
				}
			}
		case errors.BMVRevertInvalidBlockWitnessOld:
			//TODO Error handling required on Finalized
			l.updateBlockProof(rr.Id)
		default:
			l.l.Panicf("fail to GetResult RelayMessage ID:%v ErrorCoder:%+v",
				rr.Id, rr.Err)
		}
	}

	return nil
}
