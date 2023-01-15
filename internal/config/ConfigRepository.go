package config

type ConfigRepository interface {
	Upsert(c *Config) error
	Get() (*ConfigResponse, error)
	Changes() <-chan struct{}
}
