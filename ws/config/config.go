package config

import "os"

type Config struct {
	Port        string
	CodeDir     string
	SessionName string
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

	sessionName := "callcode-session-" + os.Getenv("CALLCODE_SESSION_NAME")

	return &Config{
		Port:        port,
		CodeDir:     codeDir,
		SessionName: sessionName,
	}
}
