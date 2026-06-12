package mail

import (
	"context"

	appmail "github.com/rifkifajarramadhani/golang-clean-architecture/internal/mail"
)

type Dispatcher struct{ mailer *appmail.Mailer }

func NewDispatcher(mailer *appmail.Mailer) *Dispatcher { return &Dispatcher{mailer: mailer} }

func (d *Dispatcher) DispatchWelcome(ctx context.Context, username, email string) error {
	_, err := d.mailer.Queue(ctx, appmail.Welcome{Username: username, Email: email}, appmail.QueueOptions{})
	return err
}
