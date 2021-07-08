package server

import (
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"math/rand"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"testing"
	"time"

	cbn "github.com/celer-network/cBridge-go/cbridgenode"
	"github.com/celer-network/cBridge-go/contracts"
	"github.com/celer-network/cBridge-go/gatewayrpc"
	"github.com/celer-network/goutils/eth"
	"github.com/celer-network/goutils/log"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"google.golang.org/grpc"
)

const (
	stSvr    = "localhost:3333"
	stWebSvr = "localhost:3344"
	stInfo   = "postgresql://cbridge@" + stSvr + "/cbridge?sslmode=disable"
	stDir    = "/tmp/relaynode_test_sql_db"
	stSchema = "schema.sql"
	chainID  = uint64(883)

	account1   = "0x29B563951Ed0eB9Ae5C49692266E1fbc81445cfE" // a test account
	account1Ks = "../env/ropsten/ks/account1.json"
	account2   = "0x4BEABA029d12536f0D9FB4f242c4CfcA75c4Fe76" // a test account
	account2Ks = "../env/ropsten/ks/account2.json"
	webAccount = "0xC925403763b9eBd6700Ac23c90510F0Ff174dFc3"

	testKsPwd = "123456"

	GethRPC = "wss://ropsten.infura.io/ws/v3/8712e91acda74998974bc52ed2813ac4"
	ethBase = "b5bb8b7f6f1883e0c01ffb8697024532e6f3238c"

	configFilePath = "../env/test/config.json"

	setUpLocalGeth = false

	gasApiUrl = "https://www.gasnow.org/api/v3/gas/price"
)

var (
	// root dir with ending / for all files, eg. /tmp/layer2_e2e_12345678
	// set in TestMain
	outRootDir string

	// below vars are set in setupL1
	ec                   *ethclient.Client
	ethbaseAuth          *bind.TransactOpts // keystore/etherbase.json, has eth, erc20 balance
	cbridgeAddr, daiAddr Addr
	l1dai                *contracts.Erc20 // contract obj, etherbase auth
	l1cbc                *contracts.CBridge
)

type mockGateway struct {
}

func mockGatewayAPI() *mockGateway {

	b := &mockGateway{}
	return b
}

func (mg *mockGateway) PingGateway(req *gatewayrpc.PingRequest) (*gatewayrpc.PingResponse, error) {
	return &gatewayrpc.PingResponse{
		ChainInfo: map[uint64]*gatewayrpc.GatewayChainInfo{
			3: {
				GasPrice: 10,
			},
			5: {
				GasPrice: 10,
			},
			97: {
				GasPrice: 10,
			},
		},
	}, nil
}

func (mg *mockGateway) GetFee(transferOutId Hash) (fee *big.Int, err error) {
	return big.NewInt(1), nil
}

func (mg *mockGateway) Close() {

}

// TestMain is used to setup/teardown a temporary CockroachDB instance
// and run all the unit tests in between.
func TestMain(m *testing.M) {
	flag.Parse()
	rand.Seed(time.Now().Unix())

	if err := setupDb(); err != nil {
		log.Errorf("cannot setup DB:%v", err)
		os.Exit(1)
	}

	if setUpLocalGeth {
		gethPid, err := setupGeth()
		if err != nil {
			log.Errorf("fail to set up geth:%v", err)
			os.Exit(1)
		}
		syscall.Kill(gethPid, syscall.SIGTERM)
	}

	exitCode := m.Run() // run all unittests
	teardownDb()
	os.Exit(exitCode)
}

