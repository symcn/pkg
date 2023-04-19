package authentication

import (
	"fmt"
	"testing"

	"github.com/symcn/pkg/clustermanager/client"
	"github.com/symcn/pkg/selfsigned"
)

func TestNewBuildWebhookCertInfoWithCSR(t *testing.T) {
	cfg := client.DefaultClusterCfgInfo("test")
	opt := client.DefaultOptions()
	cli, err := client.NewMingleClient(cfg, opt)
	if err != nil {
		t.Error(err)
		return
	}

	certInfo, err := BuildWebhookCertInfoWithCSR(cli, &selfsigned.CertOptions{
		// subject organization is not system:nodes
		Organization: []string{"system:nodes"},
		// subject common name does not begin with system:node:
		CommonName: "system:node:symcn-webhook",
		DNSNames:   []string{"sym-control-webhook.sym-admin.svc"},
	})
	if err != nil {
		t.Error(err)
		return
	}

	fmt.Println(string(certInfo.CABundle))
	fmt.Println(string(certInfo.TLSCert))
	fmt.Println(string(certInfo.TLSKey))
}

func TestReadCABundle(t *testing.T) {

}
