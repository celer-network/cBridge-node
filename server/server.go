package server

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

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
	"github.com/julienschmidt/httprouter"
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

	transactorWaitTimeout = 5 * time.Minute
)

type server struct {
	version      string
	cfg          *cbn.CBridgeConfig // config from local json file
	chainMap     map[uint64]*bridgeConfig
	chainMapLock sync.Mutex
	accountAddr  Addr
	db           *DAL
	gateway      GatewayAPI
	signer       *eth.CelerSigner // sign req msg

	gatewayChainInfoMap map[uint64]*gatewayrpc.GatewayChainInfo

	//<fromToken, <toChainId, toChainToken>>
	// TODO should add chain id to this map in future, otherwise there may be bug here.
	chainTokenNameMap    map[uint64]map[Addr]string
	chainTokenAddrMap    map[uint64]map[string]Addr
	chainTokenDecimalMap map[uint64]map[Addr]uint64
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

func NewServer(version string) *server {
	return &server{
		version:              version,
		chainMap:             make(map[uint64]*bridgeConfig),
		gatewayChainInfoMap:  make(map[uint64]*gatewayrpc.GatewayChainInfo),
		chainTokenNameMap:    make(map[uint64]map[Addr]string),
		chainTokenAddrMap:    make(map[uint64]map[string]Addr),
		chainTokenDecimalMap: make(map[uint64]map[Addr]uint64),
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

func (s *server) Init(config *cbn.CBridgeConfig, ks, pwdDir string) error {
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
	tcfg, err := GetTransactorConfig(ks, pwdDir)
	if err != nil {
		return err
	}
	s.accountAddr, _, err = eth.GetAddrPrivKeyFromKeystore(tcfg.Keyjson, tcfg.Passphrase)
	if err != nil {
		log.Errorf("fail to get relay node wallet from key store, err:%v", err)
		return err
	}
	s.signer, err = eth.NewSignerFromKeystore(tcfg.Keyjson, tcfg.Passphrase, big.NewInt(0))
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
			s.transactorOptions(bgc)...,
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
			subChainTokenNameMap, foundSubChainTokenNameMap := s.chainTokenNameMap[chainConfig.GetChainId()]
			if foundSubChainTokenNameMap {
				subChainTokenNameMap[Hex2Addr(tokenConfig.TokenAddress)] = tokenConfig.GetTokenName()
			} else {
				s.chainTokenNameMap[chainConfig.GetChainId()] = make(map[Addr]string)
				s.chainTokenNameMap[chainConfig.GetChainId()][Hex2Addr(tokenConfig.TokenAddress)] = tokenConfig.GetTokenName()
			}

			subChainTokenDecimalMap, foundSubChainTokenDecimalMap := s.chainTokenDecimalMap[chainConfig.GetChainId()]
			if foundSubChainTokenDecimalMap {
				subChainTokenDecimalMap[Hex2Addr(tokenConfig.TokenAddress)] = tokenConfig.GetTokenDecimal()
			} else {
				s.chainTokenDecimalMap[chainConfig.GetChainId()] = make(map[Addr]uint64)
				s.chainTokenDecimalMap[chainConfig.GetChainId()][Hex2Addr(tokenConfig.TokenAddress)] = tokenConfig.GetTokenDecimal()
			}

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

			ksfAc, err := os.Open(ks)
			if err != nil {
				return err
			}
			authAccount, err := bind.NewTransactorWithChainID(ksfAc, tcfg.Passphrase, bgc.chainId)
			if err != nil {
				return err
			}

			curAllowance, err := bgc.erc20Map[Hex2Addr(tokenConfig.GetTokenAddress())].Allowance(&bind.CallOpts{}, s.accountAddr, bgc.contractChain.GetAddr())
			if err != nil {
				return err
			}

			if curAllowance.Cmp(new(big.Int).Div(MaxUint256, big.NewInt(2))) < 0 {
				log.Infof("Approving token %s on chain %d...", tokenConfig.GetTokenName(), chainConfig.GetChainId())
				approveTx, approveErr := bgc.erc20Map[Hex2Addr(tokenConfig.GetTokenAddress())].Approve(authAccount, bgc.contractChain.GetAddr(), MaxUint256)
				if approveErr != nil {
					log.Errorf("please try again, can not approve token %s on chain %d, err:%v", tokenConfig.GetTokenName(), chainConfig.GetChainId(), approveErr)
					return approveErr
				}

				approveReceipt, approveReceiptErr := eth.WaitMinedWithTxHash(context.Background(),
					bgc.ec,
					approveTx.Hash().String(),
					eth.WithTimeout(2*time.Minute),
					eth.WithBlockDelay(2))

				if approveReceiptErr != nil {
					log.Errorf("please try again, can not approve token %s on chain %d, err:%v", tokenConfig.GetTokenName(), chainConfig.GetChainId(), approveReceiptErr)
					return approveReceiptErr
				}
				log.Infof("success to approve token:%s on chain:%d, receiptTxHash:%x, blockNum:%s, gasUsed:%d",
					tokenConfig.GetTokenName(), chainConfig.GetChainId(), approveReceipt.TxHash, approveReceipt.BlockNumber.String(), approveReceipt.GasUsed)
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
		bgc.watch = watcher.NewWatchService(bgc.ec, s.db, bgc.config.WatchConfig.GetPollingInterval(), bgc.config.WatchConfig.GetMaxBlockDelta())
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
		ChainId:      bc.chainId.Uint64(),
		EventName:    evLogTransferOut,
		Contract:     bc.contractChain,
		StartBlock:   bc.mon.GetCurrentBlockNumber(),
		ForwardDelay: bc.config.GetWatchConfig().GetForwardBlockDelay(),
	}
	return bc.mon.Monitor(cfg, func(id monitor.CallbackID, eLog ethtypes.Log) bool {
		ev := &contracts.CBridgeLogNewTransferOut{}
		err := bc.contractChain.ParseEvent(evLogTransferOut, eLog, ev)
		if err != nil {
			log.Errorf("monitorLogTransferOut: cannot parse event, chainId:%d, txHash:%x, err:%v", bc.chainId.Uint64(), eLog.TxHash, err)
			return false
		}
		if ev.Receiver != s.accountAddr {
			log.Infof("this transfer out receiver is not current relay node and skip it")
			return false
		}
		log.Infof("get monitorLogTransferOut, chain id:%d, block number:%d, transfer id:%x, eLog txHash:%x", bc.chainId.Uint64(), eLog.BlockNumber, ev.TransferId, eLog.TxHash)

		_, found := s.chainMap[ev.DstChainId]
		if found {
			tsNow := time.Now()
			tokenName, foundTokenName := s.chainTokenNameMap[bc.chainId.Uint64()][ev.Token]
			if !foundTokenName {
				log.Warnf("fail to get this token name, transferId:%x, token:%s", ev.TransferId, ev.Token.String())
				return false
			}

			dstChainTokenMap, foundDisChainTokenMap := s.chainTokenAddrMap[ev.DstChainId]
			if !foundDisChainTokenMap {
				log.Warnf("fail to get this dst chain, transferId:%x, dst chainId:%d", ev.TransferId, ev.DstChainId)
				return false
			}
			dstToken, foundDstToken := dstChainTokenMap[tokenName]
			if !foundDstToken {
				log.Warnf("fail to get this dst token, transferId:%x, tokenName:%s", ev.TransferId, tokenName)
				return false
			}

			srcTokenDecimal, foundSrcTokenDecimal := s.chainTokenDecimalMap[bc.chainId.Uint64()][ev.Token]
			if !foundSrcTokenDecimal {
				log.Warnf("fail to get this src token decimal, transferId:%x, token:%s", ev.TransferId, ev.Token.String())
				return false
			}

			dstTokenDecimal, foundDstTokenDecimal := s.chainTokenDecimalMap[ev.DstChainId][dstToken]
			if !foundDstTokenDecimal {
				log.Warnf("fail to get this dst token decimal, transferId:%x, token:%s", ev.TransferId, dstToken.String())
				return false
			}

			dstAmount := ev.Amount
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
			log.Infof("transferOutId:%x, srcAmount:%s, srcTokenDecimal:%d, dstTokenDecimal:%d, dstAmt:%s", ev.TransferId, ev.Amount.String(), srcTokenDecimal, dstTokenDecimal, dstAmount.String())
			// save transfer out
			log.Infof("save transfer out, transferOutId:%x", ev.TransferId)
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
				UpdateTs:       tsNow,
				CreateTs:       tsNow,
			})
			if dbErr != nil {
				log.Errorf("fail to insert transfer out, should try again, ev:%v, err:%v", ev, dbErr)
				return true
			}

			timeout := int64(ev.Timelock) - tsNow.Unix()
			if timeout < 3600*16 { // src timeout should be larger than 16 hours
				log.Errorf("src transfer out timeout too small: %d sec", timeout)
				return false
			}
			// dst timeout should be 2 hour smaller than src
			chain2TimeLock := tsNow.Add(time.Duration(timeout-(8*3600)) * time.Second)
			// save transfer in
			transferInId := getTransferId(ev.Receiver, ev.DstAddress, ev.Hashlock, ev.DstChainId)
			log.Infof("save transfer in, transferInId:%x", transferInId)
			dbErr = s.db.InsertTransfer(&Transfer{
				TransferId:     transferInId,
				ChainId:        ev.DstChainId,
				Token:          dstToken,
				TransferType:   cbn.TransferType_TRANSFER_TYPE_IN,
				TimeLock:       chain2TimeLock,
				HashLock:       ev.Hashlock,
				Status:         cbn.TransferStatus_TRANSFER_STATUS_TRANSFER_IN_START,
				RelatedTid:     ev.TransferId,
				RelatedChainId: bc.chainId.Uint64(),
				RelatedToken:   ev.Token,
				Amount:         *dstAmount,
				Sender:         ev.Receiver,
				Receiver:       ev.DstAddress,
				UpdateTs:       tsNow,
				CreateTs:       tsNow,
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
		ChainId:      bc.chainId.Uint64(),
		EventName:    evLogTransferIn,
		Contract:     bc.contractChain,
		StartBlock:   bc.mon.GetCurrentBlockNumber(),
		ForwardDelay: bc.config.GetWatchConfig().GetForwardBlockDelay(),
	}
	return bc.mon.Monitor(cfg, func(id monitor.CallbackID, eLog ethtypes.Log) bool {
		ev := &contracts.CBridgeLogNewTransferIn{}
		err := bc.contractChain.ParseEvent(evLogTransferIn, eLog, ev)
		if err != nil {
			log.Errorf("monitorLogTransferIn: cannot parse event, chainId:%d, txHash:%x, err:%v", bc.chainId.Uint64(), eLog.TxHash, err)
			return false
		}

		if ev.Sender != s.accountAddr {
			log.Infof("this transfer in sender is not current relay node")
			return false
		}
		log.Infof("get monitorLogTransferIn, chain id:%d, block number:%d, transfer id:%x, eLog txHash:%x", bc.chainId.Uint64(), eLog.BlockNumber, ev.TransferId, eLog.TxHash)
		dbErr := s.db.RecordTransferIn(ev.TransferId, eLog.TxHash)
		if dbErr != nil {
			log.Errorf("fail to send this transfer in to locked, transferId:%x, err:%v", ev.TransferId, dbErr)
			return true
		}

		return false
	})
}

func (s *server) monitorLogConfirm(bc *bridgeConfig) (monitor.CallbackID, error) {
	cfg := &monitor.Config{
		ChainId:      bc.chainId.Uint64(),
		EventName:    evLogTransferConfirmed,
		Contract:     bc.contractChain,
		StartBlock:   bc.mon.GetCurrentBlockNumber(),
		ForwardDelay: bc.config.GetWatchConfig().GetForwardBlockDelay(),
	}
	return bc.mon.Monitor(cfg, func(id monitor.CallbackID, eLog ethtypes.Log) bool {
		ev := &contracts.CBridgeLogTransferConfirmed{}
		err := bc.contractChain.ParseEvent(evLogTransferConfirmed, eLog, ev)
		if err != nil {
			log.Errorf("monitorLogTransferConfirm: cannot parse event, chainId:%d, txHash:%x, err:%v", bc.chainId.Uint64(), eLog.TxHash, err)
			return false
		}
		log.Infof("get monitorLogConfirm, chain id:%d, block number:%d, transfer id:%x, eLog txHash:%x, preimage:%x", bc.chainId.Uint64(), eLog.BlockNumber, ev.TransferId, eLog.TxHash, ev.Preimage)
		dbErr := s.db.ConfirmTransfer(ev.TransferId, ev.Preimage, eLog.TxHash)
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
		ChainId:      bc.chainId.Uint64(),
		EventName:    evLogTransferRefunded,
		Contract:     bc.contractChain,
		StartBlock:   bc.mon.GetCurrentBlockNumber(),
		ForwardDelay: bc.config.GetWatchConfig().GetForwardBlockDelay(),
	}
	return bc.mon.Monitor(cfg, func(id monitor.CallbackID, eLog ethtypes.Log) bool {
		ev := &contracts.CBridgeLogTransferRefunded{}
		err := bc.contractChain.ParseEvent(evLogTransferRefunded, eLog, ev)
		if err != nil {
			log.Errorf("monitorLogTransferRefund: cannot parse event, chainId:%d, txHash:%x, err:%v", bc.chainId.Uint64(), eLog.TxHash, err)
			return false
		}
		log.Infof("get monitorLogRefund, chain id:%d, block number:%d, transfer id:%x, eLog txHash:%x", bc.chainId.Uint64(), eLog.BlockNumber, ev.TransferId, eLog.TxHash)
		dbErr := s.db.RefundTransfer(ev.TransferId, eLog.TxHash)
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

func (bc *bridgeConfig) transferIn(dstAddr, token Addr, amount *big.Int, hashLock, transferId, srcTransferId Hash, timeLock, srcChainId uint64) error {
	log.Infof("start transfer in, transferId:%x, chainId:%d, srcTransferId:%x, hashLock:%x", transferId, bc.chainId.Uint64(), srcTransferId, hashLock)
	_, err := bc.trans.Transact(
		logTransactionStateHandler(fmt.Sprintf("receipt transferIn, transferId: %x, chainId: %s", transferId, bc.chainId)),
		func(ctr bind.ContractTransactor, opts *bind.TransactOpts) (*ethtypes.Transaction, error) {
			cbt, err2 := contracts.NewCBridgeTransactor(bc.contractChain.GetAddr(), ctr)
			if err2 != nil {
				return nil, err2
			}
			return cbt.TransferIn(opts, dstAddr, token, amount, hashLock, timeLock, srcChainId, srcTransferId)
		},
	)
	return err
}

func (bc *bridgeConfig) confirm(transferId, srcTransferId, preImage, hashLock Hash) error {
	log.Infof("start confirm, transferId:%x, chainId:%d, srcTransferId:%x, hashLock:%x", transferId, bc.chainId.Uint64(), srcTransferId, hashLock)
	_, err := bc.trans.Transact(
		logTransactionStateHandler(fmt.Sprintf("receipt confirm, transferId: %x, chainId: %s", transferId, bc.chainId)),
		func(ctr bind.ContractTransactor, opts *bind.TransactOpts) (*ethtypes.Transaction, error) {
			cbt, err2 := contracts.NewCBridgeTransactor(bc.contractChain.GetAddr(), ctr)
			if err2 != nil {
				return nil, err2
			}
			return cbt.Confirm(opts, transferId, preImage)
		},
	)
	return err
}

func (bc *bridgeConfig) refund(transferId, srcTransferId, hashLock Hash) error {
	log.Infof("start refund, transferId:%x, chainId:%d, srcTransferId:%x, hashLock:%x", transferId, bc.chainId.Uint64(), srcTransferId, hashLock)
	_, err := bc.trans.Transact(
		logTransactionStateHandler(fmt.Sprintf("receipt refund, transferId: %x", transferId)),
		func(ctr bind.ContractTransactor, opts *bind.TransactOpts) (*ethtypes.Transaction, error) {
			cbt, err2 := contracts.NewCBridgeTransactor(bc.contractChain.GetAddr(), ctr)
			if err2 != nil {
				return nil, err2
			}
			return cbt.Refund(opts, transferId)
		},
	)
	return err
}

func logTransactionStateHandler(desc string) *eth.TransactionStateHandler {
	return &eth.TransactionStateHandler{
		OnMined: func(receipt *ethtypes.Receipt) {
			if receipt.Status == ethtypes.ReceiptStatusSuccessful {
				log.Infof("%s transaction %x succeeded", desc, receipt.TxHash)
			} else {
				log.Errorf("%s transaction %x failed", desc, receipt.TxHash)
			}
		},
		OnError: func(tx *ethtypes.Transaction, err error) {
			log.Errorf("%s transaction %x err: %s", desc, tx.Hash(), err)
		},
	}
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

func (s *server) transactorOptions(bc *bridgeConfig) []eth.TxOption {
	ops := []eth.TxOption{
		eth.WithTimeout(transactorWaitTimeout),
		eth.WithBlockDelay(bc.config.WatchConfig.GetBlockDelay()),
		eth.WithPollingInterval(time.Duration(bc.config.WatchConfig.GetPollingInterval()) * time.Second),
	}
	if bc.config.GetTransactorConfig().GetAddGasGwei() > 0 {
		ops = append(ops, eth.WithAddGasGwei(bc.config.GetTransactorConfig().GetAddGasGwei()))
	}
	if bc.config.GetTransactorConfig().GetAddGasEstimateRatio() > 0 {
		ops = append(ops, eth.WithAddGasEstimateRatio(bc.config.GetTransactorConfig().GetAddGasEstimateRatio()))
	}
	if bc.config.GetTransactorConfig().GetGasLimit() > 0 {
		ops = append(ops, eth.WithGasLimit(bc.config.GetTransactorConfig().GetGasLimit()))
	}
	return ops
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
		EthAddr:     s.accountAddr.String(),
		ChainInfo:   []*gatewayrpc.ChainInfo{},
		NickName:    s.cfg.GetRelayNodeName(),
		Sig:         sigMsg,
		NodeVersion: s.version,
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
	return nil
}

func (s *server) setGatewayChainInfo(gatewayChainInfoMap map[uint64]*gatewayrpc.GatewayChainInfo) {
	s.gatewayChainInfoMapLock.Lock()
	defer s.gatewayChainInfoMapLock.Unlock()
	s.gatewayChainInfoMap = gatewayChainInfoMap
}

func (s *server) ProcessSendTransfer() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.processTrySendTransferIn()
		}
	}
}

func (s *server) ProcessConfirmTransfer() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.processTryConfirmTransfer()
		}
	}
}

func (s *server) ProcessRefundTransferIn() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.processTryRefundTransferIn()
		}
	}
}

func (s *server) ProcessRecoverTimeoutPendingTransfer() {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.processRecoverTimeoutPendingTransferIn()
			s.processRecoverTimeoutPendingConfirm()
			s.processRecoverTimeoutPendingRefund()
		}
	}
}

