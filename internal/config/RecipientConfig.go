package config

type RecipientConfig struct {
	Enabled bool
	Address string

	TimeLayouts map[string]string
}
