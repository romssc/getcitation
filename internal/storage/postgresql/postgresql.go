package postgresql

import (
	"database/sql"
	"fmt"
	"log/slog"

	_ "github.com/lib/pq"

	"getcitation/internal/utils/config"
)

type Storage struct {
	DB     DB
	Log    *slog.Logger
	Config config.Config
}

type Handlerer interface {
}

type DB struct {
	Implementation *sql.DB
	Handlers       Handlerer
}

func New(config config.Config, log *slog.Logger) (Storage, error) {
	const op = "postgresql.New()"

	var conn string

	switch {
	case config.PostgreSQLPassword == "":
		conn = fmt.Sprintf("postgres://%s@%s:%s/%s?sslmode=%s&&%s", config.PostgreSQLUsername, config.PostgreSQLHost, config.PostgreSQLPort, config.PostgreSQLDatabase, config.PostgreSQLSSL, config.PostgreSQLExtra)

	case config.PostgreSQLExtra == "":
		conn = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", config.PostgreSQLUsername, config.PostgreSQLPassword, config.PostgreSQLHost, config.PostgreSQLPort, config.PostgreSQLDatabase, config.PostgreSQLSSL)

	case config.PostgreSQLPassword == "" && config.PostgreSQLExtra == "":
		conn = fmt.Sprintf("postgres://%s@%s:%s/%s?sslmode=%s", config.PostgreSQLUsername, config.PostgreSQLHost, config.PostgreSQLPort, config.PostgreSQLDatabase, config.PostgreSQLSSL)

	default:
		conn = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s&&%s", config.PostgreSQLUsername, config.PostgreSQLPassword, config.PostgreSQLHost, config.PostgreSQLPort, config.PostgreSQLDatabase, config.PostgreSQLSSL, config.PostgreSQLExtra)
	}

	db, err := sql.Open("postgres", conn)
	if err != nil {
		return Storage{}, fmt.Errorf("%s: %w", op, err)
	}

	err = db.Ping()
	if err != nil {
		return Storage{}, fmt.Errorf("%s: %w", op, err)
	}

	return Storage{
		DB: DB{
			Implementation: db,
			Handlers: Handlers{
				DB:     db,
				Log:    log,
				Config: config,
			},
		},
		Log:    log,
		Config: config,
	}, nil
}

func (s Storage) Shutdown() error {
	const op = "postgresql.Shutdown()"

	err := s.DB.Implementation.Close()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

type Handlers struct {
	DB     *sql.DB
	Log    *slog.Logger
	Config config.Config
}