func (s *server) processTrySendTransferIn() {
	startedTransferIn, dbErr := s.db.GetAllStartTransferIn()
	if dbErr != nil {
		log.Warnf("fail to query started transferIn, err:%s", dbErr)
		return
	}
	for _, tx := range startedTransferIn {
		bc, foundBc := s.chainMap[tx.ChainId]
		if foundBc {
			tsNow := time.Now()
			if tx.TimeLock.Add(time.Duration(-6) * time.Minute).Before(tsNow) {
				log.Warnf("this transfer out is already timeout, transferId:%x, timeLock: %s", tx.TransferId, tx.TimeLock.String())
				continue
			}

			remoteTransferIn, err := bc.getTransfer(tx.TransferId)
			if err != nil {
				log.Errorf("fail to get transfer in, txId:%x, err:%v", tx.TransferId, err)
				continue
			}
			if remoteTransferIn.Status != 0 {
				log.Warnf("this transfer in already exist, we try to set the status to locked, transfer:%+v", remoteTransferIn)
				if remoteTransferIn.Status == remoteTransferStatusPending {
					// for some chain, we may miss transfer in event, then we should set it.
					log.Warnf("find exist pending transfer, tid:%x, try to set the status", tx.TransferId)
					tryResetPendingTransferInStatusErr := s.db.SetTransferStatusByFrom(tx.TransferId, cbn.TransferStatus_TRANSFER_STATUS_LOCKED,
						cbn.TransferStatus_TRANSFER_STATUS_TRANSFER_IN_START)
					if tryResetPendingTransferInStatusErr != nil {
						log.Errorf("fail to set exist transferIn, tid:%x, err:%v", tx.TransferId, tryResetPendingTransferInStatusErr)
					}
				}
				continue
			}

			// if the old fee is recorded, may caused by try send transfer in failed
			// then we add it back first.
			originAmt := new(big.Int).Add(&tx.Amount, &tx.Fee)

			finalFee, getFeeErr := s.gateway.GetFee(tx.RelatedTid)
			if getFeeErr != nil {
				log.Errorf("can not get the fee for this transfer, transferOutId:%x, err:%v", tx.RelatedTid, getFeeErr)
				continue
			}

			log.Infof("tx:%s, final fee:%s", tx.RelatedTid.String(), finalFee.String())

			if finalFee.Cmp(originAmt) > 0 {
				log.Errorf("this fee is bigger than amount, transferOutId:%x, fee:%s, origin amount:%s, err:%v", tx.RelatedTid, finalFee.String(), originAmt.String(), getFeeErr)
				continue
			}

			newAmount := new(big.Int).Sub(originAmt, finalFee)

			setTransferInAmountAndFeeErr := s.db.SetTransferInAmountAndFee(tx.TransferId, newAmount, finalFee)
			if setTransferInAmountAndFeeErr != nil {
				log.Errorf("fail to set transferIn fee and amount, ev:%v, err:%v", tx, setTransferInAmountAndFeeErr)
				continue
			}

			setDbTransferToPendingErr := s.db.SetPendingTransferIn(tx.TransferId, cbn.TransferStatus_TRANSFER_STATUS_TRANSFER_IN_PENDING,
				cbn.TransferStatus_TRANSFER_STATUS_TRANSFER_IN_START)
			if setDbTransferToPendingErr != nil {
				log.Errorf("fail to set transferIn to pending, ev:%v, err:%v", tx, setDbTransferToPendingErr)
				continue
			}

			sendTransferInErr := bc.transferIn(tx.Receiver, tx.Token, newAmount, tx.HashLock, tx.TransferId, tx.RelatedTid, uint64(tx.TimeLock.Unix()), tx.RelatedChainId)
			if sendTransferInErr != nil {
				log.Errorf("fail to transferIn, ev:%+v, err:%v", tx, sendTransferInErr)
				// if fail, let try again one time
				time.Sleep(6 * time.Second)
				sendTransferInAgainErr := bc.transferIn(tx.Receiver, tx.Token, newAmount, tx.HashLock, tx.TransferId, tx.RelatedTid, uint64(tx.TimeLock.Unix()), tx.RelatedChainId)
				if sendTransferInAgainErr != nil {
					log.Errorf("fail to transferIn again, ev:%+v, err:%v", tx, sendTransferInAgainErr)
				} else {
					log.Infof("retry send transferIn success, transferIn id: %x", tx.TransferId)
				}
				continue
			}
		}
	}
}

