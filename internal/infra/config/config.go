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

		CircuitBreaker CircuitBreakerConfig `yaml:"circuit-breaker"`
		Retry          RetryConfig          `yaml:"retry"`
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

type CircuitBreakerConfig struct {
	Enabled     bool    `yaml:"enabled"`
	MaxRequests uint32  `yaml:"max-requests"` // кол-во запросов в half-open состоянии
	Interval    string  `yaml:"interval"`     // окно сброса счётчиков в closed состоянии
	Timeout     string  `yaml:"timeout"`      // время в open состоянии перед переходом в half-open
	Threshold   float64 `yaml:"threshold"`    // доля ошибок для открытия (0.0–1.0)
}

type RetryConfig struct {
	Enabled        bool    `yaml:"enabled"`
	MaxAttempts    int     `yaml:"max-attempts"`
	InitialBackoff string  `yaml:"initial-backoff"`
	MaxBackoff     string  `yaml:"max-backoff"`
	Multiplier     float64 `yaml:"multiplier"`
	JitterFactor   float64 `yaml:"jitter-factor"` // доля от backoff для случайного отклонения (0.0–1.0)
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
