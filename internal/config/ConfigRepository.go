package config

type ConfigRepository interface {
	SetAll(m map[string]string) error
}
