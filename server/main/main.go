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
	port    = flag.Int("p", 8088, "web port used for get relay node stats")
	config  = flag.String("c", "", "config json file path")
	showver = flag.Bool("v", false, "Show version and exit")
)

func main() {
	flag.Parse()
	if *showver {
		printver()
		os.Exit(0)
	}
	log.Infoln("Starting cBridge node...")
	s := server.NewServer()
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

	err = s.Init(cbConfig)
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
