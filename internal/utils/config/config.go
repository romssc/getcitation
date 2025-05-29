// Пакет config предоставляет функциональность для загрузки конфигурации приложения из переменных окружения, включая параметры сервера и БД.
package config

import (
	"fmt"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

// Config содержит параметры конфигурации приложения, загружаемые из env-переменных.
type Config struct {
	AppLogMode string `env:"APP_LOG_MODE" env-required:"true" env-description:"Режим логгирования (local, dev, prod)"`

	MigrationsPath      string `env:"MIGRATIONS_PATH" env-required:"true" env-description:"Путь до миграций"`
	MigrationsDirection string `env:"MIGRATIONS_DIRECTION" env-required:"true" env-description:"Направление миграций"`
	MigrationsTable     string `env:"MIGRATIONS_TABLE" env-required:"true" env-description:"Таблица миграций"`

	ServerHost         string        `env:"SERVER_HOST" env-required:"true" env-description:"Имя хоста"`
	ServerPort         string        `env:"SERVER_PORT" env-required:"true" env-description:"Порт сервера"`
	ServerReadTimeout  time.Duration `env:"SERVER_READTIMEOUT" env-required:"true" env-description:"Таймаут сервера на Read"`
	ServerWriteTimeout time.Duration `env:"SERVER_WRITETIMEOUT" env-required:"true" env-description:"Таймаут сервера на Write"`
	ServerIdleTimeout  time.Duration `env:"SERVER_IDLETIMEOUT" env-required:"true" env-description:"Таймаут сервера на Idle"`

	PostgreSQLUsername string `env:"POSTGRESQL_USERNAME" env-required:"true" env-description:"Имя пользователя PostgreSQL"`
	PostgreSQLPassword string `env:"POSTGRESQL_PASSWORD" env-description:"Пароль PostgreSQL"`
	PostgreSQLHost     string `env:"POSTGRESQL_HOST" env-required:"true" env-description:"Имя хоста PostgreSQL"`
	PostgreSQLPort     string `env:"POSTGRESQL_PORT" env-required:"true" env-description:"Порт PostgreSQL"`
	PostgreSQLDatabase string `env:"POSTGRESQL_DBNAME" env-required:"true" env-description:"БД PostgreSQL"`
	PostgreSQLTable    string `env:"POSTGRESQL_TABLE" env-required:"true" env-description:"Таблица PostgreSQL"`
	PostgreSQLSSL      string `env:"POSTGRESQL_SSLMODE" env-required:"true" env-description:"Режим SSL PostgreSQL"`
	PostgreSQLExtra    string `env:"POSTGRESQL_EXTRA" env-description:"Дополнительные опции PostgreSQL"`
}

// New загружает конфигурацию из переменных окружения, используя .env файл и cleanenv.
func New() (Config, error) {
	const op = "config.New()"

	var config Config

	err := godotenv.Load()
	if err != nil {
		return Config{}, fmt.Errorf("%s: %w", op, err)
	}

	err = cleanenv.ReadEnv(&config)
	if err != nil {
		fmt.Println("")

		header := "Ожидаемые переменные окружения:"
		cleanenv.FUsage(os.Stdout, &config, &header)()

		fmt.Println("")

		return Config{}, fmt.Errorf("%s: %w", op, err)
	}

	return config, nil
}
