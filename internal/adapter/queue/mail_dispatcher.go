package queueinfra

import (
	"context"

	appmail "github.com/rifkifajarramadhani/golang-clean-architecture/internal/mail"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/queue"
)

type MailDispatcher struct {
	dispatcher queue.Dispatcher
}

func NewMailDispatcher(dispatcher queue.Dispatcher) *MailDispatcher {
	return &MailDispatcher{dispatcher: dispatcher}
}

func (d *MailDispatcher) DispatchMessage(ctx context.Context, job appmail.SendJob, options appmail.QueueOptions) (*appmail.QueuedMessageInfo, error) {
	info, err := d.dispatcher.Dispatch(ctx, mailJob{SendJob: job}, queue.DispatchOptions{
		Queue: options.Queue, ProcessAt: options.ProcessAt, MaxRetry: options.MaxRetry,
		Timeout: options.Timeout, UniqueFor: options.UniqueFor, Retention: options.Retention, TaskID: options.TaskID,
	})
	if err != nil {
		return nil, err
	}
	return &appmail.QueuedMessageInfo{ID: info.ID, Queue: info.Queue}, nil
}

type mailJob struct {
	appmail.SendJob
}

func (mailJob) Type() string   { return appmail.TypeSend }
func (j mailJob) Payload() any { return j.SendJob }

var _ appmail.QueuedMessageDispatcher = (*MailDispatcher)(nil)
