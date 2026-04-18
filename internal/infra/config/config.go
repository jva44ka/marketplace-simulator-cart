package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Host string `yaml:"host"`
		Port string `yaml:"port"`
	} `yaml:"server"`

	Products struct {
		Host      string `yaml:"host"`
		Port      string `yaml:"port"`
		AuthToken string `yaml:"auth-token"`
		Schema    string `yaml:"schema"`
		Timeout   string `yaml:"timeout"`
	} `yaml:"products"`

	Database struct {
		User     string `yaml:"user"`
		Password string `yaml:"password"`
		Host     string `yaml:"host"`
		Port     string `yaml:"port"`
		Name     string `yaml:"name"`
	} `yaml:"database"`

	Jobs struct {
		ReservationConfirmationOutbox        ReservationConfirmationOutboxConfig        `yaml:"reservation-confirmation-outbox"`
		ReservationConfirmationOutboxMonitor ReservationConfirmationOutboxMonitorConfig `yaml:"reservation-confirmation-outbox-monitor"`
	} `yaml:"jobs"`

	Tracing struct {
		Enabled      bool   `yaml:"enabled"`
		OtlpEndpoint string `yaml:"otlp-endpoint"`
	} `yaml:"tracing"`
}

type ReservationConfirmationOutboxConfig struct {
	Enabled     bool   `yaml:"enabled"`
	JobInterval string `yaml:"job-interval"`
	BatchSize   int    `yaml:"batch-size"`
	MaxRetries  int    `yaml:"max-retries"`
}

type ReservationConfirmationOutboxMonitorConfig struct {
	Enabled     bool   `yaml:"enabled"`
	JobInterval string `yaml:"job-interval"`
}

func LoadConfig(filename string) (*Config, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	config := &Config{}
	if err := yaml.NewDecoder(f).Decode(config); err != nil {
		return nil, err
	}

	return config, nil
}
