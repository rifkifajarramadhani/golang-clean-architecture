package config

import (
	"errors"
	"fmt"
	stdmail "net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
)

const (
	EnvironmentDevelopment = "development"
	EnvironmentTest        = "test"
	EnvironmentProduction  = "production"

	MailEncryptionNone     = "none"
	MailEncryptionStartTLS = "starttls"
	MailEncryptionTLS      = "tls"
)

type AppConfig struct {
	Name        string `mapstructure:"name"`
	Environment string `mapstructure:"environment"`
	Port        string `mapstructure:"port"`
}

type HTTPConfig struct {
	BodyLimitBytes        int      `mapstructure:"body_limit_bytes"`
	RequestTimeoutSeconds int      `mapstructure:"request_timeout_seconds"`
	CORSAllowedOrigins    []string `mapstructure:"cors_allowed_origins"`
	AuthRateLimit         int      `mapstructure:"auth_rate_limit"`
}

type DatabaseConfig struct {
	Host                 string `mapstructure:"host"`
	Port                 int    `mapstructure:"port"`
	User                 string `mapstructure:"user"`
	Password             string `mapstructure:"password"`
	Name                 string `mapstructure:"name"`
	MaxOpenConnections   int    `mapstructure:"max_open_connections"`
	MaxIdleConnections   int    `mapstructure:"max_idle_connections"`
	ConnectionMaxMinutes int    `mapstructure:"connection_max_minutes"`
	DSN                  string `mapstructure:"-"`
}

type AuthConfig struct {
	JWTAccessSecret  string `mapstructure:"jwt_access_secret"`
	JWTRefreshSecret string `mapstructure:"jwt_refresh_secret"`
	AccessTTLMinutes int    `mapstructure:"access_ttl_minutes"`
	RefreshTTLHours  int    `mapstructure:"refresh_ttl_hours"`
	Issuer           string `mapstructure:"issuer"`
	Audience         string `mapstructure:"audience"`
}

type RedisConfig struct {
	Address  string `mapstructure:"address"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type QueueConfig struct {
	Concurrency     int            `mapstructure:"concurrency"`
	ShutdownSeconds int            `mapstructure:"shutdown_seconds"`
	Queues          map[string]int `mapstructure:"queues"`
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
	HTTP      HTTPConfig      `mapstructure:"http"`
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
	setDefaults(instance)
	if err := instance.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return nil, err
		}
	}
	var cfg Config
	if err := instance.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	cfg.Database.DSN = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=UTC",
		cfg.Database.User, cfg.Database.Password, cfg.Database.Host, cfg.Database.Port, cfg.Database.Name)
	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("app.name", "go-service")
	v.SetDefault("app.environment", EnvironmentDevelopment)
	v.SetDefault("app.port", "8080")
	v.SetDefault("http.body_limit_bytes", 1<<20)
	v.SetDefault("http.request_timeout_seconds", 15)
	v.SetDefault("http.cors_allowed_origins", []string{"http://localhost:3000"})
	v.SetDefault("http.auth_rate_limit", 20)
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 3306)
	v.SetDefault("database.max_open_connections", 25)
	v.SetDefault("database.max_idle_connections", 5)
	v.SetDefault("database.connection_max_minutes", 30)
	v.SetDefault("auth.access_ttl_minutes", 15)
	v.SetDefault("auth.refresh_ttl_hours", 168)
	v.SetDefault("auth.issuer", "go-service")
	v.SetDefault("auth.audience", "go-service")
	v.SetDefault("redis.address", "localhost:6379")
	v.SetDefault("queue.concurrency", 10)
	v.SetDefault("queue.shutdown_seconds", 30)
	v.SetDefault("queue.queues", map[string]int{"default": 3, "mail": 2, "maintenance": 1})
	v.SetDefault("scheduler.timezone", "UTC")
	v.SetDefault("mail.host", "localhost")
	v.SetDefault("mail.port", 1025)
	v.SetDefault("mail.encryption", MailEncryptionNone)
	v.SetDefault("mail.from_address", "hello@example.com")
	v.SetDefault("mail.from_name", "Go Service")
}

func (c *Config) validate() error {
	c.App.Environment = strings.ToLower(strings.TrimSpace(c.App.Environment))
	if c.App.Environment != EnvironmentDevelopment && c.App.Environment != EnvironmentTest && c.App.Environment != EnvironmentProduction {
		return fmt.Errorf("unsupported environment %q", c.App.Environment)
	}
	port, err := strconv.Atoi(c.App.Port)
	if err != nil || port < 1 || port > 65535 {
		return errors.New("app port must be between 1 and 65535")
	}
	if c.Database.Host == "" || c.Database.User == "" || c.Database.Name == "" || c.Database.Port <= 0 {
		return errors.New("database host, port, user, and name are required")
	}
	if c.App.Environment == EnvironmentProduction && c.Database.Password == "" {
		return errors.New("database password is required in production")
	}
	if len(c.Auth.JWTAccessSecret) < 32 || len(c.Auth.JWTRefreshSecret) < 32 {
		return errors.New("JWT access and refresh secrets must each be at least 32 characters")
	}
	if c.Auth.JWTAccessSecret == c.Auth.JWTRefreshSecret {
		return errors.New("JWT access and refresh secrets must differ")
	}
	if c.Auth.AccessTTLMinutes <= 0 || c.Auth.RefreshTTLHours <= 0 || c.Auth.Issuer == "" || c.Auth.Audience == "" {
		return errors.New("auth TTL, issuer, and audience values must be valid")
	}
	if c.Redis.Address == "" {
		return errors.New("redis address is required")
	}
	if c.Queue.Concurrency <= 0 || c.Queue.ShutdownSeconds <= 0 || len(c.Queue.Queues) == 0 {
		return errors.New("queue concurrency, shutdown timeout, and queues must be configured")
	}
	for name, weight := range c.Queue.Queues {
		if strings.TrimSpace(name) == "" || weight <= 0 {
			return errors.New("queue names and weights must be valid")
		}
	}
	if c.HTTP.BodyLimitBytes <= 0 || c.HTTP.RequestTimeoutSeconds <= 0 || c.HTTP.AuthRateLimit <= 0 {
		return errors.New("HTTP limits and timeouts must be positive")
	}
	c.Mail.Encryption = strings.ToLower(strings.TrimSpace(c.Mail.Encryption))
	if c.Mail.Encryption != MailEncryptionNone && c.Mail.Encryption != MailEncryptionStartTLS && c.Mail.Encryption != MailEncryptionTLS {
		return fmt.Errorf("unsupported mail encryption %q", c.Mail.Encryption)
	}
	if _, err := stdmail.ParseAddress(c.Mail.FromAddress); err != nil {
		return fmt.Errorf("invalid mail from address: %w", err)
	}
	return nil
}

func (c AuthConfig) AccessTTL() time.Duration  { return time.Duration(c.AccessTTLMinutes) * time.Minute }
func (c AuthConfig) RefreshTTL() time.Duration { return time.Duration(c.RefreshTTLHours) * time.Hour }
