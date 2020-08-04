package config

type Config struct {
	StaticDir string
	SslMode   bool
	Hostname  string
	Port      string

	TurnServerAddr string
}