func setupGeth() (int, error) {
	var err error
	var gethPid int
	// start chain one
	outRootDir = fmt.Sprintf("/tmp/cbridge_%d/", time.Now().Unix())
	fmt.Println("root dir: ", outRootDir)
	err = os.MkdirAll(outRootDir, os.ModePerm)
	chkErr(err, "creating root dir")
	chainDataDir := outRootDir + "chaindata/"
	if err = os.MkdirAll(chainDataDir, os.ModePerm); err != nil {
		log.Errorf("fail to mkdir chainDataDir:%s", chainDataDir)
		return 0, err
	}
	if gethPid, err = startGeth(chainDataDir); err != nil {
		log.Infof("cannot setup DB:%v", err)
		os.Exit(1)
	}

	log.Infoln("geth pid:", gethPid)
	time.Sleep(time.Second * 3) // 3s to wait for geth up
	// L1
	err = setupCbridge(chainDataDir)
	chkErr(err, "deploy contracts")
	return gethPid, nil
}

func setupDb() error {
	// Start the DB.
	err := os.RemoveAll(stDir)
	if err != nil {
		return fmt.Errorf("cannot remove old DB directory: %s: %v", stDir, err)
	}

	schema, err := os.Open(stSchema)
	if err != nil {
		return fmt.Errorf("cannot open schema file: %s: %v", stSchema, err)
	}
	defer schema.Close()

	startDbCmd := exec.Command("cockroach", "start", "--insecure",
		"--listen-addr="+stSvr, "--http-addr="+stWebSvr,
		"--store=path="+stDir)
	if startDbErr := startDbCmd.Start(); startDbErr != nil {
		teardownDb()
		os.Exit(1)
	}

	time.Sleep(time.Second)

	// Setup the DB schema.
	cmd := exec.Command("cockroach", "sql", "--insecure", "--host="+stSvr)
	pipe, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("cannot get stdin of DB command: %v", err)
	}

	go func() {
		defer pipe.Close()
		io.Copy(pipe, schema)
	}()

	if err = cmd.Run(); err != nil {
		teardownDb()
		return fmt.Errorf("cannot setup DB schema: %v", err)
	}

	return nil
}

func teardownDb() {
	log.Infof("teardown db")
	cmd := exec.Command("cockroach", "quit", "--insecure", "--host="+stSvr)
	if err := cmd.Run(); err != nil {
		fmt.Printf("WARNING: cannot terminate DB: %v", err)
	}

	time.Sleep(3 * time.Second)
	os.RemoveAll(stDir)
}

func startGeth(chainDataDir string) (int, error) {
	// keystore is under test folder, so no need to set cmd.Dir
	cmdCopy := exec.Command("cp", "-a", "keystore", chainDataDir)
	if err := cmdCopy.Run(); err != nil {
		return -1, err
	}
	log.Infof("cmd:%s", cmdCopy)
	// init w/ genesis
	cmdInit := exec.Command("/Users/liuxiao/gethTest/geth", "--datadir", chainDataDir, "init", "genesis.json")
	if err := cmdInit.Run(); err != nil {
		return -1, err
	}
	log.Infof("cmd:%s", cmdInit)
	// actually run geth
	logFname := outRootDir + "chain.log"
	logF, _ := os.Create(logFname)
	cmd := exec.Command("/Users/liuxiao/gethTest/geth", "--networkid", strconv.FormatUint(chainID, 10), "--cache", "256", "--nousb", "--syncmode", "full", "--nodiscover", "--maxpeers", "0",
		"--netrestrict", "127.0.0.1/8", "--datadir", chainDataDir, "--keystore", chainDataDir+"keystore", "--targetgaslimit", "12500000",
		"--mine", "--allow-insecure-unlock", "--unlock", "0", "--password", "empty_password.txt", "--rpc", "--rpcaddr", "0.0.0.0", "--rpccorsdomain", "*",
		"--rpcapi", "admin,debug,eth,miner,net,personal,shh,txpool,web3", "--ws", "--wsaddr", "0.0.0.0", "--wsport", "8546", "--wsapi", "admin,debug,eth,miner,net,personal,shh,txpool,web3")
	cmd.Stderr = logF
	cmd.Stdout = logF
	log.Infof("cmd:%s", cmd.String())
	if err := cmd.Start(); err != nil {
		return -1, err
	}
	log.Infof("geth pid:%d", cmd.Process.Pid)
	return cmd.Process.Pid, nil
}

