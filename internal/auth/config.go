package auth

import (
	"os"
	"path/filepath"
)

const (
	defaultBotPublicBaseURL = "https://request.bangumi.ca"
	defaultBotAPIBaseURL    = "https://request.bangumi.ca"
)

type Config struct {
	BotPublicBaseURL string
	BotAPIBaseURL    string
	ConfigDir        string
	ClientID         string
}

func LoadConfig() Config {
	configDir := os.Getenv("NYAUPLOAD_CONFIG_DIR")
	if configDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			configDir = "."
		} else {
			configDir = filepath.Join(homeDir, ".config", "nyaupload")
		}
	}

	publicBaseURL := os.Getenv("NYAUPLOAD_BOT_PUBLIC_BASE_URL")
	if publicBaseURL == "" {
		publicBaseURL = defaultBotPublicBaseURL
	}

	apiBaseURL := os.Getenv("NYAUPLOAD_BOT_API_BASE_URL")
	if apiBaseURL == "" {
		apiBaseURL = defaultBotAPIBaseURL
	}

	clientID := os.Getenv("NYAUPLOAD_CLIENT_ID")
	if clientID == "" {
		clientID = "nyaupload-cli"
	}

	return Config{
		BotPublicBaseURL: publicBaseURL,
		BotAPIBaseURL:    apiBaseURL,
		ConfigDir:        configDir,
		ClientID:         clientID,
	}
}
