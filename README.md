# Мини-сервис Getcitation

Мини-сервис для хранения и управления цитатами. Реализован на Go с использованием стандартных библиотек и PostgreSQL.

## Описание

Данный сервис предоставляет REST API для создания, получения, фильтрации и удаления цитат.

## Функционал

### Добавление новой цитаты  

```bash
curl -X POST http://localhost:8080/quotes \ 
-H "Content-Type: application/json" \ 
-d '{"author":"Confucius", "quote":"Life is simple, but we insist on making it complicated."}'
```

### Получение всех цитат

```bash
curl http://localhost:8080/quotes
```

### Получение случайной цитаты

```bash
curl http://localhost:8080/quotes/random
```

### Фильтрация цитат по автору

```bash
curl http://localhost:8080/quotes?author=Confucius
```

### Удаление цитаты по ID

```bash
curl -X DELETE http://localhost:8080/quotes/1
```

## Запуск

**1. Клонируйте репозиторий:**

```bash
git clone https://github.com/romssc/getcitation
cd getcitation
```

**2. Настройте переменные окружения (нужно создать файл `.env` в корне проекта или отредактикровать существующий):**

```bash
APP_LOG_MODE=local

MIGRATIONS_PATH="migrations/postgresql"
MIGRATIONS_DIRECTION=up
MIGRATIONS_TABLE=migrations

SERVER_HOST=localhost
SERVER_PORT=8080
SERVER_READTIMEOUT=10s
SERVER_WRITETIMEOUT=10s
SERVER_IDLETIMEOUT=10s

POSTGRESQL_USERNAME=romssc
POSTGRESQL_PASSWORD=188696
POSTGRESQL_HOST=localhost
POSTGRESQL_PORT=5432
POSTGRESQL_DBNAME=postgres
POSTGRESQL_TABLE=quotes
POSTGRESQL_SSLMODE=disable
POSTGRESQL_EXTRA=
```

**3. Убедитесь, что PostgreSQL запущен и доступен с указанными параметрами.**

**4. Запустите миграцию базы данных (если база отсутствует):**

```bash
go run cmd/migrator/main.go
```

**5. Соберите и запустите приложение:**

```bash
go build -o build/getcitation cmd/getcitation/main.go
./build/getcitation
 ```

**6. По умолчанию сервис запущен на `http://localhost:8080`.**

## Технические детали

* Язык: Go
* Хранение данных: PostgreSQL (конфигируется через переменные окружения)
* Используемые библиотеки: стандартные библиотеки Go
* Логирование: пакет `slog`
* Конфигурация: через переменные окружения
* Валидация: базовая проверка на непустые поля

## 📄 License

[MIT](https://mit-license.org/)
