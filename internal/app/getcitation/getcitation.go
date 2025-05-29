package getcitation

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	storage "getcitation/internal/storage/postgresql"
	"getcitation/internal/utils/config"
)

// Сообщения об ошибках для ответов HTTP
const (
	errMethodNotAllowed    string = "Method Not Allowed"
	errBadRequest          string = "Invalid Request Body"
	errInternalServerError string = "Internal Server Error"
	errNotFound            string = "Not Found"
	errConflict            string = "Conflict"
)

// Сообщения для конкретных ошибок в ответах
const (
	messageNoID               string = "ID must be present as query parameter"
	messageMalformedID        string = "ID parameter is malformed"
	messageQuoteNotFoundByID  string = "Quote with the provide ID doesn't exists"
	messageQuoteAlreadyExists string = "This quote already exists"
	messageQuotesNotFound     string = "No quotes found"
)

// Сообщения успешных операций
const (
	successDelete string = "Quote deleted successfully"
)

// Ошибки для внутреннего использования
var (
	ErrDuplicateEntry = fmt.Errorf("similar entry already exists")
	ErrNoQuotesFound  = fmt.Errorf("no quotes found")
)

// App представляет основное приложение с HTTP-сервером, логгером и конфигом
type App struct {
	Server Server
	Log    *slog.Logger
	Config config.Config
}

// Server содержит HTTP сервер и обработчики маршрутов
type Server struct {
	HTTPServer *http.Server
	Handlers   Handlers
}

// New создает и инициализирует новое приложение getcitation
func New(db storage.Storage, config config.Config, log *slog.Logger) App {
	const op = "getcitation.New()"

	service := Service{
		Log:    log,
		Config: config,

		Manipulator: db.DB.Handlers,
		Getter:      db.DB.Handlers,
	}

	handlers := Handlers{
		Log:    log,
		Config: config,

		Manipulator: service,
		Getter:      service,
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/quotes", handlers.GetAndCreateQuotes)
	mux.HandleFunc("/quotes/", handlers.DeleteQuoteByID)
	mux.HandleFunc("/quotes/random", handlers.GetRandomQuote)

	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", config.ServerHost, config.ServerPort),
		Handler:      mux,
		WriteTimeout: config.ServerWriteTimeout,
		ReadTimeout:  config.ServerReadTimeout,
		IdleTimeout:  config.ServerIdleTimeout,
	}

	return App{
		Server: Server{
			HTTPServer: server,
			Handlers:   handlers,
		},
		Log:    log,
		Config: config,
	}
}

