package main

import (
	"github.com/caarlos0/env/v6"
)

type Config struct {
	AwsRegion          string `env:"AWS_REGION,required"`
	AwsStaticKeyID     string `env:"AWS_STATIC_KEY_ID,required"`
	AwsStaticKeySecret string `env:"AWS_STATIC_KEY_SECRET,required"`
	AwsApiKey          string `env:"AWS_API_KEY,required"`
	AwsBucket          string `env:"AWS_BUCKET,required"`

	BotToken   string `env:"BOT_TOKEN,required"`
	BotOwnerID int64  `env:"BOT_OWNER_ID,required"`

	TrustedIDs []int64 `env:"TRUSTED_IDS"`
}

func readConfig() (Config, error) {
	config := Config{}
	err := env.Parse(&config)
	if err != nil {
		return config, err
	}

	return config, err
}
