package server

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/julienschmidt/httprouter"

	cbn "github.com/celer-network/cBridge-go/cbridgenode"
	"github.com/celer-network/cBridge-go/contracts"
	"github.com/celer-network/cBridge-go/gatewayrpc"
	"github.com/celer-network/cBridge-go/layer1"
	"github.com/celer-network/goutils/eth"
	"github.com/celer-network/goutils/eth/monitor"
	"github.com/celer-network/goutils/eth/watcher"
	"github.com/celer-network/goutils/log"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	solsha3 "github.com/miguelmota/go-solidity-sha3"
)

const (
	dbDriver   = "postgres"
	dbFmt      = "postgresql://cbridge@%s/cbridge?sslmode=disable"
	dbPoolSize = 20
	// on-chain event names
	evLogTransferOut       = "LogNewTransferOut"
	evLogTransferIn        = "LogNewTransferIn"
	evLogTransferConfirmed = "LogTransferConfirmed"
	evLogTransferRefunded  = "LogTransferRefunded"

	remoteTransferStatusUndefined = 0
	remoteTransferStatusPending   = 1
	remoteTransferStatusConfirmed = 2
	remoteTransferStatusRefunded  = 3
)

var (
	InvalidGasPriceChain = errors.New("no valid gas price in token chain")
)

type server struct {
	cfg          *cbn.CBridgeConfig // config from local json file
	chainMap     map[uint64]*bridgeConfig
	chainMapLock sync.Mutex
	accountAddr  Addr
	db           *DAL
	gateway      GatewayAPI
	signer       *eth.CelerSigner // sign req msg

	gatewayChainInfoMap map[uint64]*gatewayrpc.GatewayChainInfo

	//<fromToken, <toChainId, toChainToken>>
	chainTokenNameMap    map[Addr]string
	chainTokenAddrMap    map[uint64]map[string]Addr
	chainTokenDecimalMap map[Addr]uint64
	chainGasTokenMap     map[uint64]*chainGasTokenInfo

	gatewayChainInfoMapLock sync.Mutex
	// signal for goroutines to exit
	quit chan bool
}

type chainGasTokenInfo struct {
	GasTokenName    string
	GasTokenDecimal uint64
}

type bridgeConfig struct {
	config *cbn.ChainConfig

	chainId  *big.Int
	ec       *ethclient.Client
	trans    *eth.Transactor
	watch    *watcher.WatchService
	mon      *monitor.Service
	erc20Map map[Addr]*contracts.Erc20

	// on-chain contracts
	contractChain layer1.Contract
}

func NewServer() *server {
	return &server{
		chainMap:             make(map[uint64]*bridgeConfig),
		gatewayChainInfoMap:  make(map[uint64]*gatewayrpc.GatewayChainInfo),
		chainTokenNameMap:    make(map[Addr]string),
		chainTokenAddrMap:    make(map[uint64]map[string]Addr),
		chainTokenDecimalMap: make(map[Addr]uint64),
		chainGasTokenMap:     make(map[uint64]*chainGasTokenInfo),
		quit:                 make(chan bool),
	}
}

func (s *server) InitGatewayClient(gatewayUrl string) error {
	var err error
	s.gateway, err = NewGatewayAPI(gatewayUrl)
	if err != nil {
		log.Fatalln("fail to connect to gateway server:", err)
		return err
	}
	return nil
}

