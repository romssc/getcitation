package main

import "getcitation/internal/app"

func main() {
	app, err := app.New()
	if err != nil {
		panic(err)
	}
	app.Run()
}