// Once we get the confirm monitor event, we will save the preimage to both transferOut and transferIn.
// Then we will scan all the transfers(both transferIn and transferOut) status is locked and preimage is not "" ot "0x00000..."
// Every transfers in db match this condition, we will try to confirm it.
func (s *server) processTryConfirmTransfer() {
	lockedTransfer, dbErr := s.db.GetAllConfirmableLockedTransfer()
	if dbErr != nil {
		log.Errorf("fail to query confirmable transfers, err:%s", dbErr)
		return
	}
	for _, tx := range lockedTransfer {
		dstBcg, foundBcg := s.chainMap[tx.ChainId]
		if foundBcg {
			remoteTransfer, err := dstBcg.getTransfer(tx.TransferId)
			if err != nil {
				log.Errorf("fail to get transfer, txId:%x, err:%v", tx.TransferId, err)
				continue
			}
			log.Infof("get remote confirmable transfer:%v", remoteTransfer)
			if remoteTransfer.Status == remoteTransferStatusPending {
				dbErr = s.db.SetTransferStatusByFrom(tx.TransferId, cbn.TransferStatus_TRANSFER_STATUS_CONFIRM_PENDING, cbn.TransferStatus_TRANSFER_STATUS_LOCKED)
				if dbErr != nil {
					log.Errorf("update refund to confirm pending failed, tx:%v, err:%v", tx, dbErr)
				}

				log.Infof("try confirm this tx, txId:%x, txType:%s", tx.TransferId, tx.TransferType.String())
				err = dstBcg.confirm(tx.TransferId, tx.RelatedTid, tx.Preimage, tx.HashLock)
				if err != nil {
					log.Errorf("fail to confirm related transfer, ev:%v, err:%v", tx, err)
					continue
				}
			} else if remoteTransfer.Status == remoteTransferStatusRefunded {
				dbErr = s.db.UpdateTransferStatus(tx.TransferId, cbn.TransferStatus_TRANSFER_STATUS_REFUNDED)
				if dbErr != nil {
					log.Errorf("this transfer is refunded by fail to update the status in db, tx:%v, err:%v", tx, dbErr)
					continue
				}
			} else if remoteTransfer.Status == remoteTransferStatusConfirmed {
				dbErr = s.db.ConfirmTransfer(tx.TransferId, tx.Preimage, Hash{})
				if dbErr != nil {
					log.Errorf("fail to update transfer status to confirmed, tx:%v, err:%v", tx, dbErr)
					continue
				}
			} else {
				log.Warnf("this transfer in status is invalid, transfer:%v", remoteTransfer)
			}
		} else {
			log.Warnf("skip to confirm this tx, because we can not find this chain:%d ,tx:%v", tx.TransferId, tx)
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
				dbErr = s.db.SetTransferStatusByFrom(tx.TransferId, cbn.TransferStatus_TRANSFER_STATUS_REFUND_PENDING, cbn.TransferStatus_TRANSFER_STATUS_LOCKED)
				if dbErr != nil {
					log.Errorf("fail to update the refund pending status in db, tx:%v, err:%v", tx, dbErr)
				}
				// before do refund, we should check the related transfer, if it is already confirmed, then we should confirm this transfer instead of refund it.
				relatedTransfer, exist, getTransferByRelatedTidErr := s.db.GetTransferByTid(tx.RelatedTid)
				if getTransferByRelatedTidErr != nil {
					log.Errorf("fail to get related transfer, RelatedTid:%s, err:%v", tx.RelatedTid.String(), err)
					continue
				}
				if !exist {
					log.Errorf("fail to get related transfer, not exist, transferId:%s, RelatedTid:%s", tx.TransferId.String(), tx.RelatedTid.String())
					continue
				}

				if relatedTransfer.Status == cbn.TransferStatus_TRANSFER_STATUS_CONFIRMED {
					err = bc.confirm(tx.TransferId, relatedTransfer.TransferId, relatedTransfer.Preimage, tx.HashLock)
					if err != nil {
						log.Errorf("fail to confirm this tx: %v", tx)
						continue
					}
				} else {
					// for some chain, we may miss confirm info.
					// here, we should check remote transfer out to make sure it is not confirmed.
					// may be useless, as we can not directly get the preimage
					transferOutBc, foundTransferOutBc := s.chainMap[relatedTransfer.ChainId]
					if !foundTransferOutBc {
						log.Errorf("fail to find transfer out bc, transferOutId:%x", relatedTransfer.TransferId)
						continue
					} else {
						remoteTransferOut, getRemoteTransferOutErr := transferOutBc.getTransfer(relatedTransfer.TransferId)
						if getRemoteTransferOutErr != nil {
							log.Errorf("fail to get remoteTransferOut, transferOutId:%x, err:%v", relatedTransfer.TransferId, err)
							continue
						}
						if remoteTransferOut.Status == remoteTransferStatusConfirmed {
							log.Errorf("this remote transfer out is already confirmed, we should not refund it, transferOutId:%x", relatedTransfer.TransferId)
							continue
						}
					}

					err = bc.refund(tx.TransferId, tx.RelatedTid, tx.HashLock)
					if err != nil {
						log.Errorf("fail to refund this tx: %v, err:%v", tx, err)
						continue
					}
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

func (s *server) processRecoverTimeoutPendingTransferIn() {
	transfers, dbErr := s.db.GetRecoverTimeoutPendingTransferIn()
	if dbErr != nil {
		log.Warnf("fail to get timeout pending transfer in, err:%s", dbErr)
		return
	}
	for _, tx := range transfers {
		dbErr = s.db.SetTransferStatusByFrom(tx.TransferId, cbn.TransferStatus_TRANSFER_STATUS_TRANSFER_IN_START,
			cbn.TransferStatus_TRANSFER_STATUS_TRANSFER_IN_PENDING)
		if dbErr != nil {
			log.Warnf("fail to recover timeout pending transfer in, err:%s", dbErr)
		}
	}
}

func (s *server) processRecoverTimeoutPendingConfirm() {
	transfers, dbErr := s.db.GetRecoverTimeoutPendingConfirm()
	if dbErr != nil {
		log.Warnf("fail to get timeout pending confirm, err:%s", dbErr)
		return
	}
	for _, tx := range transfers {
		dbErr = s.db.SetTransferStatusByFrom(tx.TransferId, cbn.TransferStatus_TRANSFER_STATUS_LOCKED,
			cbn.TransferStatus_TRANSFER_STATUS_CONFIRM_PENDING)
		if dbErr != nil {
			log.Warnf("fail to recover timeout pending confirm transfer in, err:%s", dbErr)
		}
	}
}

func (s *server) processRecoverTimeoutPendingRefund() {
	transfers, dbErr := s.db.GetRecoverTimeoutPendingRefund()
	if dbErr != nil {
		log.Warnf("fail to recover timeout pending refund refund, err:%s", dbErr)
		return
	}
	for _, tx := range transfers {
		dbErr = s.db.SetTransferStatusByFrom(tx.TransferId, cbn.TransferStatus_TRANSFER_STATUS_LOCKED,
			cbn.TransferStatus_TRANSFER_STATUS_REFUND_PENDING)
		if dbErr != nil {
			log.Warnf("fail to recover timeout pending refund transfer in, err:%s", dbErr)
		}
	}
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
					TokenName:    s.getTokenName(transferIn.ChainId, transferIn.Token),
					TokenAddr:    transferIn.Token,
					TotalVolume:  new(big.Int),
					FeeAmount:    new(big.Int),
					TokenDecimal: s.getTokenDecimal(transferIn.ChainId, transferIn.Token),
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

func (s *server) getTokenName(chainId uint64, tokenAddr Addr) string {
	name, foundName := s.chainTokenNameMap[chainId][tokenAddr]
	if !foundName {
		name = tokenAddr.String()
	}
	return name
}

func (s *server) getTokenDecimal(chainId uint64, tokenAddr Addr) uint64 {
	decimal, found := s.chainTokenDecimalMap[chainId][tokenAddr]
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