func (s *server) Init(config *cbn.CBridgeConfig) error {
	s.cfg = config
	var err error

	// init db
	log.Infoln("Initializing DB...")
	s.db, err = NewDAL(dbDriver, fmt.Sprintf(dbFmt, config.GetDb()), dbPoolSize)
	if err != nil {
		return err
	}
	log.Infoln("Successfully initialize DB")

	log.Infoln("Loading keystore...")
	ksjson, err := ioutil.ReadFile(config.GetKsPath())
	if err != nil {
		log.Fatalln("read keystore json err:", err)
		return err
	}
	tcfg := eth.NewTransactorConfig(string(ksjson), config.GetKsPwd())
	s.accountAddr, _, err = eth.GetAddrPrivKeyFromKeystore(string(ksjson), config.GetKsPwd())
	if err != nil {
		log.Errorf("fail to get relay node wallet from key store, err:%v", err)
		return err
	}
	s.signer, err = eth.NewSignerFromKeystore(string(ksjson), config.GetKsPwd(), big.NewInt(0))
	if err != nil {
		log.Errorf("fail to create relay node singer from key store, err:%v", err)
		return err
	}
	log.Infof("Successfully load keystore. Node addr:%s", s.accountAddr.String())

	for _, chainConfig := range config.GetChainConfig() {
		log.Infof("Initializing on chain %d...", chainConfig.GetChainId())
		bgc := &bridgeConfig{
			config:   chainConfig,
			erc20Map: map[Addr]*contracts.Erc20{},
		}
		bgc.ec, err = ethclient.Dial(chainConfig.GetEndpoint())
		if err != nil {
			return err
		}
		bgc.chainId, err = bgc.ec.ChainID(context.Background())
		if err != nil {
			return err
		}
		bgc.trans, err = eth.NewTransactor(
			tcfg.Keyjson,
			tcfg.Passphrase,
			bgc.ec,
			bgc.chainId,
			s.waitMinedOptions(bgc)...,
		)
		if err != nil {
			return err
		}
		bgc.contractChain, err = layer1.NewBoundContract(bgc.ec, Hex2Addr(chainConfig.ContractAddress), contracts.CBridgeABI)
		if err != nil {
			return err
		}
		s.chainGasTokenMap[chainConfig.GetChainId()] = &chainGasTokenInfo{
			GasTokenName:    chainConfig.GetGasTokenName(),
			GasTokenDecimal: chainConfig.GetGasTokenDecimal(),
		}
		for _, tokenConfig := range chainConfig.GetTokenConfig() {
			s.chainTokenNameMap[Hex2Addr(tokenConfig.TokenAddress)] = tokenConfig.GetTokenName()
			s.chainTokenDecimalMap[Hex2Addr(tokenConfig.TokenAddress)] = tokenConfig.GetTokenDecimal()
			if tokenConfig.GetTokenDecimal() <= 0 {
				return fmt.Errorf("find invalid token decimal, tokenConfig: %v", tokenConfig)
			}
			chainTokenAddr, foundChainTokenAddr := s.chainTokenAddrMap[bgc.chainId.Uint64()]
			if !foundChainTokenAddr {
				chainTokenAddr = make(map[string]Addr)
				s.chainTokenAddrMap[bgc.chainId.Uint64()] = chainTokenAddr
			}
			chainTokenAddr[tokenConfig.GetTokenName()] = Hex2Addr(tokenConfig.TokenAddress)

			bgc.erc20Map[Hex2Addr(tokenConfig.GetTokenAddress())], err = contracts.NewErc20(Hex2Addr(tokenConfig.GetTokenAddress()), bgc.ec)
			if err != nil {
				return err
			}

			ksfAc, err := os.Open(config.GetKsPath())
			if err != nil {
				return err
			}
			authAccount, err := bind.NewTransactorWithChainID(ksfAc, config.GetKsPwd(), bgc.chainId)
			if err != nil {
				return err
			}

			curAllowance, err := bgc.erc20Map[Hex2Addr(tokenConfig.GetTokenAddress())].Allowance(&bind.CallOpts{}, s.accountAddr, bgc.contractChain.GetAddr())
			if err != nil {
				return err
			}

			if curAllowance.Cmp(new(big.Int).Div(MaxUint256, big.NewInt(2))) < 0 {
				log.Infof("Approving token %s on chain %d...", tokenConfig.GetTokenName(), chainConfig.GetChainId())
				_, err = bgc.erc20Map[Hex2Addr(tokenConfig.GetTokenAddress())].Approve(authAccount, bgc.contractChain.GetAddr(), MaxUint256)
				if err != nil {
					return fmt.Errorf("Error when approving token %s on chain %d: %s", tokenConfig.GetTokenName(), chainConfig.GetChainId(), err)
				}
			}
		}

		s.chainMap[bgc.chainId.Uint64()] = bgc
		log.Infof("Successfully initialize chain %d", chainConfig.GetChainId())
	}

	smallDelay := func() {
		time.Sleep(200 * time.Millisecond)
	}

	// refresh ping first
	log.Infof("Registering in gateway: %s", config.GetGateway())
	err = s.PingAndRefreshFee()
	if err != nil {
		return err
	}
	log.Infof("Successfully registered in gateway: %s", config.GetGateway())

	// init monitoring
	for _, bgc := range s.chainMap {
		maxBlockDelta := uint64(5000)
		if bgc.config.WatchConfig.GetMaxBlockDelta() > 0 {
			maxBlockDelta = bgc.config.WatchConfig.GetMaxBlockDelta()
		}
		bgc.watch = watcher.NewWatchService(bgc.ec, s.db, bgc.config.WatchConfig.GetPollingInterval(), maxBlockDelta)
		if bgc.watch == nil {
			fmt.Println("Cannot setup watch service")
			return fmt.Errorf("NewWatchService failed: %w", err)
		}
		bgc.mon = monitor.NewService(bgc.watch, bgc.config.WatchConfig.GetBlockDelay(), true)
		bgc.mon.Init()
		_, err = s.monitorLogTransferOut(bgc)
		if err != nil {
			log.Errorf("can not start monitor for TransferOut, err:%v", err)
			return err
		}
		smallDelay()
		_, err = s.monitorLogTransferIn(bgc)
		if err != nil {
			log.Errorf("can not start monitor for TransferIn, err:%v", err)
			return err
		}
		smallDelay()
		_, err = s.monitorLogConfirm(bgc)
		if err != nil {
			log.Errorf("can not start monitor for confirm, err:%v", err)
			return err
		}
		smallDelay()
		_, err = s.monitorLogRefund(bgc)
		if err != nil {
			log.Errorf("can not start monitor for refund, err:%v", err)
			return err
		}
		smallDelay()
	}
	return nil
}

