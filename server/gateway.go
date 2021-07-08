package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"google.golang.org/grpc/credentials"
	"math/big"

	"github.com/celer-network/cBridge-go/gatewayrpc"
	"github.com/celer-network/goutils/log"
	"google.golang.org/grpc"
)

type GatewayAPI interface {
	Close()
	PingGateway(req *gatewayrpc.PingRequest) (*gatewayrpc.PingResponse, error)
	GetFee(transferOutId Hash) (fee *big.Int, err error)
}

type GatewayClient struct {
	gatewayConn *grpc.ClientConn
}

func NewGatewayAPI(gatewayUrl string) (*GatewayClient, error) {
	config := &tls.Config{}
	opts := []grpc.DialOption{grpc.WithTransportCredentials(credentials.NewTLS(config))}
	conn, err := grpc.Dial(gatewayUrl, opts...)
	if err != nil {
		return nil, err
	}
	gateway := &GatewayClient{
		gatewayConn: conn,
	}
	return gateway, nil
}

func (c *GatewayClient) GetFee(transferOutId Hash) (*big.Int, error) {
	client := gatewayrpc.NewRelayClient(c.gatewayConn)
	req := &gatewayrpc.GetFeeRequest{
		TransferOutId: transferOutId.String(),
	}
	resp, pingErr := client.GetFee(context.Background(), req)
	if pingErr != nil {
		return nil, pingErr
	}
	if beErr := resp.GetErr(); beErr != nil {
		return nil, fmt.Errorf("fail to get fee with token, err:%v", beErr)
	}
	fee, success := new(big.Int).SetString(resp.GetFee(), 10)
	if !success {
		return nil, fmt.Errorf("fail to get fee with token, set string fail, resp:%v", resp)
	}
	return fee, nil
}

func (c *GatewayClient) PingGateway(req *gatewayrpc.PingRequest) (*gatewayrpc.PingResponse, error) {
	client := gatewayrpc.NewRelayClient(c.gatewayConn)
	resp, pingErr := client.Ping(context.Background(), req)
	if pingErr != nil {
		return nil, pingErr
	}
	if beErr := resp.GetErr(); beErr != nil {
		return nil, fmt.Errorf("fail to ping gateway and get fee map, err:%v", beErr)
	}
	return resp, nil
}

func (c *GatewayClient) Close() {
	if err := c.gatewayConn.Close(); err != nil {
		log.Warnln("closeGatewayConn: error:", err)
	}
}
