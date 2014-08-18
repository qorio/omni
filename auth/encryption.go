package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
)

type EncryptionService interface {
	Encrypt(input []byte) (encrypted []byte, err error)
	Decrypt(input []byte) (decrypted []byte, err error)
	EncryptString(input string) (encrypted string, err error)
	HmacSha256(input []byte) (h []byte)
	HmacSha256String(input string) (h string)
}

// Currently implemented by the Auth service -- yeah kinda weird, move it later.

func (this *serviceImpl) HmacSha256(input []byte) (h []byte) {
	mac := hmac.New(sha256.New, this.settings.SignKey)
	mac.Write(input)
	h = mac.Sum(nil)
	return
}

func (this *serviceImpl) HmacSha256String(input string) (h string) {
	buff := this.HmacSha256([]byte(input))
	return base64.StdEncoding.EncodeToString(buff)
}

func (this *serviceImpl) Encrypt(input []byte) (encrypted []byte, err error) {
	return encrypt(this.settings.SignKey, input)
}

func (this *serviceImpl) EncryptString(input string) (encrypted string, err error) {
	buff, err := this.Encrypt([]byte(input))
	if err == nil {
		encrypted = base64.StdEncoding.EncodeToString(buff)
	}
	return
}

func (this *serviceImpl) Decrypt(input []byte) (decrypted []byte, err error) {
	return decrypt(this.settings.SignKey, input)
}

func encrypt(key, text []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	b := base64.StdEncoding.EncodeToString(text)
	ciphertext := make([]byte, aes.BlockSize+len(b))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}
	cfb := cipher.NewCFBEncrypter(block, iv)
	cfb.XORKeyStream(ciphertext[aes.BlockSize:], []byte(b))
	return ciphertext, nil
}

func decrypt(key, text []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if len(text) < aes.BlockSize {
		return nil, errors.New("ciphertext too short")
	}
	iv := text[:aes.BlockSize]
	text = text[aes.BlockSize:]
	cfb := cipher.NewCFBDecrypter(block, iv)
	cfb.XORKeyStream(text, text)
	data, err := base64.StdEncoding.DecodeString(string(text))
	if err != nil {
		return nil, err
	}
	return data, nil
}