func (s *server) monitorLogTransferOut(bc *bridgeConfig) (monitor.CallbackID, error) {
	cfg := &monitor.Config{
		ChainId:    bc.chainId.Uint64(),
		EventName:  evLogTransferOut,
		Contract:   bc.contractChain,
		StartBlock: bc.mon.GetCurrentBlockNumber(),
	}
	return bc.mon.Monitor(cfg, func(id monitor.CallbackID, eLog ethtypes.Log) bool {
		log.Infof("get monitorLogTransferOut, block number:%d", eLog.BlockNumber)
		ev := &contracts.CBridgeLogNewTransferOut{}
		err := bc.contractChain.ParseEvent(evLogTransferOut, eLog, ev)
		if err != nil {
			log.Errorf("monitorLogTransferOut: cannot parse event, err:%v", err)
			return false
		}
		if ev.Receiver != s.accountAddr {
			log.Infof("this transfer out receiver is not current relay node")
			return false
		}

		_, found := s.chainMap[ev.DstChainId]
		if found {
			tokenName, foundTokenName := s.chainTokenNameMap[ev.Token]
			if !foundTokenName {
				log.Warnf("fail to get this token name, transferId:%s, token:%s", Hash(ev.TransferId).String(), time.Unix(int64(ev.Timelock), 0).String())
				return false
			}

			dstChainTokenMap, foundDisChainTokenMap := s.chainTokenAddrMap[ev.DstChainId]
			if !foundDisChainTokenMap {
				log.Warnf("fail to get this foundDisChainTokenMap, transferId:%s, token:%s", Hash(ev.TransferId).String(), time.Unix(int64(ev.Timelock), 0).String())
				return false
			}
			dstToken, foundDstToken := dstChainTokenMap[tokenName]
			if !foundDstToken {
				log.Warnf("fail to get this foundDstToken, transferId:%s, token:%s", Hash(ev.TransferId).String(), time.Unix(int64(ev.Timelock), 0).String())
				return false
			}

			srcTokenDecimal, foundSrcTokenDecimal := s.chainTokenDecimalMap[ev.Token]
			if !foundSrcTokenDecimal {
				log.Warnf("fail to get this foundSrcTokenDecimal, transferId:%s, token:%s", Hash(ev.TransferId).String(), time.Unix(int64(ev.Timelock), 0).String())
				return false
			}

			dstTokenDecimal, foundDstTokenDecimal := s.chainTokenDecimalMap[dstToken]
			if !foundDstTokenDecimal {
				log.Warnf("fail to get this foundDstTokenDecimal, transferId:%s, token:%s", Hash(ev.TransferId).String(), time.Unix(int64(ev.Timelock), 0).String())
				return false
			}

			dstAmount := ev.Amount
			log.Infof("srcAmount:%s, srcTokenDecimal:%d, dstTokenDecimal:%d", ev.Amount.String(), srcTokenDecimal, dstTokenDecimal)
			if srcTokenDecimal > dstTokenDecimal {
				p := uint64(1)
				for i := uint64(0); i < (srcTokenDecimal - dstTokenDecimal); i++ {
					p = p * 10
				}
				dstAmount = new(big.Int).Div(ev.Amount, new(big.Int).SetUint64(p))
			} else if srcTokenDecimal < dstTokenDecimal {
				p := uint64(1)
				for i := uint64(0); i < (dstTokenDecimal - srcTokenDecimal); i++ {
					p = p * 10
				}
				dstAmount = new(big.Int).Mul(ev.Amount, new(big.Int).SetUint64(p))
			}
			log.Infof("dstAmount:%s", dstAmount.String())

			// save transfer out
			dbErr := s.db.InsertTransfer(&Transfer{
				TransferId:     ev.TransferId,
				TxHash:         eLog.TxHash,
				ChainId:        bc.chainId.Uint64(),
				Token:          ev.Token,
				TransferType:   cbn.TransferType_TRANSFER_TYPE_OUT,
				TimeLock:       time.Unix(int64(ev.Timelock), 0),
				HashLock:       ev.Hashlock,
				Status:         cbn.TransferStatus_TRANSFER_STATUS_LOCKED,
				RelatedTid:     getTransferId(ev.Receiver, ev.DstAddress, ev.Hashlock, ev.DstChainId),
				RelatedChainId: ev.DstChainId,
				RelatedToken:   dstToken,
				Amount:         *ev.Amount,
				Sender:         ev.Sender,
				Receiver:       ev.Receiver,
				UpdateTs:       time.Now(),
				CreateTs:       time.Now(),
			})
			if dbErr != nil {
				log.Errorf("fail to insert transfer out, ev:%v, err:%v", ev, dbErr)
				return true
			}

			// save transfer in

			chain2TimeLock := time.Now().Add(time.Duration((int64(ev.Timelock)-time.Now().Unix())*2/3) * time.Second)
			if chain2TimeLock.After(time.Unix(int64(ev.Timelock), 0)) {
				log.Errorln("fail to insert transfer out, the chain2 time lock is not valid", ev, chain2TimeLock, time.Unix(int64(ev.Timelock), 0))
				return true
			}
			dbErr = s.db.InsertTransfer(&Transfer{
				TransferId:     getTransferId(ev.Receiver, ev.DstAddress, ev.Hashlock, ev.DstChainId),
				ChainId:        ev.DstChainId,
				Token:          dstToken,
				TransferType:   cbn.TransferType_TRANSFER_TYPE_IN,
				TimeLock:       chain2TimeLock,
				HashLock:       ev.Hashlock,
				Status:         cbn.TransferStatus_TRANSFER_STATUS_START,
				RelatedTid:     ev.TransferId,
				RelatedChainId: bc.chainId.Uint64(),
				RelatedToken:   ev.Token,
				Amount:         *dstAmount,
				Sender:         ev.Receiver,
				Receiver:       ev.DstAddress,
				UpdateTs:       time.Now(),
				CreateTs:       time.Now(),
			})
			if dbErr != nil {
				log.Errorf("fail to insert transfer out, ev:%v, err:%v", ev, dbErr)
				return true
			}

		} else {
			log.Warnf("fail to get transfer out dst chain, transfer out:%v", ev)
		}
		return false
	})
}

func (s *server) monitorLogTransferIn(bc *bridgeConfig) (monitor.CallbackID, error) {
	cfg := &monitor.Config{
		ChainId:    bc.chainId.Uint64(),
		EventName:  evLogTransferIn,
		Contract:   bc.contractChain,
		StartBlock: bc.mon.GetCurrentBlockNumber(),
	}
	return bc.mon.Monitor(cfg, func(id monitor.CallbackID, eLog ethtypes.Log) bool {
		log.Infof("get monitorLogTransferIn, block number:%d", eLog.BlockNumber)
		ev := &contracts.CBridgeLogNewTransferIn{}
		err := bc.contractChain.ParseEvent(evLogTransferIn, eLog, ev)
		if err != nil {
			log.Errorf("monitorLogTransferIn: cannot parse event, err:%v", err)
			return false
		}

		if ev.Sender != s.accountAddr {
			log.Infof("this transfer in sender is not current relay node")
			return false
		}

		transaction, _, err := bc.ec.TransactionByHash(context.Background(), eLog.TxHash)
		if err != nil {
			log.Errorf("monitorLogRefund: cannot get receipt, txHash:%s, err:%v", eLog.TxHash.String(), err)
			return true
		}

		dbErr := s.db.RecordTransferIn(ev.TransferId, eLog.TxHash, transaction.Cost())
		if dbErr != nil {
			log.Errorf("fail to send this transfer in to locked, err:%v", dbErr)
			return true
		}

		return false
	})
}