func TestParseConfig(t *testing.T) {
	cbConfig, err := ParseCfgFile(configFilePath)
	if err != nil {
		t.Fatalf("fail to parse config, err:%v", err)
	}
	log.Infof("get cb config:%v", cbConfig)
	log.Infof(Hash{}.String())

	opts := []grpc.DialOption{grpc.WithInsecure()}
	conn, err := grpc.Dial("cbridge-test.celer.network:8081", opts...)
	if err != nil {
		log.Infof("err :%v", err)
		return
	}
	client := gatewayrpc.NewRelayClient(conn)
	req := &gatewayrpc.GetFeeRequest{
		TransferOutId: "123",
	}
	resp, pingErr := client.GetFee(context.Background(), req)
	if pingErr != nil {
		log.Infof("err :%v", pingErr)
		return
	}
	if beErr := resp.GetErr(); beErr != nil {
		log.Infof("err :%v", beErr)
		return
	}
}

func TestUnexpectedInterrupt(t *testing.T) {
	s := NewServer()
	cbConfig, err := ParseCfgFile("../env/local/config.json")
	if err != nil {
		log.Fatal(err)
		return
	}
	log.Infof("s: %s, cbConfig:%v", s.accountAddr.String(), cbConfig)

	err = s.Init(cbConfig)
	if err != nil {
		t.Errorf("fail to init server, err:%v", err)
		return
	}
	s.gateway = mockGatewayAPI()
	s.PingAndRefreshFee()
	//addr1 := Hex2Addr(account1)
	//addr2 := Hex2Addr(account2)

	// test not exist transfer
	transferInfo, err := s.chainMap[3].getTransfer(Hex2Hash("xxxxx"))
	if err != nil {
		log.Infof("fail to get unexist transfer by transfer id, err:%v", err)
	}

	// test exist transfer
	transferInfo, err = s.chainMap[3].getTransfer(Hex2Hash("0x994629749e80221e2b37304706fe816e47fa878f9bdeee8d46a9efbc85d74db4"))
	if err != nil {
		log.Infof("fail to get unexist transfer by transfer id, err:%v", err)
	}
	log.Infof("exist transfer returned:%+v", transferInfo)

}

