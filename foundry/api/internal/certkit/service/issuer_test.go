package service

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"testing"
	"time"

	rbac "github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/rbac"
	"github.com/stretchr/testify/require"
)

type testClock struct{}

func (testClock) Now() time.Time { return time.Unix(1_700_000_000, 0).UTC() }

type testPCA struct {
	certs map[string][2]string
	next  int
}

func newTestPCA() *testPCA { return &testPCA{certs: map[string][2]string{}} }
func (p *testPCA) Issue(ctx context.Context, in IssueInput) (string, error) {
	p.next++
	arn := "arn:test:" + itoa(p.next)
	// populate a fake cert for this arn
	p.certs[arn] = [2]string{"-----BEGIN CERTIFICATE-----\nFAKE\n-----END CERTIFICATE-----\n", ""}
	return arn, nil
}
func (p *testPCA) GetCertificate(ctx context.Context, caArn, certArn string) (string, string, error) {
	c := p.certs[certArn]
	return c[0], c[1], nil
}

type allowRBAC struct{ dec rbac.Decision }

func (a allowRBAC) Check(ctx context.Context, subj rbac.Subject, action rbac.PermissionKey, res rbac.ResourceRef) (rbac.Decision, error) {
	return a.dec, nil
}

func genCSRWithDNS(t *testing.T, names []string) []byte {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	tpl := x509.CertificateRequest{
		Subject:  pkix.Name{CommonName: names[0]},
		DNSNames: names,
	}
	der, err := x509.CreateCertificateRequest(rand.Reader, &tpl, key)
	require.NoError(t, err)
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: der})
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	s := ""
	for i > 0 {
		d := i % 10
		s = string(rune('0'+d)) + s
		i /= 10
	}
	return s
}

func TestIssuer_SignCSR_Allows(t *testing.T) {
	csr := genCSRWithDNS(t, []string{"svc.projectcatalyst.io"})
	iss := &Issuer{
		CAArn:            "arn:ca:test",
		PCA:              newTestPCA(),
		RBAC:             allowRBAC{dec: rbac.DecisionAllow},
		Clock:            testClock{},
		AllowedTemplates: map[string]string{"end-entity": "arn:aws:acm-pca:::template/EndEntityCertificate/V1"},
		MaxTTL:           24 * time.Hour,
	}
	res, err := iss.SignCSR(context.Background(), SignRequest{
		CSRPEM:           csr,
		TemplateKey:      "end-entity",
		SigningAlgorithm: "SHA256WITHRSA",
		TTL:              time.Hour,
		IdempotencyKey:   "abc",
		Requestor:        "user-1",
	})
	require.NoError(t, err)
	require.NotEmpty(t, res.CertificatePEM)
	require.NotEmpty(t, res.CertificateArn)
}

func TestIssuer_SignCSR_DeniedByRBAC(t *testing.T) {
	csr := genCSRWithDNS(t, []string{"bad.example.com"})
	iss := &Issuer{
		CAArn:            "arn:ca:test",
		PCA:              newTestPCA(),
		RBAC:             allowRBAC{dec: rbac.DecisionDeny},
		Clock:            testClock{},
		AllowedTemplates: map[string]string{"end-entity": "arn:aws:acm-pca:::template/EndEntityCertificate/V1"},
		MaxTTL:           24 * time.Hour,
	}
	_, err := iss.SignCSR(context.Background(), SignRequest{
		CSRPEM:         csr,
		TemplateKey:    "end-entity",
		TTL:            time.Hour,
		IdempotencyKey: "abc",
		Requestor:      "user-1",
	})
	require.Error(t, err)
}
