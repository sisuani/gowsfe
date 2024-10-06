import "github.com/sisuani/gowsfe/pkg/certs"

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"

	"go.mozilla.org/pkcs7"
)

func EncodeCMS(content []byte, certificate *x509.Certificate, privateKey *rsa.PrivateKey) ([]byte, error) {
	signedData, err := pkcs7.NewSignedData(content)
	if err != nil {
		return nil, fmt.Errorf("encodeCMS: failied to initialize SignedData. %s", err)
	}

	if err := signedData.AddSigner(certificate, privateKey, pkcs7.SignerInfoConfig{}); err != nil {
		return nil, fmt.Errorf("encodeCMS: unable to add signer: %s", err)
	}

	detachedSignature, err := signedData.Finish()
	if err != nil {
		return nil, fmt.Errorf("encodeCMS: unable to finish signature: %s", err)
	}

	return detachedSignature, nil
}

func LoadX509KeyPair(certFile, keyFile string) (*x509.Certificate, *rsa.PrivateKey, error) {
	certData, err := ioutil.ReadFile(certFile)
	if err != nil {
		return nil, nil, fmt.Errorf("LoadX509KeyPair: crt file not found: %s", err)
	}

	keyData, err := ioutil.ReadFile(keyFile)
	if err != nil {
		return nil, nil, fmt.Errorf("LoadX509KeyPair: key file not found: %s", err)
	}

	certDecode, _ := pem.Decode(certData)
	if certData == nil {
		return nil, nil, fmt.Errorf("LoadX509KeyPair: could not decode crt data: %s", err)
	}
	keyDecode, _ := pem.Decode(keyData)
	if keyDecode == nil {
		return nil, nil, fmt.Errorf("LoadX509KeyPair: could not decode key data: %s", err)
	}

	crt, err := x509.ParseCertificate(certDecode.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("LoadX509KeyPair: could not parse crt data: %s", err)
	}

	key, err := x509.ParsePKCS1PrivateKey(keyDecode.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("LoadX509KeyPair: could not parse key data: %s", err)
	}
	return crt, key, nil
}
