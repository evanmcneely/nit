package config

import (
	"strings"

	"github.com/spf13/viper"
)

type (
	// Config stores complete configuration
	Config struct {
		App    AppConfig
		Review ReviewConfig
	}

	// AppConfig stores application configuration
	AppConfig struct {
		Port          string
		Hostname      string
		WebhookSecret string
		OpenaiKey     string
		AnthropicKey  string
		GithubToken   string
	}

	// Stores review specific data
	ReviewConfig struct {
		OptIn bool
		Name  string
	}
)

// GetConfig loads and returns configuration
func GetConfig() (Config, error) {
	var c Config

	// Load the config file
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("internal/config")

	// Load env variables
	viper.SetEnvPrefix("nit")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := viper.ReadInConfig(); err != nil {
		return c, err
	}

	if err := viper.Unmarshal(&c); err != nil {
		return c, err
	}

	return c, nil
}
