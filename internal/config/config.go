package config

import "github.com/caarlos0/env/v11"

type Config struct {
	TelegramBotToken string `env:"TELEGRAM_BOT_TOKEN,required,notEmpty"`
	DBDSN            string `env:"DB_DSN,required,notEmpty"`
}

func Load() (Config, error) {
	return env.ParseAs[Config]()
}
