package secutil

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"io/ioutil"
	"os"
)

const (
	rsaKeySize = 2048
)

func GenerateRSAKey() (*rsa.PrivateKey, *rsa.PublicKey, error) {

	// Generate RSA key
	privateKey, err := rsa.GenerateKey(rand.Reader, rsaKeySize)
	if err != nil {
		return nil, nil, err
	}
	return privateKey, &privateKey.PublicKey, nil
}

func RSAKeyToBytes(privateKey *rsa.PrivateKey, publicKey *rsa.PublicKey) ([]byte, []byte, error) {
	privateKeyBytes, err := PrivateKeyToBytes(privateKey)
	if err != nil {
		return nil, nil, err
	}

	publicKeyBytes, err := PublicKeyToBytes(publicKey)
	if err != nil {
		return nil, nil, err
	}

	return privateKeyBytes, publicKeyBytes, nil
}

func PrivateKeyToBytes(privateKey *rsa.PrivateKey) ([]byte, error) {
	privateKeyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return nil, err
	}
	return privateKeyBytes, nil
}

func PublicKeyToBytes(publicKey *rsa.PublicKey) ([]byte, error) {
	publicKeyBytes, err := x509.MarshalPKCS8PrivateKey(publicKey)
	if err != nil {
		return nil, err
	}
	return publicKeyBytes, nil
}

func SaveRSAKeyToFile(privateKeyBytes []byte, pemPath string, publicKeyBytes []byte, pubPath string) error {

	if err := SavePrivateKeyToFile(privateKeyBytes, pemPath); err != nil {
		return err
	}

	if err := SavePublicKeyToFile(publicKeyBytes, pubPath); err != nil {
		return err
	}

	return nil
}

func SavePrivateKeyToFile(privateKeyBytes []byte, pemPath string) error {
	// Save private key to file
	privateKeyBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	}
	privatePem, err := os.Create(pemPath)
	if err != nil {
		return err
	}
	err = pem.Encode(privatePem, privateKeyBlock)
	if err != nil {
		return err
	}
	return nil
}

func SavePublicKeyToFile(publicKeyBytes []byte, pubPath string) error {

	// Save public key to file
	publicKeyBlock := &pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: publicKeyBytes,
	}
	publicPem, err := os.Create(pubPath)
	if err != nil {
		return err
	}

	err = pem.Encode(publicPem, publicKeyBlock)
	if err != nil {
		return err
	}

	return nil
}

func LoadPrivateKeyFromFile(pemPath string) (*rsa.PrivateKey, error) {
	// Load private key from file
	privateKeyBytes, err := ioutil.ReadFile(pemPath)
	if err != nil {
		return nil, err
	}

	privateKeyPem, _ := pem.Decode(privateKeyBytes)
	if privateKeyPem == nil || privateKeyPem.Type != "RSA PRIVATE KEY" {
		return nil, errors.New("failed to decode PEM block containing private key")
	}

	// Currently no need of password

	// var privPemBytes []byte

	// rsaPrivateKeyPassword := "" // Currently no need of password
	// if rsaPrivateKeyPassword != "" {
	// 	privPemBytes, err = x509.DecryptPEMBlock(privateKeyPem, []byte(rsaPrivateKeyPassword))
	// } else {
	// 	privPemBytes = privateKeyPem.Bytes
	// }

	return PrivateKeyFromBytes(privateKeyPem.Bytes)
}

func LoadPublicKeyFromFile(pubPath string) (*rsa.PublicKey, error) {
	// Load public key from file
	publicKeyBytes, err := ioutil.ReadFile(pubPath)
	if err != nil {
		return nil, err
	}
	publicKeyPem, _ := pem.Decode(publicKeyBytes)
	if publicKeyPem == nil || publicKeyPem.Type != "RSA PUBLIC KEY" {
		return nil, errors.New("failed to decode PEM block containing public key")
	}

	return PublicKeyFromBytes(publicKeyPem.Bytes)
}

func PublicKeyToBase64(publicKey *rsa.PublicKey) (string, error) {
	publicKeyBytes, err := PublicKeyToBytes(publicKey)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(publicKeyBytes), nil
}

func PublicKeyFromBase64(key string) (*rsa.PublicKey, error) {
	publicKeyBytes, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return nil, err
	}
	return PublicKeyFromBytes(publicKeyBytes)
}

func PublicKeyFromBytes(publicKeyBytes []byte) (*rsa.PublicKey, error) {
	publicKeyInterface, err := x509.ParsePKIXPublicKey(publicKeyBytes)
	if err != nil {
		return nil, err
	}
	publicKey, ok := publicKeyInterface.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("invalid public key")
	}
	return publicKey, nil
}

func PrivateKeyFromBytes(privateKeyBytes []byte) (*rsa.PrivateKey, error) {

	var privateKeyInterface interface{}
	var err error
	if privateKeyInterface, err = x509.ParsePKCS1PrivateKey(privateKeyBytes); err != nil {
		if privateKeyInterface, err = x509.ParsePKCS8PrivateKey(privateKeyBytes); err != nil {
			return nil, err
		}
	}
	privateKey, ok := privateKeyInterface.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("invalid private key")
	}
	return privateKey, nil
}
