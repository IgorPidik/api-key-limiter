package proxy

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"os"
)

func Decrypt(key []byte, hexData string) (string, error) {
	data, decodeErr := hex.DecodeString(hexData)
	if decodeErr != nil {
		return "", nil
	}

	blockCipher, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(blockCipher)
	if err != nil {
		return "", err
	}

	nonce, ciphertext := data[:gcm.NonceSize()], data[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

func DecryptData(data string) (string, error) {
	secretKeyHex := os.Getenv("SECRET_KEY")
	secretKey, decodeErr := hex.DecodeString(secretKeyHex)
	if decodeErr != nil {
		return "", decodeErr
	}
	return Decrypt(secretKey, data)
}
