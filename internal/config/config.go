package config

import (
	"fmt"
	stdmail "net/mail"
	"strings"

	"github.com/spf13/viper"
)

const (
	QueueDriverDatabase = "database"
	QueueDriverRedis    = "redis"

	MailEncryptionNone     = "none"
	MailEncryptionStartTLS = "starttls"
	MailEncryptionTLS      = "tls"
)

type AppConfig struct {
	Port string `mapstructure:"port"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Name     string `mapstructure:"name"`
	DSN      string `mapstructure:"-"`
}

type AuthConfig struct {
	JWTAccessSecret  string `mapstructure:"jwt_access_secret"`
	JWTRefreshSecret string `mapstructure:"jwt_refresh_secret"`
	AccessTTLMinutes int    `mapstructure:"access_ttl_minutes"`
	RefreshTTLHours  int    `mapstructure:"refresh_ttl_hours"`
}

type RedisConfig struct {
	Address  string `mapstructure:"address"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type QueueConfig struct {
	Driver          string              `mapstructure:"driver"`
	Concurrency     int                 `mapstructure:"concurrency"`
	ShutdownSeconds int                 `mapstructure:"shutdown_seconds"`
	Queues          map[string]int      `mapstructure:"queues"`
	Database        DatabaseQueueConfig `mapstructure:"database"`
}

type DatabaseQueueConfig struct {
	PollIntervalMilliseconds int `mapstructure:"poll_interval_milliseconds"`
	ReservationSeconds       int `mapstructure:"reservation_seconds"`
}

type SchedulerConfig struct {
	Timezone string `mapstructure:"timezone"`
}

type MailConfig struct {
	Host        string `mapstructure:"host"`
	Port        int    `mapstructure:"port"`
	Username    string `mapstructure:"username"`
	Password    string `mapstructure:"password"`
	Encryption  string `mapstructure:"encryption"`
	FromAddress string `mapstructure:"from_address"`
	FromName    string `mapstructure:"from_name"`
}

type Config struct {
	App       AppConfig       `mapstructure:"app"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Auth      AuthConfig      `mapstructure:"auth"`
	Redis     RedisConfig     `mapstructure:"redis"`
	Queue     QueueConfig     `mapstructure:"queue"`
	Scheduler SchedulerConfig `mapstructure:"scheduler"`
	Mail      MailConfig      `mapstructure:"mail"`
}

func Load() (*Config, error) {
	instance := viper.New()
	instance.SetConfigName("config")
	instance.SetConfigType("yaml")
	instance.AddConfigPath("./configs")
	instance.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	instance.AutomaticEnv()

	if err := instance.ReadInConfig(); err != nil {
		return nil, err
	}

	var config Config
	if err := instance.Unmarshal(&config); err != nil {
		return nil, err
	}
	if err := normalizeQueueConfig(&config.Queue); err != nil {
		return nil, err
	}
	if err := normalizeMailConfig(&config.Mail); err != nil {
		return nil, err
	}

	config.Database.DSN = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		config.Database.User,
		config.Database.Password,
		config.Database.Host,
		config.Database.Port,
		config.Database.Name,
	)

	return &config, nil
}

func normalizeMailConfig(mail *MailConfig) error {
	mail.Host = strings.TrimSpace(mail.Host)
	if mail.Host == "" {
		mail.Host = "localhost"
	}
	if mail.Port <= 0 {
		mail.Port = 1025
	}
	mail.Encryption = strings.ToLower(strings.TrimSpace(mail.Encryption))
	if mail.Encryption == "" {
		mail.Encryption = MailEncryptionNone
	}
	switch mail.Encryption {
	case MailEncryptionNone, MailEncryptionStartTLS, MailEncryptionTLS:
	default:
		return fmt.Errorf("unsupported mail encryption %q", mail.Encryption)
	}
	mail.FromAddress = strings.TrimSpace(mail.FromAddress)
	if mail.FromAddress == "" {
		mail.FromAddress = "hello@example.com"
	}
	if _, err := stdmail.ParseAddress(mail.FromAddress); err != nil {
		return fmt.Errorf("invalid mail from address: %w", err)
	}
	if strings.TrimSpace(mail.FromName) == "" {
		mail.FromName = "Golang Clean Architecture"
	}
	return nil
}

func normalizeQueueConfig(queue *QueueConfig) error {
	queue.Driver = strings.ToLower(strings.TrimSpace(queue.Driver))
	if queue.Driver == "" {
		queue.Driver = QueueDriverRedis
	}
	if queue.Driver != QueueDriverDatabase && queue.Driver != QueueDriverRedis {
		return fmt.Errorf("unsupported queue driver %q", queue.Driver)
	}
	if queue.Concurrency <= 0 {
		queue.Concurrency = 1
	}
	if queue.ShutdownSeconds <= 0 {
		queue.ShutdownSeconds = 30
	}
	if queue.Database.PollIntervalMilliseconds <= 0 {
		queue.Database.PollIntervalMilliseconds = 500
	}
	if queue.Database.ReservationSeconds <= 0 {
		queue.Database.ReservationSeconds = 60
	}
	return nil
}