func (s *server) monitorLogConfirm(bc *bridgeConfig) (monitor.CallbackID, error) {
	cfg := &monitor.Config{
		ChainId:    bc.chainId.Uint64(),
		EventName:  evLogTransferConfirmed,
		Contract:   bc.contractChain,
		StartBlock: bc.mon.GetCurrentBlockNumber(),
	}
	return bc.mon.Monitor(cfg, func(id monitor.CallbackID, eLog ethtypes.Log) bool {
		log.Infof("get monitorLogConfirm, block number:%d", eLog.BlockNumber)
		ev := &contracts.CBridgeLogTransferConfirmed{}
		err := bc.contractChain.ParseEvent(evLogTransferConfirmed, eLog, ev)
		if err != nil {
			log.Errorf("monitorLogConfirm: cannot parse event:%v", err)
			return false
		}

		transaction, _, err := bc.ec.TransactionByHash(context.Background(), eLog.TxHash)
		if err != nil {
			log.Errorf("monitorLogRefund: cannot get receipt, txHash:%s, err:%v", eLog.TxHash.String(), err)
			return true
		}

		dbErr := s.db.ConfirmTransfer(ev.TransferId, ev.Preimage, eLog.TxHash, transaction.Cost())
		if dbErr != nil {
			log.Errorf("fail to update transfer status to confirmed, ev:%v, err:%v", ev, dbErr)
			return true
		}

		dbErr = s.db.SetRelatedTxPreimage(ev.Preimage, ev.TransferId)
		if dbErr != nil {
			log.Errorf("fail to update transfer status to set preimage, ev:%v, err:%v", ev, dbErr)
			return true
		}
		return false
	})
}

// Monitor on-chain user transfer events.
func (s *server) monitorLogRefund(bc *bridgeConfig) (monitor.CallbackID, error) {
	cfg := &monitor.Config{
		ChainId:    bc.chainId.Uint64(),
		EventName:  evLogTransferRefunded,
		Contract:   bc.contractChain,
		StartBlock: bc.mon.GetCurrentBlockNumber(),
	}
	return bc.mon.Monitor(cfg, func(id monitor.CallbackID, eLog ethtypes.Log) bool {
		log.Infof("get monitorLogRefund, block number:%d", eLog.BlockNumber)
		ev := &contracts.CBridgeLogTransferRefunded{}
		err := bc.contractChain.ParseEvent(evLogTransferRefunded, eLog, ev)
		if err != nil {
			log.Errorf("monitorLogRefund: cannot parse event, txHash:%s, err:%v", eLog.TxHash.String(), err)
			return false
		}
		transaction, _, err := bc.ec.TransactionByHash(context.Background(), eLog.TxHash)
		if err != nil {
			log.Errorf("monitorLogRefund: cannot get receipt, txHash:%s, err:%v", eLog.TxHash.String(), err)
			return true
		}
		dbErr := s.db.RefundTransfer(ev.TransferId, eLog.TxHash, transaction.Cost())
		if dbErr != nil {
			log.Errorf("this transfer is refunded by fail to update the status in db, tx:%v, err:%v", ev, dbErr)
			return true
		}
		return false
	})
}

func (s *server) Close() {
	close(s.quit)
	for _, bgc := range s.chainMap {
		if bgc.mon != nil {
			// Close monitor before watch otherwise monitor recreates the watchers.
			// Be nice and wait a bit after monitor close to let it finish its cleanup.
			bgc.mon.Close()
			time.Sleep(2 * time.Second)
			bgc.watch.Close()
			bgc.mon = nil
			bgc.watch = nil
		}
	}
}

func (bc *bridgeConfig) transferIn(dstAddr, token Addr, amount *big.Int, hashLock, transferId, srcTransferId Hash, timeLock, srcChainId, gasGwei, forceGasGwei uint64) error {
	log.Infof("do transferIn, src transferId: %s", transferId.String())
	receipt, err := bc.trans.TransactWaitMined(
		fmt.Sprintf("transferin, transferId:%x", transferId),
		func(ctr bind.ContractTransactor, opts *bind.TransactOpts) (*ethtypes.Transaction, error) {
			cbt, err2 := contracts.NewCBridgeTransactor(bc.contractChain.GetAddr(), ctr)
			if err2 != nil {
				return nil, err2
			}
			return cbt.TransferIn(opts, dstAddr, token, amount, hashLock, timeLock, srcChainId, srcTransferId)
		},
		eth.WithMaxGasGwei(gasGwei),
		eth.WithMinGasGwei(gasGwei),
		eth.WithTimeout(time.Duration(10)*time.Second),
		eth.WithForceGasGwei(forceGasGwei),
	)
	if err != nil {
		log.Errorf("fail to transferIn, transferId:%s, err:%v", transferId.String(), err)
		return err
	}
	log.Infof("success to transferIn, txHash:%x", receipt.TxHash)
	return nil
}

func (bc *bridgeConfig) confirm(transferId, preImage Hash, gasGwei uint64) error {
	log.Infof("do confirm, transfer id:%s", transferId.String())
	receipt, err := bc.trans.TransactWaitMined(
		fmt.Sprintf("confirm with transferId and preImage, transferId:%x", transferId),
		func(ctr bind.ContractTransactor, opts *bind.TransactOpts) (*ethtypes.Transaction, error) {
			cbt, err2 := contracts.NewCBridgeTransactor(bc.contractChain.GetAddr(), ctr)
			if err2 != nil {
				return nil, err2
			}
			return cbt.Confirm(opts, transferId, preImage)
		},
		eth.WithMaxGasGwei(gasGwei),
		eth.WithMinGasGwei(gasGwei),
		eth.WithTimeout(time.Duration(10)*time.Second),
	)
	if err != nil {
		log.Errorf("fail to confirm, transferId:%s, err:%v", transferId.String(), err)
		return err
	}
	log.Infof("success to confirm, txHash:%x", receipt.TxHash)
	return nil
}

func (bc *bridgeConfig) refund(transferId Hash, gasGwei uint64) error {
	receipt, err := bc.trans.TransactWaitMined(
		fmt.Sprintf("refund with transferId, transferId:%x", transferId),
		func(ctr bind.ContractTransactor, opts *bind.TransactOpts) (*ethtypes.Transaction, error) {
			cbt, err2 := contracts.NewCBridgeTransactor(bc.contractChain.GetAddr(), ctr)
			if err2 != nil {
				return nil, err2
			}
			return cbt.Refund(opts, transferId)
		},
		eth.WithMaxGasGwei(gasGwei),
		eth.WithMinGasGwei(gasGwei),
		eth.WithTimeout(time.Duration(10)*time.Second),
	)
	if err != nil {
		log.Infof("fail to refund, transfer id: %s, err:%v", transferId.String(), err)
		return nil
	}
	log.Infof("success to refund, txHash:%x", receipt.TxHash)
	return nil
}

