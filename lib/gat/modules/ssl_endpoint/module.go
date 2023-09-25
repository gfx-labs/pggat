package ssl_endpoint

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"time"

	"tuxpa.in/a/zlog/log"

	"pggat/lib/gat"
	"pggat/lib/util/strutil"
)

type Module struct {
	config *tls.Config
}

func NewModule() (*Module, error) {
	return &Module{}, nil
}

func (T *Module) generateKeys() error {
	// generate private key
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	keyUsage := x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment

	notBefore := time.Now()
	notAfter := notBefore.Add(3 * 30 * 24 * time.Hour)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return err
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
	template.IPAddresses = append(template.IPAddresses, net.ParseIP("192.168.1.1"))

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return err
	}

	var cert tls.Certificate
	cert.PrivateKey = priv
	cert.Certificate = append(cert.Certificate, derBytes)

	T.config = &tls.Config{
		Certificates: []tls.Certificate{
			cert,
		},
	}
	return nil
}

func (T *Module) GatModule() {}

func (T *Module) Endpoints() []gat.Endpoint {
	if T.config == nil {
		if err := T.generateKeys(); err != nil {
			log.Printf("failed to generate ssl certificate: %v", err)
		}
	}

	return []gat.Endpoint{
		{
			Network: "tcp",
			Address: ":5432",
			AcceptOptions: gat.FrontendAcceptOptions{
				SSLRequired: false,
				SSLConfig:   T.config,
				AllowedStartupOptions: []strutil.CIString{
					strutil.MakeCIString("client_encoding"),
					strutil.MakeCIString("datestyle"),
					strutil.MakeCIString("timezone"),
					strutil.MakeCIString("standard_conforming_strings"),
					strutil.MakeCIString("application_name"),
					strutil.MakeCIString("extra_float_digits"),
					strutil.MakeCIString("options"),
				},
			},
		},
	}
}

var _ gat.Module = (*Module)(nil)
var _ gat.Listener = (*Module)(nil)
