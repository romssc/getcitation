// Пакет logger предоставляет обёртку над slog для логирования
// в разных режимах: local, dev и prod.
package logger

import (
	"fmt"
	"getcitation/internal/utils"
	"log/slog"
	"os"
)

var ErrUnknownLogMode = fmt.Errorf("неизвестный режим логирования")

const (
	devLogPath  = "log/dev/dev.log.json"
	prodLogPath = "log/log.json"
)

// Logger инкапсулирует slog.Logger и файл, в который пишутся логи (если используется).
type Logger struct {
	Log  *slog.Logger
	File *os.File
}

// New создаёт новый логгер в зависимости от режима logMode:
//   - local: логирование в stdout в текстовом формате
//   - dev: логирование в JSON-файл с уровнем debug
//   - prod: логирование в JSON-файл с уровнем info
func New(logMode string) (Logger, error) {
	const op = "logger.New()"

	var log *slog.Logger
	var file *os.File
	var err error

	switch logMode {
	case "local":
		log = slog.New(slog.NewTextHandler(
			os.Stdout,
			&slog.HandlerOptions{
				Level: slog.LevelDebug,
			},
		))

	case "dev":
		file, err = utils.GetLogFile(devLogPath)
		if err != nil {
			return Logger{}, err
		}

		log = slog.New(slog.NewJSONHandler(
			file,
			&slog.HandlerOptions{
				Level: slog.LevelDebug,
			},
		))

	case "prod":
		file, err = utils.GetLogFile(prodLogPath)
		if err != nil {
			return Logger{}, err
		}

		log = slog.New(slog.NewJSONHandler(
			file,
			&slog.HandlerOptions{
				Level: slog.LevelInfo,
			},
		))

	default:
		return Logger{}, fmt.Errorf("%s: %w", op, ErrUnknownLogMode)
	}

	return Logger{
		Log:  log,
		File: file,
	}, nil
}

// Shutdown корректно закрывает файл логов, если он был открыт. Возвращает ошибку, если файл не удалось закрыть.
func (l *Logger) Shutdown() error {
	const op = "logger.Shutdown()"

	if l.File != nil {
		err := l.File.Close()
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
	}
	return nil
}
