package selfsigned

import (
	"crypto"
	"crypto/x509"
	"encoding/pem"
)

const (
	// CertificateBlockType is a possible value for pem.Block.Type.
	CertificateBlockType = "CERTIFICATE"
	// CertificateRequestBlockType is a possible value for pem.Block.Type.
	CertificateRequestBlockType = "CERTIFICATE REQUEST"
	// PrivateKeyBlockType is a possible value for pem.Block.Type.
	PrivateKeyBlockType = "PRIVATE KEY"
)

func encodePemPrivKey(privateKey crypto.Signer) []byte {
	rawKeyData, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		panic(err)
	}

	return pem.EncodeToMemory(&pem.Block{
		Type:  PrivateKeyBlockType,
		Bytes: rawKeyData,
	})
}

func encodePemCert(cert *x509.Certificate) []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:  CertificateBlockType,
		Bytes: cert.Raw,
	})
}

func encodePemCSR(csr *x509.CertificateRequest) []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:  CertificateRequestBlockType,
		Bytes: csr.Raw,
	})
}

func encodePemPrivKeyWithRaw(raw []byte) []byte {
	return encodePemWithRaw(PrivateKeyBlockType, raw)
}

func encodePemCertWithRaw(raw []byte) []byte {
	return encodePemWithRaw(CertificateBlockType, raw)
}

func encodePemCSRWithRaw(raw []byte) []byte {
	return encodePemWithRaw(CertificateRequestBlockType, raw)
}

func encodePemWithRaw(msgType string, raw []byte) []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:  msgType,
		Bytes: raw,
	})
}
