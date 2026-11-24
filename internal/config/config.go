package config

import (
	"log/slog"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	TelegramBotToken string     `env:"TELEGRAM_BOT_TOKEN,required,notEmpty"`
	DBDSN            string     `env:"DB_DSN,required,notEmpty"`
	AdminTelegramID  int64      `env:"ADMIN_TELEGRAM_ID,required,notEmpty"`
	GroupChatID      int64      `env:"GROUP_CHAT_ID,required,notEmpty"`
	LogLevel         slog.Level `env:"LOG_LEVEL" envDefault:"info"`
	LogFormat        string     `env:"LOG_FORMAT" envDefault:"text"`
}

func Load() (Config, error) {
	return env.ParseAs[Config]()
}
