package link

import (
	"fmt"
	"math/rand"
	"sync"

	"github.com/icon-project/btp2/chain"
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
	id            int
	bls           *types.BMCLinkStatus
	bpHeight      int64
	message       []byte
	rmis          []RelayMessageItem
	sendingStatus bool
}

func (r *relayMessage) Id() int {
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

func (r *relayMessage) BpHeight() int64 {
	return r.bpHeight
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
	src        types.BtpAddress
	dst        types.BtpAddress
	rmsMtx     sync.RWMutex
	rms        []*relayMessage
	rss        []ReceiveStatus
	rmi        *relayMessageItem
	limitSize  int64
	cfg        *chain.Config //TODO config refactoring
	bls        *types.BMCLinkStatus
	blsChannel chan *types.BMCLinkStatus
	relayState RelayState
}

func NewLink(cfg *chain.Config, r Receiver, l log.Logger) types.Link {
	link := &Link{
		src: cfg.Src.Address,
		dst: cfg.Dst.Address,
		l:   l.WithFields(log.Fields{log.FieldKeyChain: fmt.Sprintf("%s", cfg.Dst.Address.NetworkID())}),
		cfg: cfg,
		r:   r,
		rms: make([]*relayMessage, 0),
		rss: make([]ReceiveStatus, 0),
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

func (l *Link) Start(sender types.Sender) error {
	l.s = sender
	errCh := make(chan error)
	if err := l.senderChannel(errCh); err != nil {
		return err
	}

	bls, err := l.s.GetStatus()
	if err != nil {
		return err
	}

	l.bls = bls

	if err := l.receiverChannel(errCh); err != nil {
		return err
	}

	l.r.FinalizedStatus(l.blsChannel)

	for {
		select {
		case err := <-errCh:
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (l *Link) Stop() {
	l.s.Stop()
	l.r.Stop()
}

func (l *Link) receiverChannel(errCh chan error) error {
	once := new(sync.Once)
	rsc, err := l.r.Start(l.bls)
	if err != nil {
		return err
	}
	go func() {
		for {
			select {
			case rs := <-rsc:
				switch t := rs.(type) {
				case ReceiveStatus:
					l.rss = append(l.rss, t)

					once.Do(func() {
						if err = l.handleUndeliveredRelayMessage(); err != nil {
							errCh <- err
						}

						if err = l.HandleRelayMessage(); err != nil {
							errCh <- err
						}
						l.relayState = PENDING
					})

					if l.bls.Verifier.Height < rs.Height() {
						if err = l.HandleRelayMessage(); err != nil {
							errCh <- err
						}
					}
				case error:
					errCh <- t
				}
			}
		}

		select {
		case errCh <- err:
		default:
		}
	}()
	return nil
}

func (l *Link) senderChannel(errCh chan error) error {
	l.limitSize = int64(l.s.TxSizeLimit()) - l.s.GetMarginForLimit()
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

		select {
		case errCh <- err:
		default:
		}
	}()
	return nil
}

func (l *Link) buildRelayMessage() error {
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
			l.rmi.rmis[len(l.rmi.rmis)-1] = append(l.rmi.rmis[len(l.rmi.rmis)-1], bu)
			l.rmi.size += bu.Len()
			if err := bu.UpdateBMCLinkStatus(l.bls); err != nil {
				return err
			}

			rs := l.getReceiveStatusForHeight()
			if l.bls.RxSeq < rs.Seq() {
				if err = l.buildProof(bu); err != nil {
					return err
				}

				if err = l.appendRelayMessage(); err != nil {
					return err
				}
			} else {
				//only blockUpdate is received
				if l.cfg.Src.FilledBlockUpdate == true {
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
			}
		}
	}

	return nil
}

func (l *Link) sendRelayMessage() error {
	for _, rm := range l.rms {
		if rm.sendingStatus == false {

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
			id:       rand.Int(),
			bls:      l.bls,
			bpHeight: l.r.GetHeightForSeq(l.bls.RxSeq),
			message:  m,
			rmis:     rmi,
		}

		rm.sendingStatus = false
		l.rms = append(l.rms, rm)
	}

	l.rmi.rmis = l.rmi.rmis[:0]
	l.resetRelayMessageItem()

	return nil
}

func (l *Link) HandleRelayMessage() error {
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
				break
			}
		}
	}
	return nil
}

func (l *Link) buildBlockUpdates(bs *types.BMCLinkStatus) ([]BlockUpdate, error) {
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
	lastSeq := l.bls.RxSeq
	for {
		h := l.r.GetHeightForSeq(lastSeq)
		if h == 0 {
			break
		}
		if h == l.bls.Verifier.Height {
			mp, err := l.r.BuildMessageProof(l.bls, l.limitSize-l.rmi.size)
			if err != nil {
				return err
			}

			if mp == nil {
				break
			}

			if mp.Len() != 0 || l.bls.RxSeq < mp.LastSeqNum() {
				l.rmi.rmis[len(l.rmi.rmis)-1] = append(l.rmi.rmis[len(l.rmi.rmis)-1], mp)
				l.rmi.size += mp.Len()
			}
			break
		} else if h < l.bls.Verifier.Height {
			err := l.buildProof(nil)
			if err != nil {
				return err
			}
		} else {
			break
		}
	}
	if l.rmi.size > 0 {
		l.appendRelayMessage()
	}
	return nil
}

func (l *Link) buildProof(bu BlockUpdate) error {
	rs := l.getReceiveStatusForHeight()
	if rs == nil {
		return nil
	}
	for {
		//TODO refactoring
		if rs.Seq() <= l.bls.RxSeq {
			break
		}
		if l.isOverLimit(l.rmi.size) {
			l.appendRelayMessage()
			if err := l.buildBlockProof(l.bls); err != nil {
				return err
			}
		} else {
			if bu == nil || bu.ProofHeight() == -1 {
				if err := l.buildBlockProof(l.bls); err != nil {
					return err
				}
			}
		}
		if err := l.buildMessageProof(); err != nil {
			return err
		}
	}
	return nil
}

func (l *Link) buildMessageProof() error {
	mp, err := l.r.BuildMessageProof(l.bls, l.limitSize-l.rmi.size)
	if err != nil {
		return err
	}
	if mp != nil {
		l.rmi.rmis[len(l.rmi.rmis)-1] = append(l.rmi.rmis[len(l.rmi.rmis)-1], mp)
		l.rmi.size += mp.Len()
		if err := mp.UpdateBMCLinkStatus(l.bls); err != nil {
			return err
		}
	}
	return nil
}

func (l *Link) buildBlockProof(bls *types.BMCLinkStatus) error {
	h := l.r.GetHeightForSeq(bls.RxSeq)
	bf, err := l.r.BuildBlockProof(bls, h)
	if err != nil {
		return err
	}

	if bf != nil {
		l.rmi.rmis[len(l.rmi.rmis)-1] = append(l.rmi.rmis[len(l.rmi.rmis)-1], bf)
		l.rmi.size += bf.Len()
		if err := bf.UpdateBMCLinkStatus(bls); err != nil {
			return err
		}

	}
	return nil
}

func (l *Link) getReceiveStatusForHeight() ReceiveStatus {
	for _, rs := range l.rss {
		if rs.Height() == l.bls.Verifier.Height {
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

func (l *Link) getRelayMessageForId(id int) *relayMessage {
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

func (l *Link) updateBlockProof(id int) error {
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
	if int64(l.s.TxSizeLimit()) < size {
		return true
	}
	return false
}

func (l *Link) resetRelayMessageItem() {
	l.rmi.rmis = append(l.rmi.rmis, make([]RelayMessageItem, 0))
	l.rmi.size = 0
}

func (l *Link) successRelayMessage(id int) error {
	rm := l.getRelayMessageForId(id)
	l.removeRelayMessage(rm.BMCLinkStatus())
	l.removeReceiveStatus(rm.BMCLinkStatus())

	l.relayState = RUNNING

	if err := l.HandleRelayMessage(); err != nil {
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
			if l.cfg.Dst.LatestResult == true {
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
				l.HandleRelayMessage()
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
						if err := l.HandleRelayMessage(); err != nil {
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
