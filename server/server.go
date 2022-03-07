package server

import (
	"context"
	"google.golang.org/grpc/keepalive"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	pb "github.com/filstar/sector-counter/proto"

	"google.golang.org/grpc"
)

// Service 定义我们的服务
type Service struct {
	SectorIDLk sync.RWMutex // 对应RPC调用GetSectorID，返回miner的sectorid
	SectorID   uint64
	SCFilePath string
}

// GetSectorID 实现 GetSectorID 方法
func (s *Service) GetSectorID(ctx context.Context, req *pb.SectorIDRequest) (*pb.SectorIDResponse, error) {
	s.SectorIDLk.Lock()
	defer s.SectorIDLk.Unlock()
	s.SectorID++
	s.WriteSectorID()
	return &pb.SectorIDResponse{Answer: s.SectorID}, nil
}

// WriteSectorID 实现 WriteSectorID 方法
func (s *Service) WriteSectorID() {
	f, err := os.OpenFile(s.SCFilePath, os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		log.Println(err)
	}
	defer f.Close()
	strID := strconv.FormatUint(s.SectorID, 10)
	_, _ = f.Write([]byte(strID))
}

func readFileSid(filePath string) (uint64, error) {
	if _, err := os.Stat(filePath); err != nil { // 文件不存在
		f, err := os.Create(filePath)
		if err != nil {
			return 0, err
		}
		_, _ = f.Write([]byte("0"))
		f.Close()
		return 0, nil
	}

	// 存在历史文件
	f, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	byteID, err := ioutil.ReadAll(f)
	if err != nil {
		return 0, err
	}

	stringID := strings.Replace(string(byteID), "\n", "", -1)   // 将最后的\n去掉
	sectorID, err := strconv.ParseUint(string(stringID), 0, 64) // 将字符型数字转化为uint64类型
	if err != nil {
		return 0, err
	}

	return sectorID, nil
}

// Run ..
func Run(scFilePath string) {
	rpcAddr, ok := os.LookupEnv("SC_LISTEN")
	if !ok {
		log.Println("NO SC_LISTEN ENV")
		return
	}

	sectorID, err := readFileSid(scFilePath)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("currn sectorid: ", sectorID)

	listener, err := net.Listen("tcp", rpcAddr) // 监听本地端口
	if err != nil {
		log.Println(err)
	}
	log.Println("grpc server Listing on", rpcAddr)
	var kaep = keepalive.EnforcementPolicy{
		MinTime:             5 * time.Second, // If a client pings more than once every 5 seconds, terminate the connection
		PermitWithoutStream: true,            // Allow pings even when there are no active streams
	}

	var kasp = keepalive.ServerParameters{
		MaxConnectionIdle:     15 * time.Second, // If a client is idle for 15 seconds, send a GOAWAY
		MaxConnectionAge:      30 * time.Second, // If any connection is alive for more than 30 seconds, send a GOAWAY
		MaxConnectionAgeGrace: 5 * time.Second,  // Allow 5 seconds for pending RPCs to complete before forcibly closing connections
		Time:                  5 * time.Second,  // Ping the client if it is idle for 5 seconds to ensure the connection is still active
		Timeout:               1 * time.Second,  // Wait 1 second for the ping ack before assuming the connection is dead
	}

	grpcServer := grpc.NewServer(grpc.KeepaliveEnforcementPolicy(kaep), grpc.KeepaliveParams(kasp)) // 新建gRPC服务器实例
	server := &Service{                                                                             // 在gRPC服务器注册我们的服务
		SectorID:   sectorID,
		SCFilePath: scFilePath,
	}
	pb.RegisterGrpcServer(grpcServer, server)

	if err = grpcServer.Serve(listener); err != nil { //用服务器 Serve() 方法以及我们的端口信息区实现阻塞等待，直到进程被杀死或者 Stop() 被调用
		log.Println(err)
	}
}
