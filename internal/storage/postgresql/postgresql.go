// Пакет postgresql предоставляет функциональность для работы с БД.
package postgresql

import (
	"database/sql"
	"fmt"
	"log/slog"

	_ "github.com/lib/pq"

	"getcitation/internal/utils"
	"getcitation/internal/utils/config"
)

// Storage содержит подключение к БД и основные зависимости (логгер, конфиг).
type Storage struct {
	DB     DB
	Log    *slog.Logger
	Config config.Config
}

// Manipulator описывает методы для создания и удаления цитат.
type Manipulator interface {
	CreateQuote(quote Quote) (int, error)
	DeleteQuoteByID(id int) error
}

// Getter описывает методы для получения цитат.
type Getter interface {
	GetRandomQuote() (Quote, error)
	GetQuotes(authorFilter string) ([]Quote, error)
}

// DB содержит подключение к PostgreSQL и обработчики.
type DB struct {
	Implementation *sql.DB

	Manipulator Manipulator
	Getter      Getter
}

// New создаёт новое соединение с PostgreSQL.
func New(config config.Config, log *slog.Logger) (Storage, error) {
	const op = "postgresql.New()"

	conn := utils.BuildPostgreSQLDSN(config)

	db, err := sql.Open("postgres", conn)
	if err != nil {
		return Storage{}, fmt.Errorf("%s: %w", op, err)
	}

	err = db.Ping()
	if err != nil {
		return Storage{}, fmt.Errorf("%s: %w", op, err)
	}

	handlers := Handlers{
		DB:     db,
		Log:    log,
		Config: config,
	}

	return Storage{
		DB: DB{
			Implementation: db,

			Manipulator: handlers,
			Getter:      handlers,
		},
		Log:    log,
		Config: config,
	}, nil
}

// Shutdown корректно закрывает соединение с БД.
func (s Storage) Shutdown() error {
	const op = "postgresql.Shutdown()"

	err := s.DB.Implementation.Close()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

// Handlers — структура для реализации логики работы с конкретной таблицей или сущностью.
type Handlers struct {
	DB     *sql.DB
	Log    *slog.Logger
	Config config.Config
}

// Quote - объект цитаты.
type Quote struct {
	ID     int    `json:"id"`
	Author string `json:"author"`
	Quote  string `json:"quote"`
}

// CreateQuote добавляет новую цитату в базу.
func (h Handlers) CreateQuote(quote Quote) (int, error) {
	const op = "postgresql.CreateQuote()"

	tx, err := h.DB.Begin()
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	defer tx.Rollback()

	var id int

	err = tx.QueryRow(`INSERT INTO quotes (author, quote) VALUES ($1, $2) RETURNING id`, quote.Author, quote.Quote).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	err = tx.Commit()
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return id, nil
}

// DeleteQuoteByID удаляет цитату по ID.
func (h Handlers) DeleteQuoteByID(id int) error {
	const op = "postgresql.DeleteQuoteByID()"

	tx, err := h.DB.Begin()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	defer tx.Rollback()

	res, err := tx.Exec(`DELETE FROM quotes WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if affected == 0 {
		return fmt.Errorf("%s: quote with id %d not found", op, id)
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// GetRandomQuote получает случайную цитату.
func (h Handlers) GetRandomQuote() (Quote, error) {
	const op = "postgresql.GetRandomQuote()"

	tx, err := h.DB.Begin()
	if err != nil {
		return Quote{}, fmt.Errorf("%s: %w", op, err)
	}
	defer tx.Rollback()

	var quote Quote

	err = tx.QueryRow(`SELECT id, author, quote FROM quotes ORDER BY RANDOM() LIMIT 1`).Scan(&quote.ID, &quote.Author, &quote.Quote)
	if err != nil {
		return Quote{}, fmt.Errorf("%s: %w", op, err)
	}

	err = tx.Commit()
	if err != nil {
		return Quote{}, fmt.Errorf("%s: %w", op, err)
	}

	return quote, nil
}

// GetQuotes получает все цитаты, при необходимости фильтрует по автору.
func (h Handlers) GetQuotes(authorFilter string) ([]Quote, error) {
	const op = "postgresql.GetQuotes()"

	tx, err := h.DB.Begin()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer tx.Rollback()

	var rows *sql.Rows

	if authorFilter == "" {
		rows, err = tx.Query(`SELECT id, author, quote FROM quotes`)
	} else {
		rows, err = tx.Query(`SELECT id, author, quote FROM quotes WHERE author = $1`, authorFilter)
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	quotes := []Quote{}

	for rows.Next() {
		var quote Quote

		err := rows.Scan(&quote.ID, &quote.Author, &quote.Quote)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}

		quotes = append(quotes, quote)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	err = tx.Commit()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return quotes, nil
}