// Run запускает HTTP сервер приложения и блокирует выполнение до его остановки
func (a App) Run() error {
	const op = "getcitation.Run()"

	err := a.Server.HTTPServer.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Shutdown корректно завершает работу HTTP сервера
func (a App) Shutdown() error {
	const op = "getcitation.Shutdown()"

	err := a.Server.HTTPServer.Shutdown(context.TODO())
	if err != nil {
		return err
	}
	return nil
}

// Интерфейс для манипуляций с цитатами (создание, удаление)
type ServiceManipulator interface {
	CreateQuote(author string, quote string) (int, error)
	DeleteQuoteByID(id int) error
}

// Интерфейс для получения цитат (рандомная, по автору)
type ServiceGetter interface {
	GetRandomQuote() (storage.Quote, error)
	GetQuotes(authorFilter string) ([]storage.Quote, error)
}

// Handlers содержит методы HTTP-обработчиков, использующих сервис
type Handlers struct {
	Log    *slog.Logger
	Config config.Config

	Manipulator ServiceManipulator
	Getter      ServiceGetter
}

// Error описывает структуру ошибки в формате JSON для ответов API
type Error struct {
	Status  Status `json:"status"`
	Message string `json:"message"`
}

// Status описывает статус HTTP ответа
type Status struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// CreateQuoteRequest описывает формат запроса на создание цитаты
type CreateQuoteRequest struct {
	ID     int    `json:"id"`
	Author string `json:"author"`
	Quote  string `json:"quote"`
}

// CreateQuoteResponse описывает формат успешного ответа при создании цитаты
type CreateQuoteResponse struct {
	Status Status `json:"status"`
	ID     int    `json:"id"`
}

// GetQuotesResponse описывает формат ответа при запросе списка цитат
type GetQuotesResponse struct {
	Status Status          `json:"status"`
	Quotes []storage.Quote `json:"quotes"`
}

// GetAndCreateQuotes обрабатывает HTTP запросы на получение списка цитат и создание новых
func (h Handlers) GetAndCreateQuotes(w http.ResponseWriter, r *http.Request) {
	const op = "getcitation.Transport.GetAndCreateQuotes()"

	switch r.Method {
	case http.MethodPost:
		var req CreateQuoteRequest

		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			h.Log.Error(
				errBadRequest,
				slog.String("op", op),
				slog.Any("error", err),
				slog.String("path", r.URL.Path),
			)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)

			json.NewEncoder(w).Encode(Error{
				Status: Status{
					Code:    http.StatusBadRequest,
					Message: errBadRequest,
				},
			})

			return
		}
		defer r.Body.Close()

		if req.Author == "" || req.Quote == "" {
			h.Log.Error(
				errBadRequest,
				slog.String("op", op),
				slog.Any("error", err),
				slog.String("path", r.URL.Path),
			)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)

			json.NewEncoder(w).Encode(Error{
				Status: Status{
					Code:    http.StatusBadRequest,
					Message: errBadRequest,
				},
			})

			return
		}

		id, err := h.Manipulator.CreateQuote(req.Author, req.Quote)
		if err != nil {
			if errors.Is(err, ErrDuplicateEntry) {
				h.Log.Error(
					errConflict,
					slog.String("op", op),
					slog.Any("error", err),
					slog.String("path", r.URL.Path),
				)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusConflict)

				json.NewEncoder(w).Encode(Error{
					Status: Status{
						Code:    http.StatusConflict,
						Message: errConflict,
					},
					Message: messageQuoteAlreadyExists,
				})

				return
			}
			h.Log.Error(
				errInternalServerError,
				slog.String("op", op),
				slog.Any("error", err),
				slog.String("path", r.URL.Path),
			)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)

			json.NewEncoder(w).Encode(Error{
				Status: Status{
					Code:    http.StatusInternalServerError,
					Message: errInternalServerError,
				},
			})

			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		json.NewEncoder(w).Encode(CreateQuoteResponse{
			Status: Status{
				Code: http.StatusOK,
			},
			ID: id,
		})

	case http.MethodGet:
		author := r.URL.Query().Get("author")

		quotes, err := h.Getter.GetQuotes(author)
		if err != nil {
			if errors.Is(err, ErrNoQuotesFound) {
				h.Log.Error(
					errNotFound,
					slog.String("op", op),
					slog.Any("error", err),
					slog.String("path", r.URL.Path),
				)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)

				json.NewEncoder(w).Encode(Error{
					Status: Status{
						Code:    http.StatusNotFound,
						Message: errNotFound,
					},
					Message: messageQuotesNotFound,
				})

				return
			}
			h.Log.Error(
				errInternalServerError,
				slog.String("op", op),
				slog.Any("error", err),
				slog.String("path", r.URL.Path),
			)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)

			json.NewEncoder(w).Encode(Error{
				Status: Status{
					Code:    http.StatusInternalServerError,
					Message: errInternalServerError,
				},
			})

			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		json.NewEncoder(w).Encode(GetQuotesResponse{
			Status: Status{
				Code: http.StatusOK,
			},
			Quotes: quotes,
		})

	default:
		h.Log.Error(
			errMethodNotAllowed,
			slog.String("op", op),
			slog.String("path", r.URL.Path),
		)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)

		json.NewEncoder(w).Encode(Error{
			Status: Status{
				Code:    http.StatusMethodNotAllowed,
				Message: errMethodNotAllowed,
			},
		})
	}
}

// DeleteQuoteByIDResponse описывает формат ответа при удалении цитаты
type DeleteQuoteByIDResponse struct {
	Status  Status `json:"status"`
	Message string `json:"message"`
}

// DeleteQuoteByID обрабатывает HTTP DELETE запрос на удаление цитаты по ID
func (h Handlers) DeleteQuoteByID(w http.ResponseWriter, r *http.Request) {
	const op = "getcitation.Transport.DeleteQuoteByID()"

	if r.Method != http.MethodDelete {
		h.Log.Error(
			errMethodNotAllowed,
			slog.String("op", op),
			slog.String("path", r.URL.Path),
		)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)

		json.NewEncoder(w).Encode(Error{
			Status: Status{
				Code:    http.StatusMethodNotAllowed,
				Message: errMethodNotAllowed,
			},
		})

		return
	}

	parts := strings.Split(r.URL.Path, "/")

	idStr := parts[2]
	if idStr == "" {
		h.Log.Error(
			errBadRequest,
			slog.String("op", op),
			slog.String("path", r.URL.Path),
		)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)

		json.NewEncoder(w).Encode(Error{
			Status: Status{
				Code:    http.StatusBadRequest,
				Message: errBadRequest,
			},
			Message: messageNoID,
		})

		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.Log.Error(
			errBadRequest,
			slog.String("op", op),
			slog.Any("error", err),
			slog.String("path", r.URL.Path),
		)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)

		json.NewEncoder(w).Encode(Error{
			Status: Status{
				Code:    http.StatusBadRequest,
				Message: errBadRequest,
			},
			Message: messageMalformedID,
		})

		return
	}

	err = h.Manipulator.DeleteQuoteByID(id)
	if err != nil {
		if errors.Is(err, ErrNoQuotesFound) {
			h.Log.Error(
				errNotFound,
				slog.String("op", op),
				slog.Any("error", err),
				slog.String("path", r.URL.Path),
			)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)

			json.NewEncoder(w).Encode(Error{
				Status: Status{
					Code:    http.StatusNotFound,
					Message: errNotFound,
				},
				Message: messageQuoteNotFoundByID,
			})

			return
		}
		h.Log.Error(
			errInternalServerError,
			slog.String("op", op),
			slog.Any("error", err),
			slog.String("path", r.URL.Path),
		)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)

		json.NewEncoder(w).Encode(Error{
			Status: Status{
				Code:    http.StatusInternalServerError,
				Message: errInternalServerError,
			},
		})

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(DeleteQuoteByIDResponse{
		Status: Status{
			Code: http.StatusOK,
		},
		Message: successDelete,
	})
}

