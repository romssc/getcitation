// Пакет utils содержит вспомогательные функции общего назначения.
package utils

import (
	"fmt"
	"os"
	"path/filepath"

	"getcitation/internal/utils/config"
)

// GetLogFile создает все необходимые директории для указанного пути, а затем открывает файл для логирования в режиме добавления (append). Если файл не существует — он будет создан. Возвращает файловый дескриптор *os.File или ошибку.
func GetLogFile(path string) (*os.File, error) {
	const op = "utils.GetLogFile()"

	err := os.MkdirAll(filepath.Dir(path), 0777)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return f, nil
}

// BuildPostgresDSN строит строку подключения к PostgreSQL из конфига.
func BuildPostgreSQLDSN(config config.Config) string {
	var conn string

	if config.PostgreSQLPassword == "" {
		conn = fmt.Sprintf(
			"postgres://%s@%s:%s/%s?sslmode=%s",
			config.PostgreSQLUsername,
			config.PostgreSQLHost,
			config.PostgreSQLPort,
			config.PostgreSQLDatabase,
			config.PostgreSQLSSL,
		)
	} else {
		conn = fmt.Sprintf(
			"postgres://%s:%s@%s:%s/%s?sslmode=%s",
			config.PostgreSQLUsername,
			config.PostgreSQLPassword,
			config.PostgreSQLHost,
			config.PostgreSQLPort,
			config.PostgreSQLDatabase,
			config.PostgreSQLSSL,
		)
	}

	if config.PostgreSQLExtra != "" {
		conn += "&&" + config.PostgreSQLExtra
	}

	return conn
}