func TestRopstenTransferOut(t *testing.T) {
	s := NewServer()
	s.gateway = mockGatewayAPI()
	cbConfig, err := ParseCfgFile("../env/local/config.json")
	if err != nil {
		log.Fatal(err)
		return
	}

	err = s.Init(cbConfig)
	if err != nil {
		t.Errorf("fail to init server, err:%v", err)
		return
	}
	log.Infof("s: %s, cbConfig:%v", s.accountAddr.String(), cbConfig)

	s.StartJob()

	preImage, err := getRandomPreImage()
	if err != nil {
		log.Errorf("can not get random preimage, err:%v", err)
		return
	}

	addr1 := Hex2Addr(account1)
	addr2 := Hex2Addr(account2)

	daiContract, err := contracts.NewErc20(Hex2Addr(s.cfg.ChainConfig[0].GetTokenConfig()[0].GetTokenAddress()), s.chainMap[3].ec)
	chkErr(err, "open ropsten dai contract")

	supply := big.NewInt(1e18)
	ksfAc1, err := os.Open(account1Ks)
	chkErr(err, "open account1.json")
	authAccount1, err := bind.NewTransactorWithChainID(ksfAc1, testKsPwd, big.NewInt(int64(3)))
	chkErr(err, "auth account1")
	_, err = daiContract.Approve(authAccount1, s.chainMap[3].contractChain.GetAddr(), new(big.Int).Mul(supply, big.NewInt(10000000)))
	chkErr(err, "Approve dai")

	// this is account1 keystore
	ksJson, err := ioutil.ReadFile(account1Ks)
	if err != nil {
		log.Fatalln("read ks json err:", err)
		return
	}

	tcfg := eth.NewTransactorConfig(string(ksJson), testKsPwd)

	chainId, err := s.chainMap[3].ec.ChainID(context.Background())
	if err != nil {
		log.Errorf("fail to get chain id, err:%v", err)
		return
	}
	var trans *eth.Transactor
	trans, err = eth.NewTransactor(
		tcfg.Keyjson,
		tcfg.Passphrase,
		s.chainMap[3].ec,
		chainId,
		s.waitMinedOptions(s.chainMap[3])...,
	)
	if err != nil {
		log.Errorf("can not get trans, err:%v", err)
		return
	}

	err = transferOut(trans, addr2, Hex2Addr(s.cfg.ChainConfig[0].GetTokenConfig()[0].GetTokenAddress()), addr1, s.chainMap[3].contractChain.GetAddr(),
		Bytes2Hash(crypto.Keccak256(preImage.Bytes())), big.NewInt(10), 97, 1681914767)
	if err != nil {
		log.Errorf("can not transfer out, err:%v", err)
		return
	}

	time.Sleep(60 * time.Second)

	txs, dbErr := s.db.GetAllTransfers()
	if dbErr != nil {
		log.Errorf("fail to get all transfer, err:%v", dbErr)
		return
	}
	for _, tx := range txs {
		if tx.TransferType == cbn.TransferType_TRANSFER_TYPE_OUT {
			cbg, foundCbg := s.chainMap[tx.ChainId]
			if foundCbg {
				err = cbg.confirm(tx.TransferId, preImage, 1)
				if err != nil {
					log.Errorf("can not confirm this tx, err:%v", err)
					return
				}
			}
		}
	}

	// test refund timeout transfer
	/*preImage, err = getRandomPreImage()
	if err != nil {
		log.Errorf("can not get random preimage, err:%v", err)
		return
	}
	err = transferOut(trans, addr2, Hex2Addr(s.cfg.ChainConfig[0].GetTokenConfig()[0].GetTokenAddress()), addr1, s.chainMap[3].contractChain.GetAddr(),
		Bytes2Hash(crypto.Keccak256(preImage.Bytes())), big.NewInt(10), 5, uint64(time.Now().Unix()+30))
	if err != nil {
		log.Errorf("can not transfer out, err:%v", err)
		return
	}*/

	time.Sleep(12 * time.Minute)
}

func TestPingGateway(t *testing.T) {
	log.Infof("ticket")
	opts := []grpc.DialOption{grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(3 * time.Second)}
	conn, err := grpc.Dial("cbridge-test.celer.network:10000", opts...)
	if err != nil {
		log.Warnf("fail to connect gateway, err:%v", err)
	} else {
		client := gatewayrpc.NewRelayClient(conn)
		req := &gatewayrpc.PingRequest{
			EthAddr: "0x4beaba029d12536f0d9fb4f242c4cfca75c4fe76",
			ChainInfo: []*gatewayrpc.ChainInfo{
				{
					ChainId:         3,
					TokenAndBalance: map[string]string{},
				},
			},
		}
		resp, pingErr := client.Ping(context.Background(), req)
		if pingErr != nil {
			log.Errorf("ping failed: %v", pingErr)
			return
		}
		if beErr := resp.GetErr(); beErr != nil {
			log.Errorf("ping failed: %v", resp.GetErr())
			return
		}
	}
}

