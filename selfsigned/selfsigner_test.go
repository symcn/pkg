package selfsigned

import (
	"fmt"
	"testing"
	"time"
)

func TestSign(t *testing.T) {
	rootSigner, err := NewSelfSigner()
	if err != nil {
		t.Error(err)
		return
	}

	rootCert, err := rootSigner.GenCert(&CertOptions{
		CommonName: "*.sym-admin.svc",
		DNSNames:   []string{"*.sym-admin.svc"},
	})
	if err != nil {
		t.Error(err)
		return
	}

	svcSigner, err := NewSelfSigner()
	if err != nil {
		t.Error(err)
		return
	}
	// err = svcSigner.GenCSR("sym-control-webhook.sym-admin.svc")
	svcCSR, err := svcSigner.GenCSR(&CertOptions{
		CommonName: "sym-control-webhook.sym-admin.svc",
		DNSNames:   []string{"sym-control-webhook.sym-admin.svc"},
	})
	if err != nil {
		t.Error(err)
		return
	}

	signedCert, err := rootSigner.Sign(svcCSR, time.Hour*24*7)
	if err != nil {
		t.Error(err)
		return
	}

	fmt.Println(string(rootCert))
	fmt.Println(string(signedCert))
	fmt.Println(string(svcSigner.PrivateKey()))
}