func (bc *bridgeConfig) getTransfer(transferId Hash) (*TransferInfo, error) {
	cbcall, err := contracts.NewCBridgeCaller(bc.contractChain.GetAddr(), bc.ec)
	if err != nil {
		return nil, err
	}

	transfer, err := cbcall.Transfers(&bind.CallOpts{Pending: true}, transferId)
	if err != nil {
		return nil, err
	}
	return &TransferInfo{
		Sender:   transfer.Sender,
		Receiver: transfer.Receiver,
		Token:    transfer.Token,
		Amount:   transfer.Amount,
		HashLock: transfer.Hashlock,
		TimeLock: transfer.Timelock,
		Status:   transfer.Status,
	}, nil
}

type TransferInfo struct {
	Sender   common.Address
	Receiver common.Address
	Token    common.Address
	Amount   *big.Int
	HashLock [32]byte
	TimeLock uint64
	Status   uint8
}

func (s *server) waitMinedOptions(bc *bridgeConfig) []eth.TxOption {
	return []eth.TxOption{
		eth.WithBlockDelay(bc.config.WatchConfig.GetBlockDelay()),
		eth.WithPollingInterval(time.Duration(bc.config.WatchConfig.GetPollingInterval()) * time.Second),
	}
}

func getTransferId(sender, receiver Addr, hashLock Hash, chainId uint64) Hash {
	hash := solsha3.SoliditySHA3(
		// types
		[]string{"address", "address", "bytes32", "uint256"},

		// values
		[]interface{}{
			sender,
			receiver,
			hashLock,
			strconv.FormatUint(chainId, 10),
		},
	)
	return Bytes2Hash(hash)
}

func (s *server) PingCron() {
	ticker := time.NewTicker(time.Second * 30)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			pingErr := s.PingAndRefreshFee()
			if pingErr != nil {
				log.Warnf("fail to get ping info, err:%v", pingErr)
			}
		}
	}
}

func (s *server) PingAndRefreshFee() error {
	sigMsg, err := s.signer.SignEthMessage(s.accountAddr.Bytes())
	if err != nil {
		log.Errorf("fail to sig for ping, err:%v", err)
		return err
	}
	req := &gatewayrpc.PingRequest{
		EthAddr:   s.accountAddr.String(),
		ChainInfo: []*gatewayrpc.ChainInfo{},
		NickName:  s.cfg.GetRelayNodeName(),
		Sig:       sigMsg,
	}
	for k, v := range s.chainMap {
		chainInfo := &gatewayrpc.ChainInfo{
			ChainId:         k,
			TokenAndBalance: map[string]string{},
			FeePer_10000:    v.config.FeeRate,
		}
		for addr, erc20 := range v.erc20Map {
			balance, balanceErr := erc20.BalanceOf(nil, s.accountAddr)
			if balanceErr != nil {
				log.Warnf("fail to get this token balance, skip it, chain id:%d, token addr:%s, err:%s", k, addr, balanceErr.Error())
				continue
			}
			chainInfo.TokenAndBalance[addr.String()] = balance.String()
		}
		req.ChainInfo = append(req.ChainInfo, chainInfo)
	}
	resp, pingErr := s.gateway.PingGateway(req)
	if pingErr != nil {
		return pingErr
	}
	if beErr := resp.GetErr(); beErr != nil {
		return fmt.Errorf("gateway err:%v", beErr)
	}

	s.setGatewayChainInfo(resp.GetChainInfo())

	//log.Infof("ping resp:%v", resp)
	return nil
}

func (s *server) setGatewayChainInfo(gatewayChainInfoMap map[uint64]*gatewayrpc.GatewayChainInfo) {
	s.gatewayChainInfoMapLock.Lock()
	defer s.gatewayChainInfoMapLock.Unlock()
	s.gatewayChainInfoMap = gatewayChainInfoMap
}

func (s *server) ProcessTransfers() {
	ticker := time.NewTicker(time.Second * 30)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.processTrySendTransferIn()
			s.processTryConfirmTransferIn()
			s.processTryRefundTransferIn()
			s.processTryConfirmTransferOut()
		}
	}
}

func (s *server) processTrySendTransferIn() {
	startedTransferIn, dbErr := s.db.GetAllStartTransferIn()
	if dbErr != nil {
		log.Warnf("fail to query waiting for sending transfer in, err:%s", dbErr)
		return
	}
	for _, tx := range startedTransferIn {
		bc, foundBc := s.chainMap[tx.ChainId]
		if foundBc {
			if tx.TimeLock.Add(time.Duration(-1*int64(20)) * time.Second).Before(time.Now()) {
				log.Warnf("this transfer out is already timeout, transferId:%s, timeLock: %s",
					tx.TransferId.String(), tx.TimeLock.String())
				continue
			}

			remoteTransferIn, err := bc.getTransfer(tx.TransferId)
			if err != nil {
				log.Errorf("fail to get transfer in, txid:%s, err:%v", tx.TransferId.String(), err)
				continue
			}
			if remoteTransferIn.Status != 0 {
				log.Warnf("this transfer in already exist, transfer:%v", remoteTransferIn)
				continue
			}

			// if the old fee is recorded, may caused by try send transfer in failed
			// then we add it back first.
			originAmt := new(big.Int).Add(&tx.Amount, &tx.Fee)

			gasGwei, getGweiErr := s.getGasPrice(bc.chainId.Uint64())
			if getGweiErr != nil {
				log.Warnf("fail to get gas gwei for this transferIn, transfer:%v, err:%v", remoteTransferIn, getGweiErr)
				continue
			}
			if gasGwei <= 0 {
				log.Warnf("fail to find gas gwei for this transferIn, transfer:%v", remoteTransferIn)
				continue
			}

			finalFee, getFeeErr := s.gateway.GetFee(tx.RelatedTid)
			if getFeeErr != nil {
				log.Errorf("can not get the fee for this transfer, transferOutId:%s, err:%v", tx.RelatedTid.String(), getFeeErr)
				continue
			}

			log.Infof("tx:%s, final fee:%s", tx.RelatedTid.String(), finalFee.String())

			if finalFee.Cmp(originAmt) > 0 {
				log.Errorf("this fee is bigger than amount, transferOutId:%s, fee:%s, origin amount:%s, err:%v",
					tx.RelatedTid.String(), finalFee.String(), originAmt.String(), getFeeErr)
				continue
			}

			newAmount := new(big.Int).Sub(originAmt, finalFee)

			setTransferInAmountAndFeeErr := s.db.SetTransferInAmountAndFee(tx.TransferId, newAmount, finalFee)
			if setTransferInAmountAndFeeErr != nil {
				log.Errorf("fail to set transferIn fee and amount, ev:%v, err:%v", tx, setTransferInAmountAndFeeErr)
				continue
			}

			sendTransferInErr := bc.transferIn(tx.Receiver, tx.Token, newAmount,
				tx.HashLock, tx.TransferId, tx.RelatedTid, uint64(tx.TimeLock.Unix()), bc.chainId.Uint64(), gasGwei, bc.config.GetForceGasGwei())
			if sendTransferInErr != nil {
				log.Errorf("fail to transferIn, ev:%v, err:%v", tx, sendTransferInErr)
				continue
			}

			setDbTransferToPendingErr := s.db.SetPendingTransferIn(tx.TransferId, cbn.TransferStatus_TRANSFER_STATUS_PENDING,
				cbn.TransferStatus_TRANSFER_STATUS_START)
			if setDbTransferToPendingErr != nil {
				log.Errorf("fail to set transferIn to pending, ev:%v, err:%v", tx, setDbTransferToPendingErr)
				continue
			}

		}
	}
}

