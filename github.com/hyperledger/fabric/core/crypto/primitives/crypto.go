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

package primitives

import (
	"errors"
	"io"
)

var (
	// 无效的密钥参数
	ErrEncryption = errors.New("Error during encryption.")

	// ErrDecryption Error during decryption
	ErrDecryption = errors.New("Error during decryption.")

	// ErrInvalidSecretKeyType Invalid Secret Key type
	ErrInvalidSecretKeyType = errors.New("Invalid Secret Key type.")

	// ErrInvalidPublicKeyType Invalid Public Key type
	ErrInvalidPublicKeyType = errors.New("Invalid Public Key type.")

	// ErrInvalidKeyParameter Invalid Key Parameter
	ErrInvalidKeyParameter = errors.New("Invalid Key Parameter.")

	// ErrInvalidNilKeyParameter Invalid Nil Key Parameter
	ErrInvalidNilKeyParameter = errors.New("Invalid Nil Key Parameter.")

	// ErrInvalidKeyGeneratorParameter Invalid Key Generator Parameter
	ErrInvalidKeyGeneratorParameter = errors.New("Invalid Key Generator Parameter.")
)

// Parameters is common interface for all the parameters
// 对于所有参数的通用接口
type Parameters interface {

	// GetRand returns the random generated associated to this parameters
	// 随机生成关联参数
	GetRand() io.Reader
}

// CipherParameters is common interface to represent cipher parameters
// 通用接口来表示密码参数
type CipherParameters interface {
	Parameters
}

// AsymmetricCipherParameters is common interface to represent asymmetric cipher parameters
// 通用接口来表示非对称密码参数
type AsymmetricCipherParameters interface {
	CipherParameters

	// IsPublic returns true if the parameters are public, false otherwise.
	// 如果参数是公开的返回true，否则为false。
	IsPublic() bool
}

// PublicKey is common interface to represent public asymmetric cipher parameters
// 通用接口代表非对称公钥参数
type PublicKey interface {
	AsymmetricCipherParameters
}

// PrivateKey is common interface to represent private asymmetric cipher parameters
//通用接口代表非对称私钥参数
type PrivateKey interface {
	AsymmetricCipherParameters

	// GetPublicKey returns the associated public key
	// 返回关联公钥
	GetPublicKey() PublicKey
}

// KeyGeneratorParameters is common interface to represent key generation parameters
// 通用接口来表示密钥生成参数
type KeyGeneratorParameters interface {
	Parameters
}

// KeyGenerator defines a key generator
// 定义密钥生成器
type KeyGenerator interface {
	// Init initializes this generated using the passed parameters
	//初始化生成使用传递的参数
	Init(params KeyGeneratorParameters) error

	// GenerateKey generates a new private key
	// 生成一个新的私钥
	GenerateKey() (PrivateKey, error)
}

// AsymmetricCipher defines an asymmetric cipher
// 定义了一个非对称密码
type AsymmetricCipher interface {
	// Init initializes this cipher with the passed parameters
	// 使用传递的参数初始化
	Init(params AsymmetricCipherParameters) error

	// Process processes the byte array given in input
	// 处理输入给出的字节数组的过程
	Process(msg []byte) ([]byte, error)
}

// SecretKey defines a symmetric key
type SecretKey interface {
	CipherParameters
}

// StreamCipher defines a stream cipher
type StreamCipher interface {
	// Init initializes this cipher with the passed parameters
	Init(forEncryption bool, params CipherParameters) error

	// Process processes the byte array given in input
	Process(msg []byte) ([]byte, error)
}

// KeySerializer defines a key serializer/deserializer
// 密钥序列化/反序列化
type KeySerializer interface {
	// ToBytes converts a key to bytes
	// 转换密钥为字节
	ToBytes(key interface{}) ([]byte, error)

	// ToBytes converts bytes to a key
	// 转换字节为密钥
	FromBytes([]byte) (interface{}, error)
}

// AsymmetricCipherSPI is a Service Provider Interface for AsymmetricCipher
type AsymmetricCipherSPI interface {

	// NewAsymmetricCipherFromPrivateKey creates a new AsymmetricCipher for decryption from a secret key
	//从秘钥创建解密一个新的非对称密码
	NewAsymmetricCipherFromPrivateKey(priv PrivateKey) (AsymmetricCipher, error)

	// NewAsymmetricCipherFromPublicKey creates a new AsymmetricCipher for encryption from a public key
	//从公钥创建解密一个新的非对称密码
	NewAsymmetricCipherFromPublicKey(pub PublicKey) (AsymmetricCipher, error)

	// NewAsymmetricCipherFromPublicKey creates a new AsymmetricCipher for encryption from a serialized public key
	NewAsymmetricCipherFromSerializedPublicKey(pub []byte) (AsymmetricCipher, error)

	// NewAsymmetricCipherFromPublicKey creates a new AsymmetricCipher for encryption from a serialized public key
	NewAsymmetricCipherFromSerializedPrivateKey(priv []byte) (AsymmetricCipher, error)

	// NewPrivateKey creates a new private key rand and default parameters
	NewDefaultPrivateKey(rand io.Reader) (PrivateKey, error)

	// NewPrivateKey creates a new private key from (rand, params)
	// 从(rand, params)创建一个新的私钥
	NewPrivateKey(rand io.Reader, params interface{}) (PrivateKey, error)

	// NewPublicKey creates a new public key from (rand, params)
	//从(rand, params)创建一个新的公钥
	NewPublicKey(rand io.Reader, params interface{}) (PublicKey, error)

	// SerializePrivateKey serializes a private key
	// 序列化私钥
	SerializePrivateKey(priv PrivateKey) ([]byte, error)

	// DeserializePrivateKey deserializes to a private key
	// 反序列化私钥
	DeserializePrivateKey(bytes []byte) (PrivateKey, error)

	// SerializePrivateKey serializes a private key
	SerializePublicKey(pub PublicKey) ([]byte, error)

	// DeserializePrivateKey deserializes to a private key
	DeserializePublicKey(bytes []byte) (PublicKey, error)
}

// StreamCipherSPI is a Service Provider Interface for StreamCipher
type StreamCipherSPI interface {
	GenerateKey() (SecretKey, error)

	GenerateKeyAndSerialize() (SecretKey, []byte, error)

	NewSecretKey(rand io.Reader, params interface{}) (SecretKey, error)

	// NewStreamCipherForEncryptionFromKey creates a new StreamCipher for encryption from a secret key
	NewStreamCipherForEncryptionFromKey(secret SecretKey) (StreamCipher, error)

	// NewStreamCipherForEncryptionFromSerializedKey creates a new StreamCipher for encryption from a serialized key
	NewStreamCipherForEncryptionFromSerializedKey(secret []byte) (StreamCipher, error)

	// NewStreamCipherForDecryptionFromKey creates a new StreamCipher for decryption from a secret key
	NewStreamCipherForDecryptionFromKey(secret SecretKey) (StreamCipher, error)

	// NewStreamCipherForDecryptionFromKey creates a new StreamCipher for decryption from a serialized key
	NewStreamCipherForDecryptionFromSerializedKey(secret []byte) (StreamCipher, error)

	// SerializePrivateKey serializes a private key
	SerializeSecretKey(secret SecretKey) ([]byte, error)

	// DeserializePrivateKey deserializes to a private key
	DeserializeSecretKey(bytes []byte) (SecretKey, error)
}
