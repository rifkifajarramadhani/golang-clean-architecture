package cron

import (
	"time"

	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/scheduler"
	"github.com/robfig/cron/v3"
)

type Parser struct{}

func (Parser) Parse(expression string, location *time.Location) (scheduler.Schedule, error) {
	return cron.ParseStandard("CRON_TZ=" + location.String() + " " + expression)
}

var _ scheduler.Parser = Parser{}