func TestCBridge(t *testing.T) {
	// start server monitor
	s := NewServer()
	cbConfig, err := ParseCfgFile(configFilePath)
	if err != nil {
		log.Fatal(err)
		return
	}
	cbConfig.ChainConfig[0].ContractAddress = cbridgeAddr.String()
	cbConfig.ChainConfig[0].TokenConfig = []*cbn.TokenConfig{}
	err = s.Init(cbConfig)
	if err != nil {
		t.Errorf("fail to init server, err:%v", err)
		return
	}

	time.Sleep(3 * time.Second)

	// this is account1 keystore
	ksJson, err := ioutil.ReadFile(account1Ks)
	if err != nil {
		log.Fatalln("read ks json err:", err)
		return
	}

	tcfg := eth.NewTransactorConfig(string(ksJson), testKsPwd)

	chainId, err := ec.ChainID(context.Background())
	if err != nil {
		log.Errorf("fail to get chain id, err:%v", err)
		return
	}

	var trans *eth.Transactor
	trans, err = eth.NewTransactor(
		tcfg.Keyjson,
		tcfg.Passphrase,
		ec,
		chainId,
		s.waitMinedOptions(s.chainMap[3])...,
	)
	if err != nil {
		log.Errorf("can not get trans, err:%v", err)
		return
	}

	preImage, err := getRandomPreImage()
	if err != nil {
		log.Errorf("can not get random preimage, err:%v", err)
		return
	}

	addr1 := Hex2Addr(account1)
	addr2 := Hex2Addr(account2)

	err = transferOut(trans, addr2, daiAddr, addr1, cbridgeAddr, Bytes2Hash(crypto.Keccak256(preImage.Bytes())), big.NewInt(200),
		5, 1681914767)
	if err != nil {
		log.Errorf("can not transfer out, err:%v", err)
		return
	}
	time.Sleep(2 * time.Second)
	printBalance(addr1)
	printBalance(addr2)

	time.Sleep(2 * time.Second)
	txs, dbErr := s.db.GetAllTransfers()
	if dbErr != nil {
		log.Errorf("fail to get all transfer, err:%v", dbErr)
		return
	}
	for _, tx := range txs {
		if tx.TransferType == cbn.TransferType_TRANSFER_TYPE_OUT {
			cbg, foundCbg := s.chainMap[tx.ChainId]
			if foundCbg {
				err = cbg.confirm(tx.TransferId, preImage, 1)
				if err != nil {
					log.Errorf("can not confirm this tx, err:%v", err)
					return
				}
			}
		}
	}

	time.Sleep(2 * time.Second)
	printBalance(Hex2Addr(account1))
	printBalance(Hex2Addr(account2))

	s.Close()
}

func setupCbridge(chainDataDir string) error {
	ksf, err := os.Open(chainDataDir + "keystore/etherbase.json")
	chkErr(err, "open etherbase.json")
	auth, err := bind.NewTransactorWithChainID(ksf, "", big.NewInt(int64(chainID)))
	ksf.Close()
	chkErr(err, "bind.NewTransactor etherbase auth")
	ethbaseAuth = auth

	conn, err := ethclient.Dial(GethRPC)
	chkErr(err, "dial geth ws")
	ec = conn

	// 1B supply, 1e18 * 1e9
	supply := big.NewInt(1e18)
	daiTokenAddr, _, erc20, err := contracts.DeployErc20(auth, conn, "DAI Asset", "DAI", new(big.Int).Mul(supply, big.NewInt(1e18)), uint8(18))
	chkErr(err, "deploy DAI")
	l1dai = erc20
	daiAddr = daiTokenAddr

	_, err = l1dai.Approve(auth, daiTokenAddr, new(big.Int).Mul(supply, big.NewInt(1e18)))
	chkErr(err, "Approve dai")

	addr, _, cb, err := contracts.DeployCBridge(auth, conn)
	if err != nil {
		return err
	}
	cbridgeAddr = addr
	l1cbc = cb

	amtbi := big.NewInt(10)
	e6 := big.NewInt(1e6)
	fundEth(Hex2Addr(account1), big.NewInt(1e18)) // 1 ETH only
	fundErc(Hex2Addr(account1), new(big.Int).Mul(amtbi, e6), l1dai)
	fundEth(Hex2Addr(account2), big.NewInt(1e18)) // 1 ETH only
	fundErc(Hex2Addr(account2), new(big.Int).Mul(amtbi, e6), l1dai)
	fundEth(Hex2Addr(webAccount), big.NewInt(1e18)) // 1 ETH only
	fundErc(Hex2Addr(webAccount), new(big.Int).Mul(amtbi, e6), l1dai)

	ksfAc1, err := os.Open(account1Ks)
	chkErr(err, "open account1.json")
	authAccount1, err := bind.NewTransactorWithChainID(ksfAc1, testKsPwd, big.NewInt(int64(chainID)))
	chkErr(err, "auth account1")
	_, err = l1dai.Approve(authAccount1, cbridgeAddr, new(big.Int).Mul(supply, big.NewInt(10000)))
	chkErr(err, "Approve usdc")
	ksfAc2, err := os.Open(account2Ks)
	chkErr(err, "open account2.json")
	authAccount2, err := bind.NewTransactorWithChainID(ksfAc2, testKsPwd, big.NewInt(int64(chainID)))
	chkErr(err, "auth account2")
	_, err = l1dai.Approve(authAccount2, cbridgeAddr, new(big.Int).Mul(supply, big.NewInt(10000)))
	chkErr(err, "Approve usdc")

	printBalance(Hex2Addr(account1))
	printBalance(Hex2Addr(account2))

	return nil
}

