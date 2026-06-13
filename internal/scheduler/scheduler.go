package scheduler

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/queue"
)

type JobFactory func() queue.Job

type Schedule interface {
	Next(time.Time) time.Time
}

type Parser interface {
	Parse(string, *time.Location) (Schedule, error)
}

type Definition struct {
	Name            string
	Cron            string
	Timezone        string
	Job             JobFactory
	DispatchOptions queue.DispatchOptions
	schedule        Schedule
}

type Registry struct {
	defaultLocation *time.Location
	parser          Parser
	definitions     []Definition
}

func NewRegistry(defaultTimezone string, parser Parser) (*Registry, error) {
	location, err := time.LoadLocation(defaultTimezone)
	if err != nil {
		return nil, fmt.Errorf("load scheduler timezone: %w", err)
	}
	if parser == nil {
		return nil, errors.New("schedule parser is required")
	}
	return &Registry{defaultLocation: location, parser: parser}, nil
}

func (r *Registry) Register(definition Definition) error {
	if definition.Name == "" || definition.Cron == "" || definition.Job == nil {
		return errors.New("schedule name, cron, and job are required")
	}
	for _, existing := range r.definitions {
		if existing.Name == definition.Name {
			return fmt.Errorf("schedule %q already registered", definition.Name)
		}
	}

	location := r.defaultLocation
	if definition.Timezone != "" {
		var err error
		location, err = time.LoadLocation(definition.Timezone)
		if err != nil {
			return fmt.Errorf("load timezone for schedule %q: %w", definition.Name, err)
		}
	}

	parsed, err := r.parser.Parse(definition.Cron, location)
	if err != nil {
		return fmt.Errorf("parse schedule %q: %w", definition.Name, err)
	}
	definition.Timezone = location.String()
	definition.schedule = parsed
	r.definitions = append(r.definitions, definition)
	sort.Slice(r.definitions, func(i, j int) bool {
		return r.definitions[i].Name < r.definitions[j].Name
	})
	return nil
}

func (r *Registry) Definitions() []Definition {
	return append([]Definition(nil), r.definitions...)
}

func (r *Registry) Due(at time.Time) []Definition {
	minute := at.UTC().Truncate(time.Minute)
	previous := minute.Add(-time.Minute)
	var due []Definition
	for _, definition := range r.definitions {
		if definition.schedule.Next(previous).Equal(minute) {
			due = append(due, definition)
		}
	}
	return due
}

func (r *Registry) Next(definition Definition, after time.Time) time.Time {
	return definition.schedule.Next(after)
}

type Runner struct {
	registry   *Registry
	dispatcher queue.Dispatcher
}

func NewRunner(registry *Registry, dispatcher queue.Dispatcher) *Runner {
	return &Runner{registry: registry, dispatcher: dispatcher}
}

func (r *Runner) Run(ctx context.Context, at time.Time) error {
	minute := at.UTC().Truncate(time.Minute)
	var dispatchErrors []error
	for _, definition := range r.registry.Due(minute) {
		options := definition.DispatchOptions
		options.TaskID = TaskID(definition.Name, minute)
		if options.Retention < 2*time.Minute {
			options.Retention = 2 * time.Minute
		}
		_, err := r.dispatcher.Dispatch(ctx, definition.Job(), options)
		if err != nil && !errors.Is(err, queue.ErrDuplicateJob) {
			dispatchErrors = append(dispatchErrors, fmt.Errorf("dispatch schedule %q: %w", definition.Name, err))
		}
	}
	return errors.Join(dispatchErrors...)
}

func TaskID(name string, minute time.Time) string {
	return fmt.Sprintf("schedule:%s:%s", name, minute.UTC().Format("20060102T1504"))
}
