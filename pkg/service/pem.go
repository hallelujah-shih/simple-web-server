package service

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"sync"
	"sync/atomic"
	"time"
)

var (
	sn sNumber
)

type sNumber struct {
	num int64
}

func (sn *sNumber) GetSerialNumber() int64 {
	return atomic.AddInt64(&sn.num, 1)
}

type PemManager struct {
	lock     sync.RWMutex
	root     *tls.Certificate
	rootx509 *x509.Certificate
	caCache  map[string]*tls.Certificate
}

func NewPemManager() *PemManager {
	pm := &PemManager{
		caCache: make(map[string]*tls.Certificate),
	}
	pm.GetRoot()
	return pm
}

func (pm *PemManager) GetRoot() *tls.Certificate {
	pm.lock.Lock()
	defer pm.lock.Unlock()
	if pm.root == nil {
		if r, rx509, err := pm.generateRootCA(); err != nil {
			panic(err)
		} else {
			pm.root = r
			pm.rootx509 = rx509
		}
	}
	return pm.root
}

func (pm *PemManager) GetCertificate(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
	pm.lock.Lock()
	defer pm.lock.Unlock()
	cert := pm.caCache[info.ServerName]
	if cert == nil {
		rt, err := pm.generateCertificate(info.ServerName)
		if err == nil {
			pm.caCache[info.ServerName] = rt
		}
		return rt, err
	}
	return cert, nil
}

func (pm *PemManager) generateRootCA() (*tls.Certificate, *x509.Certificate, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	template := &x509.Certificate{
		SerialNumber:          big.NewInt(sn.GetSerialNumber()),
		Subject:               pkix.Name{CommonName: "shih simple web service ca"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		BasicConstraintsValid: true,
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return nil, nil, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})

	root, err := tls.X509KeyPair(certPEM, keyPEM)
	return &root, template, err
}

func (pm *PemManager) generateCertificate(sni string) (*tls.Certificate, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(sn.GetSerialNumber()),
		Subject:      pkix.Name{CommonName: sni},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(1, 0, 0),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		DNSNames:     []string{sni},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, template, pm.rootx509, &key.PublicKey, pm.root.PrivateKey)
	if err != nil {
		return nil, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	return &cert, err
}
