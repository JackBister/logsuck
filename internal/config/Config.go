package config

type Config struct {
	IndexedFiles []IndexedFileConfig

	EnableWeb bool
	HttpAddr  string
}
