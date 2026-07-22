package main

import (
	"context"

	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/config"
)

type searchReadiness func(context.Context) error

func openSearch(context.Context, *config.Config) (searchReadiness, error) {
	return func(context.Context) error { return nil }, nil
}
