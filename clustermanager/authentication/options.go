package authentication

type CertInfo struct {
	// client verify Certificate
	CABundle []byte

	// server load
	TLSKey  []byte
	TLSCert []byte
}

type SignedWay string

var (
	SelfSigned SignedWay = "SelfSigned"
	CSRSigned  SignedWay = "CSRSigned"
)
