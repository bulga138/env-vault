package providers

type SecretProvider interface {
	Get(key string) ([]byte, error)
	Set(key string, value []byte) error
	Name() string
}
