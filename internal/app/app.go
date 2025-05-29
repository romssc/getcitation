package app

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"getcitation/internal/app/getcitation"
	"getcitation/internal/lib/logger"
	storage "getcitation/internal/storage/postgresql"
	"getcitation/internal/utils/config"
)

// App — основной объект приложения, агрегирующий все ключевые компоненты.
type App struct {
	GetCitation getcitation.App
	Storage     storage.Storage
	Log         logger.Logger
	Config      config.Config
}

// New — конструктор для App. Создаёт и инициализирует все зависимости приложения.
func New() (App, error) {
	const op = "app.New()"

	config, err := config.New()
	if err != nil {
		return App{}, fmt.Errorf("%s: %w", op, err)
	}

	logger, err := logger.New(config.AppLogMode)
	if err != nil {
		return App{}, fmt.Errorf("%s: %w", op, err)
	}

	storage, err := storage.New(config, logger.Log)
	if err != nil {
		return App{}, fmt.Errorf("%s: %w", op, err)
	}

	getcitation := getcitation.New(storage, config, logger.Log)

	return App{
		GetCitation: getcitation,
		Storage:     storage,
		Log:         logger,
		Config:      config,
	}, nil
}

// Run — запускает приложение, обрабатывает сигналы завершения и ошибки.
func (a App) Run() {
	const op = "app.Run()"

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	errChan := make(chan error, 1)

	a.Log.Log.Info(
		"запуск",
		slog.String("op", op),
	)

	go func() {
		err := a.GetCitation.Run()
		if err != nil {
			errChan <- err
		}
	}()

	select {
	case sig := <-sigChan:
		a.Log.Log.Error(
			"получен сигнал остановки",
			slog.String("op", op),
			slog.Any("signal", sig),
		)

	case err := <-errChan:
		a.Log.Log.Error(
			"ошибка привела к остановке",
			slog.String("op", op),
			slog.Any("error", err),
		)
	}

	a.Log.Log.Info(
		"остановка",
		slog.String("op", op),
	)

	errs := a.shutdown()
	if errs != nil {
		a.Log.Log.Error(
			"произошли ошибки во время остановки",
			slog.String("op", op),
			slog.Any("errors", errs),
		)
	}
}

// shutdown — корректно завершает работу всех компонентов приложения.
func (a App) shutdown() []error {
	const op = "app.shutdown()"

	var errs []error

	err := a.Storage.Shutdown()
	if err != nil {
		errs = append(errs, err)
	}

	err = a.Log.Shutdown()
	if err != nil {
		errs = append(errs, err)
	}

	err = a.GetCitation.Shutdown()
	if err != nil {
		errs = append(errs, err)
	}

	return errs
}