func (s *server) processTryConfirmTransferIn() {
	lockedTransferIn, dbErr := s.db.GetAllLockedTransferIn()
	if dbErr != nil {
		log.Warnf("fail to query refund able transfers, err:%s", dbErr)
		return
	}
	for _, tx := range lockedTransferIn {
		transferOut, foundTx, getTxOutDbErr := s.db.GetTransferByTid(tx.RelatedTid)
		if getTxOutDbErr != nil {
			log.Warnf("fail to get transfer out by transfer in, tx.RelatedTid:%s, err:%v", tx.RelatedTid.String(), getTxOutDbErr)
			continue
		}
		if !foundTx {
			log.Warnf("fail to found transfer out by transfer in, tx.RelatedTid:%s", tx.RelatedTid.String())
			continue
		}
		if transferOut.Status != cbn.TransferStatus_TRANSFER_STATUS_CONFIRMED {
			continue
		}

		dstBcg, foundBcg := s.chainMap[tx.ChainId]
		if foundBcg {
			remoteTransferIn, err := dstBcg.getTransfer(tx.TransferId)
			if err != nil {
				log.Errorf("fail to get transfer in, txid:%s, err:%v", tx.TransferId.String(), err)
				continue
			}
			log.Infof("get remote confirmable transfer in:%v", remoteTransferIn)
			if remoteTransferIn.Status == remoteTransferStatusPending {
				gasGwei, getGweiErr := s.getGasPrice(dstBcg.chainId.Uint64())
				if getGweiErr != nil {
					log.Warnf("fail to get gas gwei for this confirm, transfer:%v, err:%v", remoteTransferIn, getGweiErr)
					continue
				}
				if gasGwei <= 0 {
					log.Warnf("fail to find gas gwei for this confirm, transfer:%v", remoteTransferIn)
					continue
				}

				err = dstBcg.confirm(tx.TransferId, transferOut.Preimage, gasGwei)
				if err != nil {
					log.Errorf("fail to confirm related transfer, ev:%v, err:%v", tx, err)
					continue
				}
			} else if remoteTransferIn.Status == remoteTransferStatusRefunded {
				dbErr = s.db.UpdateTransferStatus(tx.TransferId, cbn.TransferStatus_TRANSFER_STATUS_REFUNDED)
				if dbErr != nil {
					log.Errorf("this transfer is refunded by fail to update the status in db, tx:%v, err:%v", tx, dbErr)
				}
			} else if remoteTransferIn.Status == remoteTransferStatusConfirmed {
				dbErr = s.db.ConfirmTransfer(tx.TransferId, transferOut.Preimage, Hash{}, big.NewInt(0))
				if dbErr != nil {
					log.Errorf("fail to update transfer status to confirmed, tx:%v, err:%v", tx, dbErr)
				}
			} else {
				log.Warnf("this transfer in status is invalid, transfer:%v", remoteTransferIn)
			}
		}
	}
}

func (s *server) processTryConfirmTransferOut() {
	lockedTransferOut, dbErr := s.db.GetAllConfirmableTransferOut()
	if dbErr != nil {
		log.Warnf("fail to query confirmable transfer out, err:%s", dbErr)
		return
	}
	for _, tx := range lockedTransferOut {
		dstBcg, foundBcg := s.chainMap[tx.ChainId]
		if foundBcg {
			remoteTransferOut, err := dstBcg.getTransfer(tx.TransferId)
			if err != nil {
				log.Errorf("fail to get transfer out, txid:%s, err:%v", tx.TransferId.String(), err)
				continue
			}
			log.Infof("get remote confirmable transfer in:%v", remoteTransferOut)
			if remoteTransferOut.Status == remoteTransferStatusPending {
				gasGwei, getGweiErr := s.getGasPrice(dstBcg.chainId.Uint64())
				if getGweiErr != nil {
					log.Warnf("fail to get gas gwei for this confirm, transfer:%v, err:%v", tx, getGweiErr)
					continue
				}
				if gasGwei <= 0 {
					log.Warnf("fail to find gas gwei for this confirm, transfer:%v", tx)
					continue
				}
				err = dstBcg.confirm(tx.TransferId, tx.Preimage, gasGwei)
				if err != nil {
					log.Errorf("fail to confirm related transfer, ev:%v, err:%v", tx, err)
					continue
				}
			} else if remoteTransferOut.Status == remoteTransferStatusRefunded {
				dbErr = s.db.UpdateTransferStatus(tx.TransferId, cbn.TransferStatus_TRANSFER_STATUS_REFUNDED)
				if dbErr != nil {
					log.Errorf("this transfer is refunded by fail to update the status in db, tx:%v, err:%v", tx, dbErr)
				}
			} else if remoteTransferOut.Status == remoteTransferStatusConfirmed {
				dbErr = s.db.ConfirmTransfer(tx.TransferId, tx.Preimage, Hash{}, big.NewInt(0))
				if dbErr != nil {
					log.Errorf("fail to update transfer status to confirmed, tx:%v, err:%v", tx, dbErr)
				}
			} else {
				log.Warnf("this transfer in status is invalid, transfer:%v", tx)
			}
		}
	}
}

