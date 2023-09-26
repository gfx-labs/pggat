package certs

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"time"
)

func SelfSign() (tls.Certificate, error) {
	// generate private key
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}

	keyUsage := x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment

	notBefore := time.Now()
	notAfter := notBefore.Add(3 * 30 * 24 * time.Hour)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return tls.Certificate{}, err
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"GFX Labs"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              keyUsage,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	// TODO(garet)
	template.IPAddresses = append(template.IPAddresses, net.ParseIP("127.0.0.1"))

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return tls.Certificate{}, err
	}

	var cert tls.Certificate
	cert.PrivateKey = priv
	cert.Certificate = append(cert.Certificate, derBytes)

	return cert, nil
}
