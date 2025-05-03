package main

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"log"

	"github.com/mark3labs/mcp-go/mcp"
)

// SecurityTool 提供安全加密功能
type SecurityTool struct{}

// AESEncrypt AES加密
func (s *SecurityTool) AESEncrypt(key, plaintext string) (string, error) {
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		log.Printf("AES加密创建cipher失败: %v", err)
		return "", err
	}

	plaintextBytes := []byte(plaintext)
	ciphertext := make([]byte, aes.BlockSize+len(plaintextBytes))
	iv := ciphertext[:aes.BlockSize]
	if _, err := rand.Read(iv); err != nil {
		log.Printf("AES加密生成IV失败: %v", err)
		return "", err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], plaintextBytes)

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// AESDecrypt AES解密
func (s *SecurityTool) AESDecrypt(key, ciphertext string) (string, error) {
	ciphertextBytes, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		log.Printf("AES解密base64解码失败: %v", err)
		return "", err
	}

	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		log.Printf("AES解密创建cipher失败: %v", err)
		return "", err
	}

	if len(ciphertextBytes) < aes.BlockSize {
		return "", errors.New("ciphertext too short")
	}

	iv := ciphertextBytes[:aes.BlockSize]
	ciphertextBytes = ciphertextBytes[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(ciphertextBytes, ciphertextBytes)

	return string(ciphertextBytes), nil
}

// GenerateRSAKeyPair 生成RSA密钥对
func (s *SecurityTool) GenerateRSAKeyPair(bits int) (*rsa.PrivateKey, *rsa.PublicKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		log.Printf("生成RSA密钥对失败: %v", err)
		return nil, nil, err
	}
	return privateKey, &privateKey.PublicKey, nil
}

// RSAEncrypt RSA加密
func (s *SecurityTool) RSAEncrypt(publicKey *rsa.PublicKey, plaintext string) (string, error) {
	ciphertext, err := rsa.EncryptPKCS1v15(rand.Reader, publicKey, []byte(plaintext))
	if err != nil {
		log.Printf("RSA加密失败: %v", err)
		return "", err
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// RSADecrypt RSA解密
func (s *SecurityTool) RSADecrypt(privateKey *rsa.PrivateKey, ciphertext string) (string, error) {
	ciphertextBytes, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		log.Printf("RSA解密base64解码失败: %v", err)
		return "", err
	}

	plaintext, err := rsa.DecryptPKCS1v15(rand.Reader, privateKey, ciphertextBytes)
	if err != nil {
		log.Printf("RSA解密失败: %v", err)
		return "", err
	}
	return string(plaintext), nil
}

// MD5Hash MD5哈希计算
func (s *SecurityTool) MD5Hash(data string) string {
	hash := md5.Sum([]byte(data))
	return hex.EncodeToString(hash[:])
}

// SHA256Hash SHA256哈希计算
func (s *SecurityTool) SHA256Hash(data string) string {
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// GenerateRandomBytes 生成随机字节
func (s *SecurityTool) GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		log.Printf("生成随机字节失败: %v", err)
		return nil, err
	}
	return b, nil
}

// SecurityToolHandler 处理安全工具请求
func SecurityToolHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tool := &SecurityTool{}
	action := request.Params.Arguments["action"].(string)
	algorithm := request.Params.Arguments["algorithm"].(string)
	data := request.Params.Arguments["data"].(string)
	key := ""
	if k, ok := request.Params.Arguments["key"]; ok {
		key = k.(string)
	}

	switch action {
	case "encrypt":
		switch algorithm {
		case "aes":
			result, err := tool.AESEncrypt(key, data)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(result), nil
		case "rsa":
			block, _ := pem.Decode([]byte(key))
			if block == nil {
				return nil, errors.New("failed to parse PEM block containing the public key")
			}

			pub, err := x509.ParsePKIXPublicKey(block.Bytes)
			if err != nil {
				return nil, err
			}

			publicKey := pub.(*rsa.PublicKey)
			result, err := tool.RSAEncrypt(publicKey, data)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(result), nil
		default:
			return nil, fmt.Errorf("不支持的加密算法: %s", algorithm)
		}
	case "decrypt":
		switch algorithm {
		case "aes":
			result, err := tool.AESDecrypt(key, data)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(result), nil
		case "rsa":
			block, _ := pem.Decode([]byte(key))
			if block == nil {
				return nil, errors.New("failed to parse PEM block containing the private key")
			}

			privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
			if err != nil {
				return nil, err
			}

			result, err := tool.RSADecrypt(privateKey, data)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(result), nil
		default:
			return nil, fmt.Errorf("不支持的解密算法: %s", algorithm)
		}
	case "hash":
		switch algorithm {
		case "md5":
			return mcp.NewToolResultText(tool.MD5Hash(data)), nil
		case "sha256":
			return mcp.NewToolResultText(tool.SHA256Hash(data)), nil
		default:
			return nil, fmt.Errorf("不支持的哈希算法: %s", algorithm)
		}
	default:
		return nil, fmt.Errorf("不支持的操作: %s", action)
	}
}
