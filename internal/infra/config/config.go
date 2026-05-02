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
	Enabled           bool    `yaml:"enabled"`
	HalfOpenRequests  uint32  `yaml:"half-open-requests"` // кол-во запросов в half-open состоянии
	Interval          string  `yaml:"interval"`           // окно сброса счётчиков в closed состоянии
	Timeout           string  `yaml:"timeout"`            // время в open состоянии перед переходом в half-open
	Threshold         float64 `yaml:"threshold"`          // доля ошибок для открытия (0.0–1.0)
	MinRequestsToTrip uint32  `yaml:"min-requests-to-trip"` // минимум запросов в окне перед проверкой threshold
}

type RetryConfig struct {
	Enabled        bool    `yaml:"enabled"`
	MaxAttempts    int     `yaml:"max-attempts"`    // общее кол-во попыток включая первую (1 = без ретраев)
	InitialBackoff string  `yaml:"initial-backoff"` // пауза перед первым ретраем
	MaxBackoff     string  `yaml:"max-backoff"`     // максимальная пауза между ретраями
	Multiplier     float64 `yaml:"multiplier"`      // множитель для exponential backoff (2.0 = каждый раз вдвое дольше)
	JitterFactor   float64 `yaml:"jitter-factor"`   // случайное отклонение паузы (0.2 = ±20%, 0.0 = без jitter)
}

type ReservationConfirmationOutboxConfig struct {
	Enabled        bool   `yaml:"enabled"`
	IdleInterval   string `yaml:"idle-interval"`   // пауза когда очередь пуста
	ActiveInterval string `yaml:"active-interval"` // пауза когда в прошлом тике были записи (0 = сразу)
	BatchSize      int    `yaml:"batch-size"`
	MaxRetries     int    `yaml:"max-retries"`
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
