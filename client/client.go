package client

import (
	"context"
	"google.golang.org/grpc/keepalive"
	"log"
	"os"
	"time"

	pb "github.com/filstar/sector-counter/proto"
	"google.golang.org/grpc"
)

// Client ..
type Client struct {
	DialAddr string
}

// NewClient ..
func NewClient() *Client {
	rpcAddr, ok := os.LookupEnv("SC_LISTEN")
	if !ok {
		log.Println("NO SC_LISTEN ENV")
	}

	return &Client{
		DialAddr: rpcAddr,
	}
}

func (c *Client) connect() (pb.GrpcClient, *grpc.ClientConn, error) {
	var kacp = keepalive.ClientParameters{
		Time:                10 * time.Second, // send pings every 10 seconds if there is no activity
		Timeout:             time.Second,      // wait 1 second for ping ack before considering the connection dead
		PermitWithoutStream: true,             // send pings even without active streams
	}

	conn, err := grpc.Dial(c.DialAddr, grpc.WithInsecure(), grpc.WithKeepaliveParams(kacp)) //连接gRPC服务器
	if err != nil {
		return nil, nil, err
	}
	client := pb.NewGrpcClient(conn) //建立客户端
	return client, conn, nil
}

// GetSectorID ..
func (c *Client) GetSectorID(ctx context.Context, param string) (uint64, error) {
	client, conn, err := c.connect()
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	req := new(pb.SectorIDRequest)

	req.Question = param
	resp, err := client.GetSectorID(ctx, req) //调用方法
	if err != nil {
		return 0, err
	}
	return resp.Answer, nil
}
