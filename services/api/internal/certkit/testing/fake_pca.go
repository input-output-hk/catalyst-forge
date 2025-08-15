package testing

import (
	"context"
	"errors"
	"sync"
	"time"
)

// FakePCA is a simple in-memory PCA client for tests.
type FakePCA struct {
	mu        sync.Mutex
	certs     map[string]struct{ cert, chain string }
	issued    []IssueRecord
	delay     time.Duration
	nextArnID int
}

type IssueRecord struct {
	In any
}

func NewFakePCA() *FakePCA { return &FakePCA{certs: map[string]struct{ cert, chain string }{}} }

func (f *FakePCA) WithDelay(d time.Duration) *FakePCA { f.delay = d; return f }

func (f *FakePCA) Issue(ctx context.Context, in any) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nextArnID++
	arn := "arn:fake:cert/" + time.Now().Format("150405") + "/" + itoa(f.nextArnID)
	f.issued = append(f.issued, IssueRecord{In: in})
	// Stub certificate content; in real tests set with SetCert
	if _, ok := f.certs[arn]; !ok {
		f.certs[arn] = struct{ cert, chain string }{cert: "-----BEGIN CERTIFICATE-----\nFAKE\n-----END CERTIFICATE-----\n", chain: ""}
	}
	return arn, nil
}

func (f *FakePCA) GetCertificate(ctx context.Context, caArn, certArn string) (string, string, error) {
	// simulate delay
	if f.delay > 0 {
		time.Sleep(f.delay)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	cc, ok := f.certs[certArn]
	if !ok {
		return "", "", errors.New("not found")
	}
	return cc.cert, cc.chain, nil
}

func (f *FakePCA) GetCACertificate(ctx context.Context, caArn string) (string, string, error) {
	return "-----BEGIN CERTIFICATE-----\nFAKE-CA\n-----END CERTIFICATE-----\n", "", nil
}

func (f *FakePCA) SetCert(arn, cert, chain string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.certs[arn] = struct{ cert, chain string }{cert: cert, chain: chain}
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
