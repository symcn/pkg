package selfsigned

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"
)

var (
	rsaKeySize = 4096
	validity   = time.Hour * 24 * 365 * 10

	_ Signer = &signer{}
)

type signer struct {
	privKey crypto.Signer
	cert    *x509.Certificate
	csr     *x509.CertificateRequest
}

func NewSelfSigner() (Signer, error) {
	privKey, err := rsa.GenerateKey(rand.Reader, rsaKeySize)
	if err != nil {
		return nil, fmt.Errorf("RSA key generation failed: %+v", err)
	}

	return &signer{
		privKey: privKey,
	}, nil
}

// PrivateKey implements authentication.Signer
func (s *signer) PrivateKey() []byte {
	return encodePemPrivKey(s.privKey)
}

// GenCSR implements authentication.Signer
func (s *signer) GenCSR(opts *CertOptions) (csrRaw []byte, err error) {
	tmpl := &x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName: opts.CommonName,
		},
		DNSNames:           opts.DNSNames,
		SignatureAlgorithm: sigType(s.privKey),
	}
	csrRaw, err = x509.CreateCertificateRequest(rand.Reader, tmpl, s.privKey)
	if err != nil {
		return nil, fmt.Errorf("create CertificateRequest failed: %+v", err)
	}

	s.csr, err = x509.ParseCertificateRequest(csrRaw)
	if err != nil {
		return nil, fmt.Errorf("parse CertificateRequest failed: %+v", err)
	}
	return encodePemCSRWithRaw(csrRaw), nil
}

// GenCert implements authentication.Signer
func (s *signer) GenCert(opts *CertOptions) (certRaw []byte, err error) {
	now := time.Now()
	tmpl := &x509.Certificate{
		SerialNumber: new(big.Int).SetInt64(0),
		Subject: pkix.Name{
			CommonName: opts.CommonName,
		},
		DNSNames:              opts.DNSNames,
		NotBefore:             now.UTC(),
		NotAfter:              now.Add(validity).UTC(),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	certRaw, err = x509.CreateCertificate(rand.Reader, tmpl, tmpl, s.privKey.Public(), s.privKey)
	if err != nil {
		return nil, fmt.Errorf("create Certificate failed: %+v", err)
	}

	s.cert, err = x509.ParseCertificate(certRaw)
	if err != nil {
		return nil, fmt.Errorf("parse Certificate failed: %+v", err)
	}
	return encodePemCertWithRaw(certRaw), nil
}

// Sign implements authentication.Signer
func (s *signer) Sign(csrRaw []byte, expireTime time.Duration) (certRaw []byte, err error) {
	if s.cert == nil {
		return nil, fmt.Errorf("signer's Certificate is empty, must call GenCert first")
	}

	p, _ := pem.Decode(csrRaw)
	csr, err := x509.ParseCertificateRequest(p.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse CSR(%s) failed: %+v", string(csrRaw), err)
	}

	tmpl := &x509.Certificate{
		SerialNumber:       s.cert.SerialNumber,
		Subject:            csr.Subject,
		DNSNames:           csr.DNSNames,
		IPAddresses:        csr.IPAddresses,
		EmailAddresses:     csr.EmailAddresses,
		URIs:               csr.URIs,
		PublicKeyAlgorithm: csr.PublicKeyAlgorithm,
		PublicKey:          csr.PublicKey,
		Extensions:         csr.Extensions,
		ExtraExtensions:    csr.ExtraExtensions,
		IsCA:               false,
		KeyUsage:           x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
			x509.ExtKeyUsageClientAuth,
		},
	}
	now := time.Now()
	tmpl.NotBefore = now
	tmpl.NotAfter = now.Add(expireTime)

	certRaw, err = x509.CreateCertificate(rand.Reader, tmpl, s.cert, csr.PublicKey, s.privKey)
	if err != nil {
		return nil, fmt.Errorf("sign Certificate failed: %+v", err)
	}
	return encodePemCertWithRaw(certRaw), nil
}

func sigType(privateKey interface{}) x509.SignatureAlgorithm {
	// Customize the signature for RSA keys, depending on the key size
	if privateKey, ok := privateKey.(*rsa.PrivateKey); ok {
		keySize := privateKey.N.BitLen()
		switch {
		case keySize >= 4096:
			return x509.SHA512WithRSA
		case keySize >= 3072:
			return x509.SHA384WithRSA
		default:
			return x509.SHA256WithRSA
		}
	}
	return x509.UnknownSignatureAlgorithm
}
