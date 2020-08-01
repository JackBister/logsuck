package config

type ForwarderConfig struct {
	Enabled           bool
	MaxBufferedEvents int
	RecipientAddress  string
}
