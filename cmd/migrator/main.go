package main

import (
	"fmt"

	"github.com/golang-migrate/migrate/v4"

	"getcitation/internal/utils"
	"getcitation/internal/utils/config"
)

const (
	directionUp   = "up"
	directionDown = "down"
)

func main() {
	config, err := config.New()
	if err != nil {
		panic(err)
	}

	conn := utils.BuildPostgreSQLDSN(config)

	m, err := migrate.New("file://"+config.MigrationsPath, conn)
	if err != nil {
		panic(err)
	}

	switch config.MigrationsDirection {
	case directionUp:
		err := m.Up()
		if err != nil {
			panic(err)
		}

	case directionDown:
		err := m.Down()
		if err != nil {
			panic(err)
		}

	default:
		panic("unknown direction")
	}

	fmt.Println("миграция завершена")
}
