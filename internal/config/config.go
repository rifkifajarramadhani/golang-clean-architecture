package config

import (
	"errors"
	"fmt"
	"net"
	stdmail "net/mail"
	"os"
	"strconv"
	"strings"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
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
	Port        string `mapstructure:"port"`
	Environment string `mapstructure:"environment"`
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
	JWTAccessSecret      string `mapstructure:"jwt_access_secret"`
	JWTRefreshSecret     string `mapstructure:"jwt_refresh_secret"`
	AccessTTLMinutes     int    `mapstructure:"access_ttl_minutes"`
	RefreshTTLHours      int    `mapstructure:"refresh_ttl_hours"`
	Issuer               string `mapstructure:"issuer"`
	Audience             string `mapstructure:"audience"`
	VerificationTTLHours int    `mapstructure:"verification_ttl_hours"`
	BootstrapAdminEmail  string `mapstructure:"bootstrap_admin_email"`
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

type LoggingConfig struct {
	Level string `mapstructure:"level"`
	File  string `mapstructure:"file"`
}

type Config struct {
	App       AppConfig       `mapstructure:"app"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Auth      AuthConfig      `mapstructure:"auth"`
	Redis     RedisConfig     `mapstructure:"redis"`
	Queue     QueueConfig     `mapstructure:"queue"`
	Scheduler SchedulerConfig `mapstructure:"scheduler"`
	Mail      MailConfig      `mapstructure:"mail"`
	Logging   LoggingConfig   `mapstructure:"logging"`
}

func Load() (*Config, error) {
	if err := loadDotEnv(".env"); err != nil {
		return nil, err
	}

	instance := viper.New()
	instance.SetConfigName("config")
	instance.SetConfigType("yaml")
	instance.AddConfigPath("./configs")
	instance.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	instance.AutomaticEnv()
	for _, key := range configKeys {
		if err := instance.BindEnv(key); err != nil {
			return nil, fmt.Errorf("bind environment key %s: %w", key, err)
		}
	}

	if err := instance.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return nil, err
		}
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
	if err := normalizeAuthConfig(&config.App, &config.Auth); err != nil {
		return nil, err
	}
	normalizeLoggingConfig(&config.Logging)
	if err := validateConfig(&config); err != nil {
		return nil, err
	}

	dsn := mysqldriver.NewConfig()
	dsn.User = config.Database.User
	dsn.Passwd = config.Database.Password
	dsn.Net = "tcp"
	dsn.Addr = config.Database.Host + ":" + strconv.Itoa(config.Database.Port)
	dsn.DBName = config.Database.Name
	dsn.ParseTime = true
	dsn.Collation = "utf8mb4_unicode_ci"
	dsn.Loc = time.UTC
	config.Database.DSN = dsn.FormatDSN()

	return &config, nil
}

func loadDotEnv(filename string) error {
	if err := godotenv.Load(filename); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("load dotenv file: %w", err)
	}
	return nil
}

var configKeys = []string{
	"app.port", "app.environment",
	"database.host", "database.port", "database.user", "database.password", "database.name",
	"auth.jwt_access_secret", "auth.jwt_refresh_secret", "auth.access_ttl_minutes", "auth.refresh_ttl_hours",
	"auth.issuer", "auth.audience", "auth.verification_ttl_hours", "auth.bootstrap_admin_email",
	"redis.address", "redis.password", "redis.db",
	"queue.driver", "queue.concurrency", "queue.shutdown_seconds", "queue.queues",
	"queue.database.poll_interval_milliseconds", "queue.database.reservation_seconds",
	"scheduler.timezone", "mail.host", "mail.port", "mail.username", "mail.password", "mail.encryption",
	"mail.from_address", "mail.from_name", "logging.level", "logging.file",
}

func validateConfig(config *Config) error {
	port, err := strconv.Atoi(config.App.Port)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("app port must be between 1 and 65535")
	}
	if strings.TrimSpace(config.Database.Host) == "" || config.Database.Port < 1 || config.Database.Port > 65535 ||
		strings.TrimSpace(config.Database.User) == "" || strings.TrimSpace(config.Database.Name) == "" {
		return fmt.Errorf("database host, port, user, and name must be valid")
	}
	if isProduction(config.App.Environment) && strings.TrimSpace(config.Database.Password) == "" {
		return fmt.Errorf("DATABASE_PASSWORD is required in production")
	}
	if _, _, err := net.SplitHostPort(config.Redis.Address); err != nil {
		return fmt.Errorf("redis address must be host:port")
	}
	if len(config.Queue.Queues) == 0 {
		return fmt.Errorf("at least one queue must be configured")
	}
	for name, weight := range config.Queue.Queues {
		if strings.TrimSpace(name) == "" || weight <= 0 {
			return fmt.Errorf("queue names must be non-empty and weights must be positive")
		}
	}
	return nil
}

func normalizeAuthConfig(app *AppConfig, auth *AuthConfig) error {
	app.Environment = strings.ToLower(strings.TrimSpace(app.Environment))
	if app.Environment == "" {
		app.Environment = "development"
	}
	auth.Issuer = strings.TrimSpace(auth.Issuer)
	if auth.Issuer == "" {
		auth.Issuer = "golang-clean-architecture"
	}
	auth.Audience = strings.TrimSpace(auth.Audience)
	if auth.Audience == "" {
		auth.Audience = "golang-clean-architecture-api"
	}
	if auth.VerificationTTLHours <= 0 {
		auth.VerificationTTLHours = 24
	}
	if auth.AccessTTLMinutes <= 0 {
		auth.AccessTTLMinutes = 15
	}
	if auth.RefreshTTLHours <= 0 {
		auth.RefreshTTLHours = 168
	}
	auth.BootstrapAdminEmail = strings.ToLower(strings.TrimSpace(auth.BootstrapAdminEmail))
	if auth.BootstrapAdminEmail != "" {
		address, err := stdmail.ParseAddress(auth.BootstrapAdminEmail)
		if err != nil {
			return fmt.Errorf("invalid bootstrap admin email: %w", err)
		}
		if address.Address != auth.BootstrapAdminEmail {
			return fmt.Errorf("invalid bootstrap admin email %q", auth.BootstrapAdminEmail)
		}
	}
	if isProduction(app.Environment) {
		if strings.TrimSpace(auth.JWTAccessSecret) == "" {
			return fmt.Errorf("AUTH_JWT_ACCESS_SECRET is required in production")
		}
		if strings.TrimSpace(auth.JWTRefreshSecret) == "" {
			return fmt.Errorf("AUTH_JWT_REFRESH_SECRET is required in production")
		}
		if len(auth.JWTAccessSecret) < 32 || len(auth.JWTRefreshSecret) < 32 ||
			auth.JWTAccessSecret == auth.JWTRefreshSecret ||
			isPlaceholderSecret(auth.JWTAccessSecret) || isPlaceholderSecret(auth.JWTRefreshSecret) {
			return fmt.Errorf("production JWT secrets must be distinct, non-placeholder values of at least 32 bytes")
		}
	}
	return nil
}

func isProduction(environment string) bool {
	return environment != "development" && environment != "test"
}

func isPlaceholderSecret(secret string) bool {
	normalized := strings.ToLower(secret)
	return strings.Contains(normalized, "change") || strings.Contains(normalized, "placeholder")
}

func normalizeLoggingConfig(logging *LoggingConfig) {
	logging.Level = strings.ToLower(strings.TrimSpace(logging.Level))
	if logging.Level == "" {
		logging.Level = "info"
	}
	if strings.TrimSpace(logging.File) == "" {
		logging.File = "logs/app.log"
	}
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
