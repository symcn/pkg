package selfsigned

import "time"

type Signer interface {
	// GenCSR generate CERTIFICATE REQUEST
	GenCSR(opts *CertOptions) (csrRaw []byte, err error)

	// GenCert generate CERTIFICATE
	GenCert(opts *CertOptions) (certRaw []byte, err error)

	// Sign use self sign csr, return new cert, must invoke GenCert first, otherwise return error
	Sign(csrRaw []byte, expireTime time.Duration) (certRaw []byte, err error)

	// PrivateKey get PRIVATE KEY
	PrivateKey() []byte
}

// CertOptions contains options for generating a new certificate.
type CertOptions struct {
	CommonName string
	DNSNames   []string
}
