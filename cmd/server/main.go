package main

import (
	"github.com/gofiber/fiber/v3"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/delivery/http/router"
)

func main() {
	app := fiber.New()

	router.Setup(app)

	app.Listen(":8080")
}
