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

package crypto

import (
	"crypto/ecdsa"
	"crypto/x509"

	"github.com/hyperledger/fabric/core/crypto/primitives"
	"github.com/hyperledger/fabric/core/crypto/utils"
)

// 公共结构体
type nodeImpl struct {
	isRegistered  bool
	isInitialized bool

	// 节点类型
	eType NodeType // int32

	// 配置
	conf *configuration

	// 密钥存储
	ks *keyStore

	// Certs 池
	rootsCertPool *x509.CertPool
	tlsCertPool   *x509.CertPool
	ecaCertPool   *x509.CertPool
	tcaCertPool   *x509.CertPool

	// 48比特的标志符
	id []byte

	// 登记证书和私钥
	enrollID       string
	enrollCert     *x509.Certificate
	enrollPrivKey  *ecdsa.PrivateKey
	enrollCertHash []byte

	// 登记链
	enrollChainKey interface{}

	// TLS
	tlsCert *x509.Certificate

	// Crypto SPI
	eciesSPI primitives.AsymmetricCipherSPI
}

type registerFunc func(eType NodeType, name string, pwd []byte, enrollID, enrollPWD string) error
type initalizationFunc func(eType NodeType, name string, pwd []byte) error

func (node *nodeImpl) GetType() NodeType {
	return node.eType
}

func (node *nodeImpl) GetName() string {
	return node.conf.name
}

func (node *nodeImpl) IsInitialized() bool {
	return node.isInitialized
}

func (node *nodeImpl) setInitialized() {
	node.isInitialized = true
}

func (node *nodeImpl) IsRegistered() bool {
	return node.isRegistered
}

func (node *nodeImpl) setRegistered() {
	node.isRegistered = true
}

func (node *nodeImpl) register(eType NodeType, name string, pwd []byte, enrollID, enrollPWD string, regFunc registerFunc) error {
	// Set entity type
	// 设置实体类型
	node.eType = eType

	// 初始化配置
	if err := node.initConfiguration(name); err != nil {
		node.Errorf("Failed initiliazing configuration [%s]: [%s].", enrollID, err)
		return err
	}

	// Initialize keystore
	// 初始化密钥存储
	err := node.initKeyStore(pwd)
	if err != nil {
		if err == utils.ErrKeyStoreAlreadyInitialized {
			node.Error("Keystore already initialized.")
		} else {
			node.Errorf("Failed initiliazing keystore [%s].", err.Error())
		}
		return err
	}

	if node.IsRegistered() {
		return utils.ErrAlreadyRegistered
	}
	if node.IsInitialized() {
		return utils.ErrAlreadyInitialized
	}

	err = node.nodeRegister(eType, name, pwd, enrollID, enrollPWD)
	if err != nil {
		return err
	}

	if regFunc != nil {
		err = regFunc(eType, name, pwd, enrollID, enrollPWD)
		if err != nil {
			return err
		}
	}

	node.setRegistered()
	node.Debugf("Registration of node [%s] with name [%s] completed", eType, name)

	return nil
}

func (node *nodeImpl) nodeRegister(eType NodeType, name string, pwd []byte, enrollID, enrollPWD string) error {
	// Register crypto engine
	// 注册加密引擎
	err := node.registerCryptoEngine(enrollID, enrollPWD)
	if err != nil {
		node.Errorf("Failed registering node crypto engine [%s].", err.Error())
		return err
	}

	return nil
}

func (node *nodeImpl) init(eType NodeType, name string, pwd []byte, initFunc initalizationFunc) error {
	// Set entity type
	// 设置安全类型
	node.eType = eType

	// Init Conf
	// 初始化配置
	if err := node.initConfiguration(name); err != nil {
		node.Errorf("Failed initiliazing configuration: [%s]", err)
		return err
	}

	// Initialize keystore
	// 初始化密钥存储
	err := node.initKeyStore(pwd)
	if err != nil {
		if err == utils.ErrKeyStoreAlreadyInitialized {
			node.Error("Keystore already initialized.")
		} else {
			node.Errorf("Failed initiliazing keystore [%s].", err.Error())
		}
		return err
	}

	if node.IsInitialized() {
		return utils.ErrAlreadyInitialized
	}

	err = node.nodeInit(eType, name, pwd)
	if err != nil {
		return err
	}

	if initFunc != nil {
		err = initFunc(eType, name, pwd)
		if err != nil {
			return err
		}
	}

	node.setInitialized()

	return nil
}

func (node *nodeImpl) nodeInit(eType NodeType, name string, pwd []byte) error {
	// Init crypto engine
	// 初始化加密引擎
	err := node.initCryptoEngine()
	if err != nil {
		node.Errorf("Failed initiliazing crypto engine [%s]. %s", err.Error(), utils.ErrRegistrationRequired.Error())
		return err
	}

	return nil
}

func (node *nodeImpl) close() error {
	// Close keystore
	var err error

	if node.ks != nil {
		err = node.ks.close()
	}

	return err
}
