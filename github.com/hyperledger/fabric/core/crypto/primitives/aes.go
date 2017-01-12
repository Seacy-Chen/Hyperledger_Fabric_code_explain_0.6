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
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
)

const (
	// AESKeyLength is the default AES key length
	// AES 秘钥的默认长度
	AESKeyLength = 32

	// NonceSize is the default NonceSize
	// 默认nonce大小
	NonceSize = 24
)

// GenAESKey returns a random AES key of length AESKeyLength
// 返回一个长度AESKeyLength为随机 AES 密钥
func GenAESKey() ([]byte, error) {
	return GetRandomBytes(AESKeyLength)
}

// PKCS7Padding pads as prescribed by the PKCS7 standard
// //基于PKCS7标准填充
func PKCS7Padding(src []byte) []byte {
	padding := aes.BlockSize - len(src)%aes.BlockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(src, padtext...)
}

// PKCS7UnPadding unpads as prescribed by the PKCS7 standard
//基于PKCS7标准反填充
func PKCS7UnPadding(src []byte) ([]byte, error) {
	length := len(src)
	unpadding := int(src[length-1])

	if unpadding > aes.BlockSize || unpadding == 0 {
		return nil, fmt.Errorf("invalid padding")
	}

	pad := src[len(src)-unpadding:]
	for i := 0; i < unpadding; i++ {
		if pad[i] != byte(unpadding) {
			return nil, fmt.Errorf("invalid padding")
		}
	}

	return src[:(length - unpadding)], nil
}

// CBCEncrypt encrypts using CBC mode
//使用CBC模式加密
func CBCEncrypt(key, s []byte) ([]byte, error) {
	// CBC mode works on blocks so plaintexts may need to be padded to the
	// next whole block. For an example of such padding, see
	// https://tools.ietf.org/html/rfc5246#section-6.2.3.2. Here we'll
	// assume that the plaintext is already of the correct length.
	// CBC模式适用于块，这样的明文可能需要填充到下一个整块。对于这种填充的一个实例，参见
	// https://tools.ietf.org/html/rfc5246#section-6.2.3.2. Here we'll
	// 假定所述明文已确定长度。
	if len(s)%aes.BlockSize != 0 {
		return nil, errors.New("plaintext is not a multiple of the block size")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// The IV needs to be unique, but not secure. Therefore it's common to
	// include it at the beginning of the ciphertext.
	// 需要唯一，但并不安全。因此它是常见的包括其在密文的开始
	ciphertext := make([]byte, aes.BlockSize+len(s))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext[aes.BlockSize:], s)

	// It's important to remember that ciphertexts must be authenticated
	// (i.e. by using crypto/hmac) as well as being encrypted in order to
	// be secure.
	// 要记住，密文必须经过验证是非常重要的（即，通过使用加密/ HMAC）以及为了被加密是安全的。
	return ciphertext, nil
}

// CBCDecrypt decrypts using CBC mode
// 使用CBC模式解密
func CBCDecrypt(key, src []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// The IV needs to be unique, but not secure. Therefore it's common to
	// include it at the beginning of the ciphertext.
	// 需要唯一，但并不安全。因此它是常见的包括其在密文的开始。
	if len(src) < aes.BlockSize {
		return nil, errors.New("ciphertext too short")
	}
	iv := src[:aes.BlockSize]
	src = src[aes.BlockSize:]

	// CBC mode always works in whole blocks.
	// CBC模式往往工作于整个块中
	if len(src)%aes.BlockSize != 0 {
		return nil, errors.New("ciphertext is not a multiple of the block size")
	}

	mode := cipher.NewCBCDecrypter(block, iv)

	// CryptBlocks can work in-place if the two arguments are the same.
	//可以就地工作，如果两个参数是相同的
	mode.CryptBlocks(src, src)

	// If the original plaintext lengths are not a multiple of the block
	// size, padding would have to be added when encrypting, which would be
	// removed at this point. For an example, see
	// https://tools.ietf.org/html/rfc5246#section-6.2.3.2. However, it's
	// critical to note that ciphertexts must be authenticated (i.e. by
	// using crypto/hmac) before being decrypted in order to avoid creating
	// a padding oracle.
	// 如果原来的明文的长度不是块大小的倍数，加密填充时，这将在这一点被移除被添加

	return src, nil
}

// CBCPKCS7Encrypt combines CBC encryption and PKCS7 padding
// 结合CBC加密填充PKCS7
func CBCPKCS7Encrypt(key, src []byte) ([]byte, error) {
	return CBCEncrypt(key, PKCS7Padding(src))
}

// CBCPKCS7Decrypt combines CBC decryption and PKCS7 unpadding
//结合CBC加密反填充填充PKCS7
func CBCPKCS7Decrypt(key, src []byte) ([]byte, error) {
	pt, err := CBCDecrypt(key, src)
	if err != nil {
		return nil, err
	}

	original, err := PKCS7UnPadding(pt)
	if err != nil {
		return nil, err
	}

	return original, nil
}
