package authentication

import (
	"time"

	"github.com/symcn/pkg/selfsigned"
)

func BuildWebhookCertInfoWithSelf(rootOpts, svcOpts *selfsigned.CertOptions, expireTime time.Duration) (*CertInfo, error) {
	// build root cert
	rootSigner, err := selfsigned.NewSelfSigner()
	if err != nil {
		return nil, err
	}
	rootCert, err := rootSigner.GenCert(rootOpts)
	if err != nil {
		return nil, err
	}

	// build svc csr
	svcSigner, err := selfsigned.NewSelfSigner()
	if err != nil {
		return nil, err
	}
	svcCSR, err := svcSigner.GenCSR(svcOpts)
	if err != nil {
		return nil, err
	}

	signedCert, err := rootSigner.Sign(svcCSR, expireTime)
	if err != nil {
		return nil, err
	}

	return &CertInfo{
		CABundle: rootCert,
		TLSKey:   svcSigner.PrivateKey(),
		TLSCert:  signedCert,
	}, nil
}
