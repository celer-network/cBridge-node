package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/celer-network/cBridge-go/server"
	"github.com/celer-network/goutils/log"
	"github.com/julienschmidt/httprouter"
)

var (
	port       = flag.Int("p", 8088, "web port used for get relay node stats")
	config     = flag.String("c", "", "config json file path")
	showver    = flag.Bool("v", false, "Show version and exit")
	ks         = flag.String("ks", "", "path to keystore json file")
	pwdDir     = flag.String("pwddir", "", "path to the directory containing passwords")
	vaultToken = flag.String("vaultToken", "s.IlxKxBRbXwKw3OLdmNDXUdIb", "Vault token for operations")
	vaultAddr  = flag.String("vaultAddr", "http://127.0.0.1:8200", "Vault address")
	secretPath = flag.String("secretPath", "", "the path for the secret") // e.g. secret/data/password
)

func main() {
	flag.Parse()
	if version == "" {
		version = "v1.0.2"
	}
	checkFlags()
	if *showver {
		printver()
		os.Exit(0)
	}
	log.Infof("Starting cBridge node, version:%s  ...", version)
	s := server.NewServer(version)
	log.Infoln("Loading config file...")
	cbConfig, err := server.ParseCfgFile(*config)
	if err != nil {
		log.Fatal(err)
		return
	}
	log.Infoln("Successfully load config file")

	log.Infoln("Connecting to gateway server...")
	err = s.InitGatewayClient(cbConfig.GetGateway())
	if err != nil {
		log.Fatal(err)
		return
	}
	log.Infof("Successfully connected to gateway server")

	err = s.Init(cbConfig, *ks, *pwdDir, *secretPath, *vaultAddr, *vaultToken)
	if err != nil {
		log.Fatal(err)
		return
	}
	log.Infof("cBridge relay node successfully starts")

	go s.PingCron()
	go s.ProcessSendTransfer()
	go s.ProcessConfirmTransfer()
	go s.ProcessRefundTransferIn()
	go s.ProcessRecoverTimeoutPendingTransfer()

	webRouter := httprouter.New()
	webRouter.GET("/v1/summary/total", s.GetTotalSummary)
	webRouter.GET("/v1/transfer/:limit", s.GetTransfer)
	startListenAndServeByPort(*port, webRouter)
}

func startListenAndServeByPort(port int, hanlder http.Handler) {
	err := http.ListenAndServe(fmt.Sprintf(":%d", port), hanlder)
	if err != nil {
		log.Errorf("fail to startListenAndServeByPort, err:%v", err)
	}
}

var (
	version string
	commit  string
)

func printver() {
	fmt.Println("Version:", version)
	fmt.Println("Commit:", commit)
}

func checkFlags() {
	if *config == "" {
		log.Fatalln("-c config not specified")
	}
	if *ks == "" {
		log.Fatalln("-ks keystore not specified")
	}
	if *pwdDir == "" && *secretPath == "" {
		log.Fatalln("-pwddir password file or secretPath not specified")
	}
}
