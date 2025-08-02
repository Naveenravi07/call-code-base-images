package config

import "os"

type Config struct {
	Port    string
	CodeDir string
}

func LoadConfig() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	codeDir := os.Getenv("CODE_DIR")
	if codeDir == "" {
		codeDir = "/code"
	}

	return &Config{
		Port:    port,
		CodeDir: codeDir,
	}
}
