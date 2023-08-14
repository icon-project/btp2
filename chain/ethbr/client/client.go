/*
 * Copyright 2022 ICON Foundation
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

package client

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/icon-project/btp2/common/log"
)

const (
	DefaultSendTransactionRetryInterval        = 3 * time.Second         //3sec
	DefaultGetTransactionResultPollingInterval = 1500 * time.Millisecond //1.5sec
	DefaultTimeout                             = 10 * time.Second        //
	ChainID                                    = 56
	DefaultGasLimit                            = 8000000
	DefaultGasPrice                            = 5000000000
)

var (
	tendermintLightClientContractAddr = common.HexToAddress("0x0000000000000000000000000000000000001003")
	BlockRetryInterval                = time.Second * 3
	BlockRetryLimit                   = 5
)

type Client struct {
	uri       string
	log       log.Logger
	ethClient *ethclient.Client
	rpcClient *rpc.Client
	chainID   *big.Int
	stop      <-chan bool
}

func toBlockNumArg(number *big.Int) string {
	if number == nil {
		return "latest"
	}
	pending := big.NewInt(-1)
	if number.Cmp(pending) == 0 {
		return "pending"
	}
	return hexutil.EncodeBig(number)
}

func newTransaction(nonce uint64, to common.Address, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte) *types.Transaction {
	return types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		To:       &to,
		Value:    amount,
		Gas:      gasLimit,
		GasPrice: gasPrice,
		Data:     data,
	})
}

func (c *Client) GetEthClient() *ethclient.Client {
	return c.ethClient
}

func (c *Client) NewTransactOpts(k *ecdsa.PrivateKey) (*bind.TransactOpts, error) {
	txo, err := bind.NewKeyedTransactorWithChainID(k, c.chainID)
	if err != nil {
		return nil, err
	}
	txo.GasPrice, _ = c.ethClient.SuggestGasPrice(context.Background())
	txo.GasLimit = uint64(DefaultGasLimit)
	return txo, nil
}

func (c *Client) SignTransaction(signerKey *ecdsa.PrivateKey, tx *types.Transaction) error {
	signer := types.LatestSignerForChainID(c.chainID)
	tx, err := types.SignTx(tx, signer, signerKey)
	if err != nil {
		c.log.Errorf("could not sign tx: %v", err)
		return err
	}
	return nil
}

func (c *Client) SendTransaction(tx *types.Transaction) error {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultTimeout)
	defer cancel()

	err := c.ethClient.SendTransaction(ctx, tx)

	if err != nil {
		c.log.Errorf("could not send tx: %v", err)
		return nil
	}

	return nil
}

func (c *Client) GetTransactionReceipt(hash common.Hash) (*types.Receipt, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultTimeout)
	defer cancel()
	tr, err := c.ethClient.TransactionReceipt(ctx, hash)
	if err != nil {
		return nil, err
	}
	return tr, nil
}

func (c *Client) GetTransaction(hash common.Hash) (*types.Transaction, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultTimeout)
	defer cancel()
	tx, pending, err := c.ethClient.TransactionByHash(ctx, hash)
	if err != nil {
		return nil, pending, err
	}
	return tx, pending, err
}

func (c *Client) GetBlockByHeight(height *big.Int) (*types.Block, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultTimeout)
	defer cancel()
	return c.ethClient.BlockByNumber(ctx, height)
}

func (c *Client) GetHeaderByHeight(height *big.Int) (*types.Header, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultTimeout)
	defer cancel()
	return c.ethClient.HeaderByNumber(ctx, height)
}

func (c *Client) GetProof(height *big.Int, addr common.Address) (StorageProof, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultTimeout)
	defer cancel()
	var proof StorageProof
	if err := c.rpcClient.CallContext(ctx, &proof, "eth_getProof", addr, nil, toBlockNumArg(height)); err != nil {
		return proof, err
	}
	return proof, nil
}

func (c *Client) GetBlockReceipts(block *types.Block) ([]*types.Receipt, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultTimeout)
	defer cancel()
	var receipts []*types.Receipt
	for _, tx := range block.Transactions() {
		receipt, err := c.ethClient.TransactionReceipt(ctx, tx.Hash())
		if err != nil {
			return nil, err
		}
		receipts = append(receipts, receipt)
	}
	return receipts, nil
}

func (c *Client) FilterLogs(fq ethereum.FilterQuery) ([]types.Log, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultTimeout)
	defer cancel()
	logs, err := c.ethClient.FilterLogs(ctx, fq)
	if err != nil {
		return nil, err
	}
	return logs, nil
}

func (c *Client) GetChainID() (*big.Int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultTimeout)
	defer cancel()
	return c.ethClient.ChainID(ctx)
}

func (c *Client) GetBlockNumber() (uint64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultTimeout)
	defer cancel()
	return c.ethClient.BlockNumber(ctx)
}

// Poll deprecated
func (c *Client) Poll(cb func(bh *types.Header) error, errCb func(int64, error)) error {
	n, err := c.GetBlockNumber()
	if err != nil {
		return err
	}
	current := new(big.Int).SetUint64(n)
	for {
		select {
		case <-c.stop:
			return nil
		default:
			var bh *types.Header
			if bh, err = c.GetHeaderByHeight(current); err != nil {
				if ethereum.NotFound == err {
					c.log.Trace("Block not ready, will retry ", current)
				} else {
					c.log.Error("Unable to get block ", current, err)
				}
				<-time.After(BlockRetryInterval)
				continue
			}

			if err = cb(bh); err != nil {
				c.log.Errorf("Poll callback return err:%+v", err)
				errCb(bh.Number.Int64()-1, err)
			}

			current.Add(current, big.NewInt(1))
		}
	}
}

func (c *Client) MonitorBlock(br *BlockRequest, cb func(b *BlockNotification) error, errCb func(int64, error)) error {
	onBlockHeader := func(bh *types.Header) error {
		bn := &BlockNotification{
			Hash:   bh.Hash(),
			Height: bh.Number,
			Header: bh,
		}
		if br.FilterQuery != nil {
			var err error
			fq := *br.FilterQuery
			fq.BlockHash = &bn.Hash
			if bn.Logs, err = c.FilterLogs(fq); err != nil {
				c.log.Info("Unable to get logs ", err)
				return err
			}
		}
		return cb(bn)
	}
	var (
		h   *big.Int
		one = big.NewInt(1)
	)
	return c.Monitor(func(bh *types.Header) error {
		if h == nil {
			h = new(big.Int).Set(br.Height)
			for ; h.Cmp(bh.Number) < 0; h = h.Add(h, one) {
				for {
					tbh, err := c.GetHeaderByHeight(h)
					if err != nil {
						c.log.Debugf("failure GetHeaderByHeight(%v) err:%+v", h, err)
						continue
					} else {
						err = onBlockHeader(tbh)
						if err != nil {
							return err
						}
						break
					}
				}
			}
		}
		return onBlockHeader(bh)
	}, errCb)
}

func (c *Client) Monitor(cb func(bh *types.Header) error, errCb func(int64, error)) error {
	if strings.HasPrefix(c.uri, "http") {
		return c.Poll(cb, errCb)
	}
	var (
		s   ethereum.Subscription
		err error
		ch  = make(chan *types.Header)
	)
	if s, err = c.ethClient.SubscribeNewHead(context.Background(), ch); err != nil {
		if rpc.ErrNotificationsUnsupported == err {
			c.log.Infoln("%v, try polling", err)
			return c.Poll(cb, errCb)
		}
		return err
	}
	for {
		select {
		case err = <-s.Err():
			return err
		case bh := <-ch:
			if err = cb(bh); err != nil {
				c.log.Errorf("MonitorBlock callback return err:%+v", err)
				return err
			}
			c.log.Debugf("MonitorBlock %v", bh.Number.Int64())
		}
	}
}

func (c *Client) CloseMonitor() {
	c.log.Debugf("CloseMonitor %s", c.rpcClient)
	c.ethClient.Close()
	c.rpcClient.Close()
}

func (c *Client) CloseAllMonitor() {
	// TODO: do we need to multiple connections?
	c.CloseMonitor()
}

func (c *Client) GetBackend() bind.ContractBackend {
	return c.ethClient
}

func (c *Client) GetRevertMessage(hash common.Hash) (string, error) {
	tx, _, err := c.ethClient.TransactionByHash(context.Background(), hash)
	if err != nil {
		return "", err
	}

	from, err := types.Sender(types.NewEIP155Signer(tx.ChainId()), tx)
	if err != nil {
		return "", err
	}

	msg := ethereum.CallMsg{
		From:     from,
		To:       tx.To(),
		Gas:      tx.Gas(),
		GasPrice: tx.GasPrice(),
		Value:    tx.Value(),
		Data:     tx.Data(),
	}

	_, err = c.ethClient.CallContract(context.Background(), msg, nil)
	return err.Error(), nil

}

func NewClient(uri string, l log.Logger) *Client {
	//TODO options {MaxRetrySendTx, MaxRetryGetResult, MaxIdleConnsPerHost, Debug, Dump} }
	rpcClient, err := rpc.Dial(uri)
	if err != nil {
		l.Fatal("Error creating client", err)
	}
	c := &Client{
		uri:       uri,
		rpcClient: rpcClient,
		ethClient: ethclient.NewClient(rpcClient),
		log:       l,
	}
	c.chainID, _ = c.GetChainID()
	l.Tracef("Client Connected Chain ID: ", c.chainID)
	if err != nil {
		c.log.Error("Error creating tendermintLightclient system contract", err)
	}
	return c
}
