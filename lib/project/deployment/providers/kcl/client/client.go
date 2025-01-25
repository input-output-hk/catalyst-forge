package client

//go:generate go run github.com/matryer/moq@latest -skip-ensure -pkg mocks -out mocks/kcl.go . KCLClient

// KCLClient is the interface for a KCL client.
type KCLClient interface {
	Run(string, KCLModuleConfig) (string, error)
	Log() string
}
