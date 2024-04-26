package config

import (
	"strings"

	"github.com/spf13/viper"
)

type (
	// Config stores complete configuration
	Config struct {
		App    AppConfig
		Github GithubConfig
		AI     AIConfig
	}

	// AppConfig stores application configuration
	AppConfig struct {
		Name     string
		Port     string
		Hostname string
	}

	// GithubConfig stores the Github app config
	GithubConfig struct {
		WebhookSecret string
	}

	AIConfig struct {
		OpenaiKey    string
		AnthropicKey string
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