func (s *server) processTryRefundTransferIn() {
	// do refund
	refundableTransferIn, dbErr := s.db.GetAllRefundAbleTransferIn()
	if dbErr != nil {
		log.Warnf("fail to query refund able transfers, err:%s", dbErr)
		return
	}
	for _, tx := range refundableTransferIn {
		bc, foundBc := s.chainMap[tx.ChainId]
		if foundBc {
			remoteTransferIn, err := bc.getTransfer(tx.TransferId)
			if err != nil {
				log.Errorf("fail to get transfer in, txid:%s, err:%v", tx.TransferId.String(), err)
				continue
			}
			log.Infof("get remote refundable transfer in:%v", remoteTransferIn)
			if remoteTransferIn.Status == remoteTransferStatusPending {
				gasGwei, getGweiErr := s.getGasPrice(bc.chainId.Uint64())
				if getGweiErr != nil {
					log.Warnf("fail to get gas gwei for this refund, transfer:%v, err:%v", remoteTransferIn, getGweiErr)
					continue
				}
				if gasGwei <= 0 {
					log.Warnf("fail to find gas gwei for this refund, transfer:%v", remoteTransferIn)
					continue
				}
				err = bc.refund(tx.TransferId, gasGwei)
				if err != nil {
					log.Errorf("fail to refund this tx: %v", tx)
					continue
				}
			} else if remoteTransferIn.Status == remoteTransferStatusRefunded {
				dbErr = s.db.UpdateTransferStatus(tx.TransferId, cbn.TransferStatus_TRANSFER_STATUS_REFUNDED)
				if dbErr != nil {
					log.Errorf("this transfer is refunded by fail to update the status in db, tx:%v, err:%v", tx, dbErr)
				}
			} else {
				log.Warnf("this transfer in status is invalid, transfer:%v", remoteTransferIn)
			}

		}
	}
}

func (s *server) getGasPrice(chainId uint64) (uint64, error) {
	s.gatewayChainInfoMapLock.Lock()
	defer s.gatewayChainInfoMapLock.Unlock()
	gatewayChainInfo, foundFee := s.gatewayChainInfoMap[chainId]
	if !foundFee {
		return 0, InvalidGasPriceChain
	}
	return gatewayChainInfo.GetGasPrice() / 1e9, nil
}

func (s *server) GetTotalSummary(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	log.Infof("GetTotalSummary")
	perChain2ChainSummary := make(map[string]*Chain2ChainBreakDownDetail)

	transfers, dbErr := s.db.GetAllTransfers()
	if dbErr != nil {
		return
	}

	for _, tx := range transfers {
		if tx.TransferType == cbn.TransferType_TRANSFER_TYPE_OUT {
			s.recordTransferOutSummary(tx, perChain2ChainSummary)
		} else if tx.TransferType == cbn.TransferType_TRANSFER_TYPE_IN {
			s.recordTransferInSummary(tx, perChain2ChainSummary)
		}
	}

	log.Infof("finished")

	content := []string{"------------------------------------------------"}
	for _, v := range perChain2ChainSummary {
		content = append(content, fmt.Sprintf("chain %d -> chain %d", v.SrcChainId, v.DstChainId))
		content = append(content, fmt.Sprintf("Received %d transfers", v.TotalTransferOutNumber))
		content = append(content, fmt.Sprintf("Successfully processed %d transfers", v.TotalSuccessTransferInNumber))
		var successRate float64
		if v.TotalTransferOutNumber != 0 {
			successRate = float64(v.TotalSuccessTransferInNumber*100) / float64(v.TotalTransferOutNumber)
		}
		content = append(content, fmt.Sprintf("Success rate: %.2f%s", successRate, "%"))
		for _, tokenFee := range v.FeeReceived {
			tokenVolumeFormat := new(big.Float).Mul(big.NewFloat(0).SetInt(tokenFee.TotalVolume), big.NewFloat(1/math.Pow10(int(tokenFee.TokenDecimal))))
			tokenEarnFormat := new(big.Float).Mul(big.NewFloat(0).SetInt(tokenFee.FeeAmount), big.NewFloat(1/math.Pow10(int(tokenFee.TokenDecimal))))
			content = append(content, fmt.Sprintf("Token name: %s, transfer volume:%s %s, earned fee:%s %s", tokenFee.TokenName, tokenVolumeFormat.String(), tokenFee.TokenName, tokenEarnFormat.String(), tokenFee.TokenName))
		}
		content = append(content, fmt.Sprintf("------------------------------------------------"))
	}
	content = append(content, fmt.Sprintf(""))

	resp := strings.Join(content, "\n")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	_, err := w.Write([]byte(resp))
	if err != nil {
		log.Errorf("write response err: %v", err)
		http.Error(w, "write response failed", http.StatusExpectationFailed)
		return
	}
}

func (s *server) GetTransfer(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	log.Infof("GetTransfer")
	limitString := ps.ByName("limit")
	limit, err := strconv.ParseUint(limitString, 10, 64)
	if err != nil {
		http.Error(w, "invalid limit param", http.StatusBadRequest)
		return
	}
	if limit <= 0 {
		limit = 500
	}
	transfers, dbErr := s.db.GetAllTransfersWithLimit(limit)
	if dbErr != nil {
		http.Error(w, fmt.Sprintf("db err happened, err:%v", dbErr), http.StatusBadRequest)
		return
	}
	content := []string{}
	for _, tx := range transfers {
		createTs := tx.CreateTs.String()
		txType := tx.TransferType.String()
		txStatus := tx.Status.String()
		volume := new(big.Int).Add(&tx.Amount, &tx.Fee).String()
		srcChain := tx.ChainId
		dstChain := tx.RelatedChainId
		if tx.TransferType == cbn.TransferType_TRANSFER_TYPE_IN {
			dstChain = tx.ChainId
			srcChain = tx.RelatedChainId
		}
		content = append(content, fmt.Sprintf("userAddr:%s, chain %d -> chain %d ,transferAmount:%s, createTs:%s, type:%s, status:%s",
			tx.Sender.String(), srcChain, dstChain, volume, createTs, txType, txStatus))
	}
	resp := strings.Join(content, "\n")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	_, err = w.Write([]byte(resp))
	if err != nil {
		log.Errorf("write response err: %v", err)
		http.Error(w, "write response failed", http.StatusExpectationFailed)
		return
	}
}

