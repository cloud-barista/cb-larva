package secutil

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cloud-barista/cb-larva/poc-cb-net/internal/file"
	cblog "github.com/cloud-barista/cb-log"
	"github.com/sirupsen/logrus"
)

// CBLogger represents a logger to show execution processes according to the logging level.
var CBLogger *logrus.Logger

func init() {
	fmt.Println("Start......... init() of secret-util.go")
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exePath := filepath.Dir(ex)
	fmt.Printf("exePath: %v\n", exePath)

	// Load cb-log config from the current directory (usually for the production)
	logConfPath := filepath.Join(exePath, "config", "log_conf.yaml")
	fmt.Printf("logConfPath: %v\n", logConfPath)
	if !file.Exists(logConfPath) {
		// Load cb-log config from the project directory (usually for development)
		path, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
		if err != nil {
			panic(err)
		}
		projectPath := strings.TrimSpace(string(path))
		logConfPath = filepath.Join(projectPath, "poc-cb-net", "config", "log_conf.yaml")
	}
	CBLogger = cblog.GetLoggerWithConfigPath("cb-network", logConfPath)
	CBLogger.Debugf("Load %v", logConfPath)
	fmt.Println("End......... init() of secret-util.go")
}

const (
	rsaKeySize = 2048
)

// GenerateRSAKey generates a pair of RSA private and public keys.
func GenerateRSAKey() (*rsa.PrivateKey, *rsa.PublicKey, error) {
	CBLogger.Debug("Start.........")

	// Generate RSA key
	privateKey, err := rsa.GenerateKey(rand.Reader, rsaKeySize)
	if err != nil {
		return nil, nil, err
	}

	CBLogger.Debug("End.........")
	return privateKey, &privateKey.PublicKey, nil
}

// RSAKeyToBytes converts a pair of RSA private and public keys to []byte.
func RSAKeyToBytes(privateKey *rsa.PrivateKey, publicKey *rsa.PublicKey) ([]byte, []byte, error) {
	CBLogger.Debug("Start.........")
	privateKeyBytes, err := PrivateKeyToBytes(privateKey)
	if err != nil {
		return nil, nil, err
	}

	publicKeyBytes, err := PublicKeyToBytes(publicKey)
	if err != nil {
		return nil, nil, err
	}

	CBLogger.Debug("End.........")
	return privateKeyBytes, publicKeyBytes, nil
}

// PrivateKeyToBytes converts a pair of RSA private key to []byte.
func PrivateKeyToBytes(privateKey *rsa.PrivateKey) ([]byte, error) {
	CBLogger.Debug("Start.........")

	privateKeyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return nil, err
	}
	CBLogger.Tracef("privateKeyBytes: %+v", privateKeyBytes)

	CBLogger.Debug("End.........")
	return privateKeyBytes, nil
}

// PublicKeyToBytes converts a pair of RSA public key to []byte.
func PublicKeyToBytes(publicKey *rsa.PublicKey) ([]byte, error) {
	CBLogger.Debug("Start.........")

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return nil, err
	}
	CBLogger.Tracef("publicKeyBytes: %+v", publicKeyBytes)

	CBLogger.Debug("End.........")
	return publicKeyBytes, nil
}

// SaveRSAKeyToFile saves a pair of RSA private and public keys to each key file.
func SaveRSAKeyToFile(privateKeyBytes []byte, pemPath string, publicKeyBytes []byte, pubPath string) error {
	CBLogger.Debug("Start.........")

	if err := SavePrivateKeyToFile(privateKeyBytes, pemPath); err != nil {
		return err
	}

	if err := SavePublicKeyToFile(publicKeyBytes, pubPath); err != nil {
		return err
	}

	CBLogger.Debug("End.........")
	return nil
}

// SavePrivateKeyToFile saves a RSA private key to a key file.
func SavePrivateKeyToFile(privateKeyBytes []byte, pemPath string) error {
	CBLogger.Debug("Start.........")

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

	CBLogger.Debug("End.........")
	return nil
}

// SavePublicKeyToFile saves a RSA public key to a key file.
func SavePublicKeyToFile(publicKeyBytes []byte, pubPath string) error {
	CBLogger.Debug("Start.........")

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

	CBLogger.Debug("End.........")
	return nil
}

// LoadPrivateKeyFromFile loads a RSA private key from a key file.
func LoadPrivateKeyFromFile(pemPath string) (*rsa.PrivateKey, error) {
	CBLogger.Debug("Start.........")

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

	CBLogger.Debug("End.........")
	return PrivateKeyFromBytes(privateKeyPem.Bytes)
}

// LoadPublicKeyFromFile loads a RSA public key from a key file.
func LoadPublicKeyFromFile(pubPath string) (*rsa.PublicKey, error) {
	CBLogger.Debug("Start.........")

	// Load public key from file
	publicKeyBytes, err := ioutil.ReadFile(pubPath)
	if err != nil {
		return nil, err
	}
	publicKeyPem, _ := pem.Decode(publicKeyBytes)
	if publicKeyPem == nil || publicKeyPem.Type != "RSA PUBLIC KEY" {
		return nil, errors.New("failed to decode PEM block containing public key")
	}

	CBLogger.Debug("End.........")
	return PublicKeyFromBytes(publicKeyPem.Bytes)
}

// PublicKeyToBase64 convert a RSA public key to a base64 string.
func PublicKeyToBase64(publicKey *rsa.PublicKey) (string, error) {
	CBLogger.Debug("Start.........")
	publicKeyBytes, err := PublicKeyToBytes(publicKey)
	if err != nil {
		return "", err
	}

	CBLogger.Debug("End.........")
	return base64.StdEncoding.EncodeToString(publicKeyBytes), nil
}

// PublicKeyFromBase64 convert a base64 string to a RSA public key.
func PublicKeyFromBase64(key string) (*rsa.PublicKey, error) {
	CBLogger.Debug("Start.........")

	publicKeyBytes, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return nil, err
	}

	CBLogger.Debug("End.........")
	return PublicKeyFromBytes(publicKeyBytes)
}

// PublicKeyFromBytes convert a base64 bytes to a RSA public key.
func PublicKeyFromBytes(publicKeyBytes []byte) (*rsa.PublicKey, error) {
	CBLogger.Debug("Start.........")

	publicKeyInterface, err := x509.ParsePKIXPublicKey(publicKeyBytes)
	if err != nil {
		return nil, err
	}
	publicKey, ok := publicKeyInterface.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("invalid public key")
	}

	CBLogger.Debug("End.........")
	return publicKey, nil
}

// PrivateKeyFromBytes convert a base64 bytes to a RSA private key.
func PrivateKeyFromBytes(privateKeyBytes []byte) (*rsa.PrivateKey, error) {
	CBLogger.Debug("Start.........")

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

	CBLogger.Debug("End.........")
	return privateKey, nil
}
