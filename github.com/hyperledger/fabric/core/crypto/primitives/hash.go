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
	"crypto/hmac"
	"hash"
)

var (
	defaultHash          func() hash.Hash
	defaultHashAlgorithm string
)

// GetDefaultHash returns the default hash function used by the crypto layer
// 返回加密层中使用的默认哈希值
func GetDefaultHash() func() hash.Hash {
	return defaultHash
}

// GetHashAlgorithm return the default hash algorithm
func GetHashAlgorithm() string {
	return defaultHashAlgorithm
}

// NewHash returns a new hash function
// 返回一个新的散列函数
func NewHash() hash.Hash {
	return GetDefaultHash()()
}

// Hash hashes the msh using the predefined hash function
// 散列使用预定义散列函数的MSH
func Hash(msg []byte) []byte {
	hash := NewHash()
	hash.Write(msg)
	return hash.Sum(nil)
}

// HMAC hmacs x using key key
// hmacs x 使用密钥的密钥
func HMAC(key, x []byte) []byte {
	mac := hmac.New(GetDefaultHash(), key)
	mac.Write(x)

	return mac.Sum(nil)
}

// HMACTruncated hmacs x using key key and truncate to truncation
// hmacs x 使用密钥的密钥，并截断
func HMACTruncated(key, x []byte, truncation int) []byte {
	mac := hmac.New(GetDefaultHash(), key)
	mac.Write(x)

	return mac.Sum(nil)[:truncation]
}

// HMACAESTruncated hmacs x using key key and truncate to AESKeyLength
func HMACAESTruncated(key, x []byte) []byte {
	return HMACTruncated(key, x, AESKeyLength)
}