func (s *server) recordTransferInSummary(transferIn *Transfer, perChain2ChainSummary map[string]*Chain2ChainBreakDownDetail) {
	if transferIn.Status == cbn.TransferStatus_TRANSFER_STATUS_LOCKED ||
		transferIn.Status == cbn.TransferStatus_TRANSFER_STATUS_REFUNDED ||
		transferIn.Status == cbn.TransferStatus_TRANSFER_STATUS_CONFIRMED {
		key := generateChain2ChainKey(transferIn.RelatedChainId, transferIn.ChainId)
		chain2ChainSummary, foundChain2ChainSummary := perChain2ChainSummary[key]
		if !foundChain2ChainSummary {
			gasTokenInfo := s.getGasTokenInfo(transferIn.ChainId)
			chain2ChainSummary = &Chain2ChainBreakDownDetail{
				SrcChainId:                   transferIn.RelatedChainId,
				DstChainId:                   transferIn.ChainId,
				TotalTransferOutNumber:       0,
				TotalSuccessTransferInNumber: 0,
				FeeReceived:                  make(map[Addr]*TokenFeeSummary),
				GasCost:                      big.NewInt(0),
				GasTokenName:                 gasTokenInfo.GasTokenName,
				GasDecimal:                   gasTokenInfo.GasTokenDecimal,
			}
			perChain2ChainSummary[key] = chain2ChainSummary
		}

		switch transferIn.Status {
		case cbn.TransferStatus_TRANSFER_STATUS_LOCKED:
			chain2ChainSummary.GasCost = new(big.Int).Add(chain2ChainSummary.GasCost, &transferIn.TransferGasCost)
		case cbn.TransferStatus_TRANSFER_STATUS_REFUNDED:
			chain2ChainSummary.GasCost = new(big.Int).Add(chain2ChainSummary.GasCost, &transferIn.TransferGasCost)
			chain2ChainSummary.GasCost = new(big.Int).Add(chain2ChainSummary.GasCost, &transferIn.RefundGasCost)
		case cbn.TransferStatus_TRANSFER_STATUS_CONFIRMED:
			chain2ChainSummary.GasCost = new(big.Int).Add(chain2ChainSummary.GasCost, &transferIn.TransferGasCost)
			chain2ChainSummary.GasCost = new(big.Int).Add(chain2ChainSummary.GasCost, &transferIn.ConfirmGasCost)
			chain2ChainSummary.TotalSuccessTransferInNumber++
			chain2ChainSummaryFee, foundChain2ChainSummaryFee := chain2ChainSummary.FeeReceived[transferIn.Token]
			if !foundChain2ChainSummaryFee {
				chain2ChainSummaryFee = &TokenFeeSummary{
					TokenName:    s.getTokenName(transferIn.Token),
					TokenAddr:    transferIn.Token,
					TotalVolume:  new(big.Int),
					FeeAmount:    new(big.Int),
					TokenDecimal: s.getTokenDecimal(transferIn.Token),
				}
				chain2ChainSummary.FeeReceived[transferIn.Token] = chain2ChainSummaryFee
			}
			chain2ChainSummaryFee.TotalVolume = new(big.Int).Add(chain2ChainSummaryFee.TotalVolume, &transferIn.Amount)
			chain2ChainSummaryFee.FeeAmount = new(big.Int).Add(chain2ChainSummaryFee.FeeAmount, &transferIn.Fee)
		}
	}
}

func (s *server) recordTransferOutSummary(transferOut *Transfer, perChain2ChainSummary map[string]*Chain2ChainBreakDownDetail) {
	key := generateChain2ChainKey(transferOut.ChainId, transferOut.RelatedChainId)
	chain2ChainSummary, foundChain2ChainSummary := perChain2ChainSummary[key]
	if !foundChain2ChainSummary {
		chain2ChainSummary = &Chain2ChainBreakDownDetail{
			SrcChainId:                   transferOut.ChainId,
			DstChainId:                   transferOut.RelatedChainId,
			TotalTransferOutNumber:       0,
			TotalSuccessTransferInNumber: 0,
			FeeReceived:                  make(map[Addr]*TokenFeeSummary),
			GasCost:                      big.NewInt(0),
		}
		perChain2ChainSummary[key] = chain2ChainSummary
	}
	chain2ChainSummary.TotalTransferOutNumber++
}

func (s *server) getTokenName(tokenAddr Addr) string {
	name, foundName := s.chainTokenNameMap[tokenAddr]
	if !foundName {
		name = tokenAddr.String()
	}
	return name
}

func (s *server) getTokenDecimal(tokenAddr Addr) uint64 {
	decimal, found := s.chainTokenDecimalMap[tokenAddr]
	if !found {
		decimal = 1
	}
	return decimal
}

func (s *server) getGasTokenInfo(chainId uint64) *chainGasTokenInfo {
	info, found := s.chainGasTokenMap[chainId]
	if !found {
		return &chainGasTokenInfo{
			GasTokenName:    "unknown",
			GasTokenDecimal: 1,
		}
	}
	return info
}

func generateChain2ChainKey(srcChainId, dstChainId uint64) string {
	return fmt.Sprintf("%d->%d", srcChainId, dstChainId)
}

type Chain2ChainBreakDownDetail struct {
	SrcChainId                   uint64
	DstChainId                   uint64
	TotalTransferOutNumber       uint64
	TotalSuccessTransferInNumber uint64
	FeeReceived                  map[Addr]*TokenFeeSummary
	GasCost                      *big.Int
	GasDecimal                   uint64
	GasTokenName                 string
}

type TokenFeeSummary struct {
	TokenName    string
	TokenAddr    Addr
	FeeAmount    *big.Int
	TotalVolume  *big.Int
	TokenDecimal uint64
}