// GetRandomQuoteResponse описывает формат ответа при получении случайной цитаты
type GetRandomQuoteResponse struct {
	Status Status        `json:"status"`
	Quote  storage.Quote `json:"quote"`
}

// GetRandomQuote обрабатывает HTTP GET запрос на получение случайной цитаты
func (h Handlers) GetRandomQuote(w http.ResponseWriter, r *http.Request) {
	const op = "getcitation.Transport.GetRandomQuote()"

	if r.Method != http.MethodGet {
		h.Log.Error(
			errMethodNotAllowed,
			slog.String("op", op),
			slog.String("path", r.URL.Path),
		)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)

		json.NewEncoder(w).Encode(Error{
			Status: Status{
				Code:    http.StatusMethodNotAllowed,
				Message: errMethodNotAllowed,
			},
		})

		return
	}

	quote, err := h.Getter.GetRandomQuote()
	if err != nil {
		h.Log.Error(
			errInternalServerError,
			slog.String("op", op),
			slog.Any("error", err),
			slog.String("path", r.URL.Path),
		)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)

		json.NewEncoder(w).Encode(Error{
			Status: Status{
				Code:    http.StatusInternalServerError,
				Message: errInternalServerError,
			},
		})

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(GetRandomQuoteResponse{
		Status: Status{
			Code: http.StatusOK,
		},
		Quote: quote,
	})
}

// DBManipulator описывает интерфейс для операций с БД, связанными с цитатами (создание, удаление)
type DBManipulator interface {
	CreateQuote(quote storage.Quote) (int, error)
	DeleteQuoteByID(id int) error
}

// DBGetter описывает интерфейс для получения цитат из БД
type DBGetter interface {
	GetRandomQuote() (storage.Quote, error)
	GetQuotes(authorFilter string) ([]storage.Quote, error)
}

// Service реализует бизнес-логику приложения — создание, удаление и получение цитат
type Service struct {
	Log    *slog.Logger
	Config config.Config

	Manipulator DBManipulator
	Getter      DBGetter
}

// CreateQuote создает новую цитату через слой хранилища и обрабатывает возможные ошибки дубликатов
func (s Service) CreateQuote(author string, quote string) (int, error) {
	const op = "getcitation.Service.CreateQuote()"

	id, err := s.Manipulator.CreateQuote(storage.Quote{
		Author: author,
		Quote:  quote,
	})
	if err != nil {
		if errors.Is(err, storage.ErrDuplicateEntry) {
			return 0, fmt.Errorf("%s: %w", op, ErrDuplicateEntry)
		}
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	return id, nil
}

// DeleteQuoteByID удаляет цитату по ID, возвращает ошибку, если цитата не найдена
func (s Service) DeleteQuoteByID(id int) error {
	const op = "getcitation.Service.DeleteQuoteByID()"

	err := s.Manipulator.DeleteQuoteByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%s: %w", op, ErrNoQuotesFound)
		}
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

// GetRandomQuote получает случайную цитату из хранилища
func (s Service) GetRandomQuote() (storage.Quote, error) {
	const op = "getcitation.Service.GetRandomQuote()"

	quote, err := s.Getter.GetRandomQuote()
	if err != nil {
		return storage.Quote{}, fmt.Errorf("%s: %w", op, err)
	}
	return quote, nil
}

// GetQuotes возвращает список цитат с возможным фильтром по автору
func (s Service) GetQuotes(authorFilter string) ([]storage.Quote, error) {
	const op = "getcitation.Service.GetQuotes()"

	quotes, err := s.Getter.GetQuotes(authorFilter)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, ErrNoQuotesFound)
		}
		return nil, err
	}
	return quotes, nil
}