// use etherbase to send eth, wait mined
func fundEth(toaddr Addr, amt *big.Int) {
	ctx := context.Background()
	nonce, _ := ec.PendingNonceAt(ctx, Hex2Addr(ethBase))
	tx := ethtypes.NewTransaction(nonce, toaddr, amt, 21000, big.NewInt(1e9), nil)
	signed, _ := ethbaseAuth.Signer(Hex2Addr(ethBase), tx)
	err := ec.SendTransaction(ctx, signed)
	if err != nil {
		log.Errorf("fail to fund eth, err:%v", err)
		return
	}
	_, err = eth.WaitMined(ctx, ec, signed, eth.WithPollingInterval(time.Second))
	if err != nil {
		log.Errorf("fail to wait mined fund eth, err:%v", err)
		return
	}
}

// don't wait
func fundErc(addr Addr, amt *big.Int, erc *contracts.Erc20) {
	_, err := erc.Transfer(ethbaseAuth, addr, amt)
	if err != nil {
		log.Errorf("fail to fund erc, err:%v", err)
		return
	}
}

func printBalance(addr Addr) {
	balance, err := l1dai.BalanceOf(nil, addr)
	if err != nil {
		log.Errorf("fail to get dai balance for account:%v, err:%v", addr, err)
		return
	}
	log.Infof("account:%v dai balance: %+v", addr, balance)

	balance, err = ec.BalanceAt(context.Background(), addr, nil)
	if err != nil {
		log.Errorf("fail to get eth balance for account:%v, err:%v", addr, err)
		return
	}
	log.Infof("account:%v eth balance: %+v", addr, balance)
}

func transferOut(trans *eth.Transactor, cbridgeNodeAddr, token, dstAddr, contract Addr, hasLock Hash, amount *big.Int, dstChainId, timeLock uint64) error {
	log.Infof("do transferOut")
	receipt, err := trans.TransactWaitMined(
		fmt.Sprintf("transferOut:%d", time.Now().Unix()),
		func(ctr bind.ContractTransactor, opts *bind.TransactOpts) (*ethtypes.Transaction, error) {
			cbt, err2 := contracts.NewCBridgeTransactor(contract, ctr)
			if err2 != nil {
				return nil, err2
			}
			return cbt.TransferOut(opts, cbridgeNodeAddr, token, amount, hasLock, timeLock, dstChainId, dstAddr)
		},
		eth.WithTimeout(time.Second), // wait at most 1 minute
	)
	if err != nil {
		log.Infof("error for transferOut: %+v", err)
		return err
	}
	log.Infof("receipt for transferOut: %+v", receipt)
	return nil
}
