/*
Copyright IBM Corp. 2016 All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

		 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package node

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/hyperledger/fabric/consensus/helper"
	"github.com/hyperledger/fabric/core"
	"github.com/hyperledger/fabric/core/chaincode"
	"github.com/hyperledger/fabric/core/comm"
	"github.com/hyperledger/fabric/core/crypto"
	"github.com/hyperledger/fabric/core/db"
	"github.com/hyperledger/fabric/core/ledger/genesis"
	"github.com/hyperledger/fabric/core/peer"
	"github.com/hyperledger/fabric/core/rest"
	"github.com/hyperledger/fabric/core/system_chaincode"
	"github.com/hyperledger/fabric/events/producer"
	pb "github.com/hyperledger/fabric/protos"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/grpclog"
)

var chaincodeDevMode bool

func startCmd() *cobra.Command {
	// 在node节点上设置node启动命令
	flags := nodeStartCmd.Flags()
	flags.BoolVarP(&chaincodeDevMode, "peer-chaincodedev", "", false,
		"Whether peer in chaincode development mode")

	return nodeStartCmd
}

var nodeStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Starts the node.",
	Long:  `Starts a node that interacts with the network.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return serve(args)
	},
}

//==============================================================================
//peer node start 之后做日志初始化之后就进入到server函数
//==============================================================================
func serve(args []string) error {
	// Parameter overrides must be processed before any paramaters are
	// cached. Failures to cache cause the server to terminate immediately.
	//在其他参数被缓存起来之前参数覆盖必须处理，失败缓存导致服务立即结束
	if chaincodeDevMode {
		logger.Info("Running in chaincode development mode")
		logger.Info("Set consensus to NOOPS and user starts chaincode")
		logger.Info("Disable loading validity system chaincode")

		viper.Set("peer.validator.enabled", "true")
		viper.Set("peer.validator.consensus", "noops")
		viper.Set("chaincode.mode", chaincode.DevModeUserRunsChaincode)

	}
	// 调用CacheConfiguration() 函数设置缓存数据，缓存数据包括该peer的LocalAdress、
	// PeerEndpoint（是VP or NVP）
	if err := peer.CacheConfiguration(); err != nil {
		return err
	}

	peerEndpoint, err := peer.GetPeerEndpoint()
	if err != nil {
		err = fmt.Errorf("Failed to get Peer Endpoint: %s", err)
		return err
	}

	// 启动grpc服务，监听7051端口
	// 设置服务器地址，创建服务器实例，后续代码会使用lis
	listenAddr := viper.GetString("peer.listenAddress")

	if "" == listenAddr {
		logger.Debug("Listen address not specified, using peer endpoint address")
		listenAddr = peerEndpoint.Address
	}

	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		grpclog.Fatalf("Failed to listen: %v", err)
	}
	//创建EventHub服务器，通过调用createEventHubServer方法实现，该服务也是grpc,只有VP
	//才能开启
	ehubLis, ehubGrpcServer, err := createEventHubServer()
	if err != nil {
		grpclog.Fatalf("Failed to create ehub server: %v", err)
	}

	logger.Infof("Security enabled status: %t", core.SecurityEnabled())
	if viper.GetBool("security.privacy") {
		if core.SecurityEnabled() {
			logger.Infof("Privacy enabled status: true")
		} else {
			panic(errors.New("Privacy cannot be enabled as requested because security is disabled"))
		}
	} else {
		logger.Infof("Privacy enabled status: false")
	}
	//启动rockdb数据库
	db.Start()

	var opts []grpc.ServerOption
	if comm.TLSEnabled() {
		creds, err := credentials.NewServerTLSFromFile(viper.GetString("peer.tls.cert.file"),
			viper.GetString("peer.tls.key.file"))

		if err != nil {
			grpclog.Fatalf("Failed to generate credentials %v", err)
		}
		opts = []grpc.ServerOption{grpc.Creds(creds)}
	}

	//创建一个grpc服务
	grpcServer := grpc.NewServer(opts...)

	//注册Chaincode支持服务器
	secHelper, err := getSecHelper()
	if err != nil {
		return err
	}

	secHelperFunc := func() crypto.Peer {
		return secHelper
	}

	registerChaincodeSupport(chaincode.DefaultChain, grpcServer, secHelper)

	var peerServer *peer.Impl

	// 创建peer服务器，主意区分VP和NVP节点
	if peer.ValidatorEnabled() {
		logger.Debug("Running as validating peer - making genesis block if needed")
		makeGenesisError := genesis.MakeGenesis()
		if makeGenesisError != nil {
			return makeGenesisError
		}
		logger.Debugf("Running as validating peer - installing consensus %s",
			viper.GetString("peer.validator.consensus"))

		// 网络初始化的过程中执行以下内容，在创建节点Engine过程中该节点作为客户端的身份
		// 连接到其他Peer
		peerServer, err = peer.NewPeerWithEngine(secHelperFunc, helper.GetEngine)
	} else {
		logger.Debug("Running as non-validating peer")
		peerServer, err = peer.NewPeerWithHandler(secHelperFunc, peer.NewPeerHandler)
	}

	if err != nil {
		logger.Fatalf("Failed creating new peer with handler %v", err)

		return err
	}

	// 注册peer服务
	pb.RegisterPeerServer(grpcServer, peerServer)

	// 注册管理服务器
	pb.RegisterAdminServer(grpcServer, core.NewAdminServer())

	// 注册Devops服务器
	// 在Peer节点初始化的时候 创建DevopsServer
	serverDevops := core.NewDevopsServer(peerServer)
	pb.RegisterDevopsServer(grpcServer, serverDevops)

	// 注册ServerOpenchain服务器
	serverOpenchain, err := rest.NewOpenchainServerWithPeerInfo(peerServer)
	if err != nil {
		err = fmt.Errorf("Error creating OpenchainServer: %s", err)
		return err
	}

	pb.RegisterOpenchainServer(grpcServer, serverOpenchain)

	// 如果配置了的话创建和注册REST服务
	if viper.GetBool("rest.enabled") {
		go rest.StartOpenchainRESTServer(serverOpenchain, serverDevops)
	}

	logger.Infof("Starting peer with ID=%s, network ID=%s, address=%s, rootnodes=%v, validator=%v",
		peerEndpoint.ID, viper.GetString("peer.networkId"), peerEndpoint.Address,
		viper.GetString("peer.discovery.rootnode"), peer.ValidatorEnabled())

	// 启动GRPC服务器. 如果是必须的话在一个goroutine中完成这样我们能够部署genesis
	serve := make(chan error)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		fmt.Println()
		fmt.Println(sig)
		serve <- nil
	}()

	go func() {
		var grpcErr error
		if grpcErr = grpcServer.Serve(lis); grpcErr != nil {
			grpcErr = fmt.Errorf("grpc server exited with error: %s", grpcErr)
		} else {
			logger.Info("grpc server exited")
		}
		serve <- grpcErr
	}()

	if err := writePid(viper.GetString("peer.fileSystemPath")+"/peer.pid", os.Getpid()); err != nil {
		return err
	}

	// 启动eventHub服务
	if ehubGrpcServer != nil && ehubLis != nil {
		go ehubGrpcServer.Serve(ehubLis)
	}

	if viper.GetBool("peer.profile.enabled") {
		go func() {
			profileListenAddress := viper.GetString("peer.profile.listenAddress")
			logger.Infof("Starting profiling server with listenAddress = %s", profileListenAddress)
			if profileErr := http.ListenAndServe(profileListenAddress, nil); profileErr != nil {
				logger.Errorf("Error starting profiler: %s", profileErr)
			}
		}()
	}

	// Block until grpc server exits 产生块直到grpc服务退出
	return <-serve
}

// 该函数主要作用是将系统chaincode部署到Docker上，同时根据第一个参数chainname创建
// ChainCodeSupport 实例;该实例包括 chaincode路径、超时时间、chainname等数据信息。
// 将得到的ChainCodeSupport实例注册到grpcServer
func registerChaincodeSupport(chainname chaincode.ChainName, grpcServer *grpc.Server,
	secHelper crypto.Peer) {

	//get user mode
	//获取用户模式
	userRunsCC := false
	if viper.GetString("chaincode.mode") == chaincode.DevModeUserRunsChaincode {
		userRunsCC = true
	}

	//get chaincode startup timeout
	//获取链码启动超时
	tOut, err := strconv.Atoi(viper.GetString("chaincode.startuptimeout"))
	if err != nil { //what went wrong ?
		fmt.Printf("could not retrive timeout var...setting to 5secs\n")
		tOut = 5000
	}
	ccStartupTimeout := time.Duration(tOut) * time.Millisecond

	ccSrv := chaincode.NewChaincodeSupport(chainname, peer.GetPeerEndpoint, userRunsCC,
		ccStartupTimeout, secHelper)

	//Now that chaincode is initialized, register all system chaincodes.
	// RegisterSysCCs 该函数注册部署系统chaincode
	system_chaincode.RegisterSysCCs()
	// RegisterChaincodeSupportServer 该函数比较重要，该函数向grpcServer注册
	// chainCodeSupport实例，同时传入chainCodeSupport的服务规范
	pb.RegisterChaincodeSupportServer(grpcServer, ccSrv)
}

// 创建 createEventHubServer 该函数创建事件Hub服务器（创建线程运行），同时为该服务器创建监听实例，调用
// RegisterEventsServer 函数将EventHubServer注册到grpcService上
func createEventHubServer() (net.Listener, *grpc.Server, error) {
	var lis net.Listener
	var grpcServer *grpc.Server
	var err error
	// ValidatorEnabled返回peer.validator.enabled是否可用
	if peer.ValidatorEnabled() {
		lis, err = net.Listen("tcp", viper.GetString("peer.validator.events.address"))
		if err != nil {
			return nil, nil, fmt.Errorf("failed to listen: %v", err)
		}

		//TODO - do we need different SSL material for events ?
		var opts []grpc.ServerOption
		// TLSEnabled返回peer.tls.enabled配置好的值的缓存值
		if comm.TLSEnabled() {
			//NewServerTLSFromFile是gRPC库的函数，主要目的是为获取tls的证书和密钥
			creds, err := credentials.NewServerTLSFromFile(
				viper.GetString("peer.tls.cert.file"),
				viper.GetString("peer.tls.key.file"))

			if err != nil {
				return nil, nil, fmt.Errorf("Failed to generate credentials %v", err)
			}
			opts = []grpc.ServerOption{grpc.Creds(creds)}
		}

		grpcServer = grpc.NewServer(opts...)
		ehServer := producer.NewEventsServer(
			uint(viper.GetInt("peer.validator.events.buffersize")),
			viper.GetInt("peer.validator.events.timeout"))

		//注册事件服务
		pb.RegisterEventsServer(grpcServer, ehServer)
	}
	return lis, grpcServer, err
}

func writePid(fileName string, pid int) error {
	err := os.MkdirAll(filepath.Dir(fileName), 0755)
	if err != nil {
		return err
	}

	fd, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer fd.Close()
	if err := syscall.Flock(int(fd.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return fmt.Errorf("can't lock '%s', lock is held", fd.Name())
	}

	if _, err := fd.Seek(0, 0); err != nil {
		return err
	}

	if err := fd.Truncate(0); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(fd, "%d", pid); err != nil {
		return err
	}

	if err := fd.Sync(); err != nil {
		return err
	}

	if err := syscall.Flock(int(fd.Fd()), syscall.LOCK_UN); err != nil {
		return fmt.Errorf("can't release lock '%s', lock is held", fd.Name())
	}
	return nil
}

var once sync.Once

//这个方法只能被调用一次并且缓存结果
//注意这个加密本质上属于加密包并且在哪儿都可以调用
func getSecHelper() (crypto.Peer, error) {
	var secHelper crypto.Peer
	var err error
	once.Do(func() {
		if core.SecurityEnabled() {
			enrollID := viper.GetString("security.enrollID")
			enrollSecret := viper.GetString("security.enrollSecret")
			if peer.ValidatorEnabled() {
				logger.Debugf("Registering validator with enroll ID: %s", enrollID)
				if err = crypto.RegisterValidator(enrollID, nil, enrollID, enrollSecret); nil != err {
					return
				}
				logger.Debugf("Initializing validator with enroll ID: %s", enrollID)
				secHelper, err = crypto.InitValidator(enrollID, nil)
				if nil != err {
					return
				}
			} else {
				logger.Debugf("Registering non-validator with enroll ID: %s", enrollID)
				if err = crypto.RegisterPeer(enrollID, nil, enrollID, enrollSecret); nil != err {
					return
				}
				logger.Debugf("Initializing non-validator with enroll ID: %s", enrollID)
				secHelper, err = crypto.InitPeer(enrollID, nil)
				if nil != err {
					return
				}
			}
		}
	})
	return secHelper, err
}
