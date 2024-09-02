package config

import "github.com/nifetency/nife.io/utils"

type DatabaseConfig struct {
	User string
	Host string
	Port string
	Pass string
	Name string
}

type AppConfig struct {
	Database *DatabaseConfig
}

func LoadDBConfig() *DatabaseConfig {
	dbConfig := &DatabaseConfig{
		User: utils.GetEnv("DB_USER", "nife_user"),
		Host: utils.GetEnv("DB_HOST", "34.93.245.107"),
		Port: utils.GetEnv("DB_PORT", "3306"),
		Pass: utils.GetEnv("DB_PASSWORD", "N!f3@321"),
		Name: utils.GetEnv("DB_NAME", "nife"),
	}
	return dbConfig
}

func LoadEnvironmentVars() *AppConfig {
	appConfig := &AppConfig{
		Database: LoadDBConfig(),
	}
	return appConfig
}
