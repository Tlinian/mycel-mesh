package wireguard

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/curve25519"
)

// GenerateKey 生成 WireGuard 密钥对
// 返回私钥、公钥和错误
// 私钥和公钥都是 base64 编码的 32 字节
func GenerateKey() (privateKey, publicKey string, err error) {
	// 生成随机私钥 (32 字节)
	var privKey [32]byte
	if _, err := rand.Read(privKey[:]); err != nil {
		return "", "", fmt.Errorf("failed to generate private key: %w", err)
	}

	// 清除最高位和最低位以符合 Curve25519 要求
	privKey[0] &= 248
	privKey[31] &= 127
	privKey[31] |= 64

	// 从私钥推导公钥
	pubKey, err := curve25519.X25519(privKey[:], curve25519.Basepoint)
	if err != nil {
		return "", "", fmt.Errorf("failed to compute public key: %w", err)
	}

	// Base64 编码
	privateKey = base64.StdEncoding.EncodeToString(privKey[:])
	publicKey = base64.StdEncoding.EncodeToString(pubKey[:])

	return privateKey, publicKey, nil
}
