package authentication

import (
	"context"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/symcn/api"
	"github.com/symcn/pkg/selfsigned"
	csrv1 "k8s.io/api/certificates/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/klog/v2"
)

var (
	csrNameFormat = "webhook-csr-%s"
	signerName    = "symcn.io/authentication"
	controllName  = "SymcnAuthentication"
)

// BuildWebhookCertInfoWithCSR
// 1. submitCSR
// 2. approveCSR
// 3. readSignedCertificate
// This way may use those rules:
//   - apiGroups: ["certificates.k8s.io"]
//     resources: ["certificatesigningrequests"]
//     verbs: ["create", "get", "watch"]
//   - apiGroups: ["certificates.k8s.io"]
//     resources: ["certificatesigningrequests/approval"]
//     verbs: ["update"]
//   - apiGroups: ["certificates.k8s.io"]
//     resources: ["signers"]
//     resourceNames: ["kubernetes.io/kubelet-serving"]
//     verbs: ["approve"]

func BuildWebhookCertInfoWithCSR(client api.MingleClient, svcOpts *selfsigned.CertOptions) (*CertInfo, error) {
	caBundle, err := readCABundle(client)
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

	csrName, err := submitCSR(client, svcCSR)
	if err != nil {
		return nil, err
	}

	err = approveCSR(client, csrName)
	if err != nil {
		return nil, err
	}

	signedCert, err := readSignedCertificate(client, csrName)
	if err != nil {
		return nil, err
	}

	return &CertInfo{
		CABundle: caBundle,
		TLSKey:   svcSigner.PrivateKey(),
		TLSCert:  signedCert,
	}, nil
}

func submitCSR(client api.MingleClient, csrRaw []byte) (csrName string, err error) {
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second*5)
	defer cancel()

	csrName = buildCSRName()
	_, err = client.GetKubeInterface().CertificatesV1().CertificateSigningRequests().Create(ctx, &csrv1.CertificateSigningRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name: csrName,
		},
		Spec: csrv1.CertificateSigningRequestSpec{
			SignerName: csrv1.KubeletServingSignerName,
			Request:    csrRaw,
			Usages: []csrv1.KeyUsage{
				// usages did not match [digital signature key encipherment server auth]
				csrv1.UsageDigitalSignature,
				csrv1.UsageKeyEncipherment,
				csrv1.UsageServerAuth,
			},
		},
	}, metav1.CreateOptions{})
	if err != nil {
		klog.ErrorS(err, "create CSR failed", "name", csrName)
		return "", err
	}
	klog.V(4).InfoS("create CSR success", "name", csrName)
	return csrName, nil
}

func approveCSR(client api.MingleClient, csrName string) error {
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second*5)
	defer cancel()

	csr, err := client.GetKubeInterface().CertificatesV1().CertificateSigningRequests().Get(ctx, csrName, metav1.GetOptions{})
	if err != nil {
		klog.ErrorS(err, "get CertificateSigningRequests failed", "name", csrName)
		return err
	}
	if csr.Status.Conditions == nil {
		csr.Status.Conditions = make([]csrv1.CertificateSigningRequestCondition, 0)
	}
	csr.Status.Conditions = append(csr.Status.Conditions, csrv1.CertificateSigningRequestCondition{
		Status:         corev1.ConditionTrue,
		Type:           csrv1.CertificateApproved,
		Reason:         fmt.Sprintf("%sApprove", controllName),
		Message:        fmt.Sprintf("This CSR was approved by %s certificate approve.", controllName),
		LastUpdateTime: metav1.Now(),
	})

	_, err = client.GetKubeInterface().CertificatesV1().CertificateSigningRequests().UpdateApproval(ctx, csrName, csr, metav1.UpdateOptions{})
	if err != nil {
		klog.ErrorS(err, "approve CSR failed", "name", csrName)
		return err
	}
	klog.V(4).InfoS("approve CSR success", "name", csrName)
	return nil
}

func readSignedCertificate(client api.MingleClient, csrName string) (signedCert []byte, err error) {
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second*20)
	defer cancel()

	watcher, err := client.GetKubeInterface().CertificatesV1().CertificateSigningRequests().Watch(ctx, metav1.ListOptions{
		FieldSelector: fields.OneTermEqualSelector("metadata.name", csrName).String(),
	})
	if err != nil {
		klog.ErrorS(err, "watch CSR failed", "name", csrName)
		return nil, err
	}

	for r := range watcher.ResultChan() {
		csr, ok := r.Object.(*csrv1.CertificateSigningRequest)
		if !ok {
			return nil, fmt.Errorf("readSignedCertificate watch failed, please check FieldSelector")
		}
		if csr.Status.Certificate != nil {
			p, _ := pem.Decode(csr.Status.Certificate)
			if p == nil {
				return nil, fmt.Errorf("invalid PEM encoded certificate")
			}
			return selfsigned.EncodePemCertWithRaw(p.Bytes), nil
		}

		for _, condition := range csr.Status.Conditions {
			if condition.Type == csrv1.CertificateFailed {
				klog.Errorf("CSR %s SignerValidationFailure: %s", csrName, condition.Message)
				return nil, errors.New(condition.Message)
			}
		}
	}

	return nil, fmt.Errorf("readSignedCertificate %s timeout", csrName)
}

// readCABundle read from rest.Config or defaultCACertPath
// const defaultCACertPath = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
func readCABundle(client api.MingleClient) (caBundle []byte, err error) {
	if len(client.GetKubeRestConfig().CAData) != 0 {
		return client.GetKubeRestConfig().CAData, nil
	}

	if len(client.GetKubeRestConfig().CAFile) != 0 {
		b, err := os.ReadFile(client.GetKubeRestConfig().CAFile)
		if err != nil {
			return nil, err
		}
		return b, nil
	}

	return nil, errors.New("not found CABundle with rest.Config's CAData and CAFile")
}

func buildCSRName() string {
	return fmt.Sprintf(csrNameFormat, rand.String(8))
}
