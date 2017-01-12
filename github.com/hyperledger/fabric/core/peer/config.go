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

// The 'viper' package for configuration handling is very flexible, but has
// been found to have extremely poor performance when configuration values are
// accessed repeatedly. The function CacheConfiguration() defined here caches
// all configuration values that are accessed frequently.  These parameters
// are now presented as function calls that access local configuration
// variables.  This seems to be the most robust way to represent these
// parameters in the face of the numerous ways that configuration files are
// loaded and used (e.g, normal usage vs. test cases).

// The CacheConfiguration() function is allowed to be called globally to
// ensure that the correct values are always cached; See for example how
// certain parameters are forced in 'ChaincodeDevMode' in main.go.

package peer

import (
	"fmt"
	"net"

	"github.com/spf13/viper"

	pb "github.com/hyperledger/fabric/protos"
)

// cached是不是configuration缓存?
var configurationCached = false

// 计算常量函数getLocalAddress(),getValidatorStreamAddress(), and getPeerEndpoint()
// 的缓存值和错误值
var localAddress string
var localAddressError error
var peerEndpoint *pb.PeerEndpoint
var peerEndpointError error

// 通常使用的配置常量的缓存值
var syncStateSnapshotChannelSize int
var syncStateDeltasChannelSize int
var syncBlocksChannelSize int
var validatorEnabled bool

// 有一些循环导入问题，阻止我们将“core”包导入“peer”包。
// 'peer.SecurityEnabled'比特是一个'core.SecurityEnabled'的副本。
var securityEnabled bool

// CacheConfiguration计算和缓存经常使用的常量且计算常量做为包变量，按照惯例前面的全局变量
// 已经被嵌入在这里为了保留原始的抽象状态
func CacheConfiguration() (err error) {
	// getLocalAddress 返回正在操作的本地peer的address:port，受到env:peer.addressAutoDetect的影响
	getLocalAddress := func() (peerAddress string, err error) {
		if viper.GetBool("peer.addressAutoDetect") {
			// 需要从peer.address设置中获取端口号，并将其添加到已经确定的主机ip后
			_, port, err := net.SplitHostPort(viper.GetString("peer.address"))
			if err != nil {
				err = fmt.Errorf("Error auto detecting Peer's address: %s", err)
				return "", err
			}
			peerAddress = net.JoinHostPort(GetLocalIP(), port)
			peerLogger.Infof("Auto detected peer address: %s", peerAddress)
		} else {
			peerAddress = viper.GetString("peer.address")
		}
		return
	}

	// getPeerEndpoint 对于这个Peer实例来说，返回PeerEndpoint，受到env:peer.addressAutoDetect的影响
	getPeerEndpoint := func() (*pb.PeerEndpoint, error) {
		var peerAddress string
		var peerType pb.PeerEndpoint_Type
		peerAddress, err := getLocalAddress()
		if err != nil {
			return nil, err
		}
		if viper.GetBool("peer.validator.enabled") {
			peerType = pb.PeerEndpoint_VALIDATOR
		} else {
			peerType = pb.PeerEndpoint_NON_VALIDATOR
		}
		return &pb.PeerEndpoint{ID: &pb.PeerID{Name: viper.GetString("peer.id")}, Address: peerAddress, Type: peerType}, nil
	}

	localAddress, localAddressError = getLocalAddress()
	peerEndpoint, peerEndpointError = getPeerEndpoint()

	syncStateSnapshotChannelSize = viper.GetInt("peer.sync.state.snapshot.channelSize")
	syncStateDeltasChannelSize = viper.GetInt("peer.sync.state.deltas.channelSize")
	syncBlocksChannelSize = viper.GetInt("peer.sync.blocks.channelSize")
	validatorEnabled = viper.GetBool("peer.validator.enabled")

	securityEnabled = viper.GetBool("security.enabled")

	configurationCached = true

	if localAddressError != nil {
		return localAddressError
	} else if peerEndpointError != nil {
		return peerEndpointError
	}
	return
}

// cacheConfiguration如果检查失败打一个错误日志
func cacheConfiguration() {
	if err := CacheConfiguration(); err != nil {
		peerLogger.Errorf("Execution continues after CacheConfiguration() failure : %s", err)
	}
}

//函数形式
// GetLocalAddress返回peer.address
func GetLocalAddress() (string, error) {
	if !configurationCached {
		cacheConfiguration()
	}
	return localAddress, localAddressError
}

// GetPeerEndpoint 从缓存配置中返回peerEndpoint
func GetPeerEndpoint() (*pb.PeerEndpoint, error) {
	if !configurationCached {
		cacheConfiguration()
	}
	return peerEndpoint, peerEndpointError
}

// SyncStateSnapshotChannelSize返回peer.sync.state.snapshot.channelSize性能
func SyncStateSnapshotChannelSize() int {
	if !configurationCached {
		cacheConfiguration()
	}
	return syncStateSnapshotChannelSize
}

// SyncStateDeltasChannelSize返回peer.sync.state.deltas.channelSize性能
func SyncStateDeltasChannelSize() int {
	if !configurationCached {
		cacheConfiguration()
	}
	return syncStateDeltasChannelSize
}

// SyncBlocksChannelSize返回peer.sync.blocks.channelSize性能
func SyncBlocksChannelSize() int {
	if !configurationCached {
		cacheConfiguration()
	}
	return syncBlocksChannelSize
}

// ValidatorEnabled返回peer.validator.enabled是否可用
func ValidatorEnabled() bool {
	if !configurationCached {
		cacheConfiguration()
	}
	return validatorEnabled
}

// SecurityEnabled 从配置中返回安全可用性能
func SecurityEnabled() bool {
	if !configurationCached {
		cacheConfiguration()
	}
	return securityEnabled
}
