package config

import (
	"log"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"

	"github.com/m1ll3r1337/order-service/internal/app/config/section"
)

type Config struct {
	Repository section.Repository `required:"true"`
	Monitor    section.Monitor    `required:"true"`
	Processor  section.Processor  `required:"true"`
}

var Root Config

func Load() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found")
	}

	if err := envconfig.Process("APP", &Root); err != nil {
		log.Fatalf("could not load config: %v", err)
	}
}
