package certificates

type CertificatesCmd struct {
	Sign SignCmd    `cmd:"" help:"Sign a Certificate Signing Request (CSR)."`
	Root GetRootCmd `cmd:"" help:"Get the CA root certificate."`
}
