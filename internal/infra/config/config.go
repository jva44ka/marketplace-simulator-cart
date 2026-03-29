package config

import (
	"errors"
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
		ReservationExpiredConsumer struct {
			Enabled bool `yaml:"enabled"`
		} `yaml:"reservation-expired-consumer"`
	} `yaml:"jobs"`

	Kafka KafkaConfig `yaml:"kafka"`
}

type TopicConfig struct {
	Name          string `yaml:"name"`
	ConsumerGroup string `yaml:"consumer-group"`
}

type KafkaConfig struct {
	Brokers []string      `yaml:"brokers"`
	Topics  []TopicConfig `yaml:"topics"`
}

func (cfg KafkaConfig) GetReservationExpiredTopicConfig() (TopicConfig, error) {
	for _, topicConfig := range cfg.Topics {
		if topicConfig.Name == "reservation-expired-topic" {
			return topicConfig, nil
		}
	}

	return TopicConfig{}, errors.New("reservation-expired-topic not found")
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
