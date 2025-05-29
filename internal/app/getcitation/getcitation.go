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

const (
	errMethodNotAllowed    string = "Method Not Allowed"
	errBadRequest          string = "Invalid Request Body"
	errInternalServerError string = "Internal Server Error"
	errNotFound            string = "Not Found"
	errConflict            string = "Conflict"
)

const (
	messageNoID               string = "ID must be present as query parameter"
	messageMalformedID        string = "ID parameter is malformed"
	messageQuoteNotFoundByID  string = "Quote with the provide ID doesn't exists"
	messageQuoteAlreadyExists string = "This quote already exists"
	messageQuotesNotFound     string = "No quotes found"
)

const (
	successDelete string = "Quote deleted successfully"
)

var (
	ErrDuplicateEntry = fmt.Errorf("similar entry already exists")
	ErrNoQuotesFound  = fmt.Errorf("no quotes found")
)

type App struct {
	Server Server
	Log    *slog.Logger
	Config config.Config
}

type Server struct {
	HTTPServer *http.Server
	Handlers   Handlers
}

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

func (a App) Run() error {
	const op = "getcitation.Run()"

	err := a.Server.HTTPServer.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (a App) Shutdown() error {
	const op = "getcitation.Shutdown()"

	err := a.Server.HTTPServer.Shutdown(context.TODO())
	if err != nil {
		return err
	}
	return nil
}

type ServiceManipulator interface {
	CreateQuote(author string, quote string) (int, error)
	DeleteQuoteByID(id int) error
}

type ServiceGetter interface {
	GetRandomQuote() (storage.Quote, error)
	GetQuotes(authorFilter string) ([]storage.Quote, error)
}

type Handlers struct {
	Log    *slog.Logger
	Config config.Config

	Manipulator ServiceManipulator
	Getter      ServiceGetter
}

type Error struct {
	Status  Status `json:"status"`
	Message string `json:"message"`
}

type Status struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type CreateQuoteRequest struct {
	ID     int    `json:"id"`
	Author string `json:"author"`
	Quote  string `json:"quote"`
}

type CreateQuoteResponse struct {
	Status Status `json:"status"`
	ID     int    `json:"id"`
}

type GetQuotesResponse struct {
	Status Status          `json:"status"`
	Quotes []storage.Quote `json:"quotes"`
}

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

type DeleteQuoteByIDResponse struct {
	Status  Status `json:"status"`
	Message string `json:"message"`
}

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

type GetRandomQuoteResponse struct {
	Status Status        `json:"status"`
	Quote  storage.Quote `json:"quote"`
}

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

type DBManipulator interface {
	CreateQuote(quote storage.Quote) (int, error)
	DeleteQuoteByID(id int) error
}

type DBGetter interface {
	GetRandomQuote() (storage.Quote, error)
	GetQuotes(authorFilter string) ([]storage.Quote, error)
}

type Service struct {
	Log    *slog.Logger
	Config config.Config

	Manipulator DBManipulator
	Getter      DBGetter
}

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

func (s Service) GetRandomQuote() (storage.Quote, error) {
	const op = "getcitation.Service.GetRandomQuote()"

	quote, err := s.Getter.GetRandomQuote()
	if err != nil {
		return storage.Quote{}, fmt.Errorf("%s: %w", op, err)
	}
	return quote, nil
}

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
