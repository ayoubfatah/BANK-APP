package utils

import "github.com/spf13/viper"

type Config struct {
	DBURL         string `mapstructure:"DB_URL"`
	ServerAddress string `mapstructure:"SERVER_ADDRESS"`
}

func LoadConfig(path string) (Config, error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("app")
	viper.SetConfigType("env")

	viper.AutomaticEnv() // env vars override the file

	if err := viper.ReadInConfig(); err != nil {
		return Config{}, err
	}

	var config Config
	err := viper.Unmarshal(&config)
	return config, err
}