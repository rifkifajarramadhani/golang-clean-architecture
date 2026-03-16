package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	App struct {
		Port string `yaml:"port"`
	}

	Database struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
		Name     string `yaml:"name"`
		DSN      string `yaml:"-"`
	}

	Auth struct {
		JWTAccessSecret  string `yaml:"jwt_access_secret"`
		JWTRefreshSecret string `yaml:"jwt_refresh_secret"`
		AccessTTLMinutes int    `yaml:"access_ttl_minutes"`
		RefreshTTLHours  int    `yaml:"refresh_ttl_hours"`
	}
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
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
