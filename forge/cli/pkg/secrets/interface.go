package secrets

//go:generate go run github.com/matryer/moq@latest -out interface_mock.go . SecretProvider

// SecretProvider is an interface for getting and setting secrets.
type SecretProvider interface {
	Get(key string) (string, error)
	Set(key, value string) (string, error)
}
