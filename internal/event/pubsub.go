package event

import (
	"context"
	"reflect"
	"sync"
	`time`
	
	"github.com/sirupsen/logrus"
)

type Handler = func(ctx context.Context, ev interface{}) error

type Publisher interface {
	Publish(ctx context.Context, event interface{})
}

type Subscriber interface {
	Subscribe(evType interface{}, evHandler Handler)
}

// PubSub implements a simple event broker which allows to send event across the application.
type PubSub struct {
	mu  sync.Mutex
	log logrus.FieldLogger

	handlers map[reflect.Type][]Handler
}

func NewPubSub(log logrus.FieldLogger) *PubSub {
	return &PubSub{
		log:      log.WithField("source", "@pubsub"),
		handlers: make(map[reflect.Type][]Handler),
	}
}

func (b *PubSub) Publish(ctx context.Context, ev interface{}) {
	tt := reflect.TypeOf(ev)
	hList, found := b.handlers[tt]
	if found {
		for _, handler := range hList {
			go func(h Handler) {
				t := reflect.TypeOf(ev)
				err := h(ctx, ev)
				if err != nil {
					b.log.Errorf("error while calling event handler for %s type: %s", t.String(), err.Error())
				} else {
					b.log.Infof("event handler handled event for %s type with success at %s", t.String(), time.Now())
				}
			}(handler)
		}
	}
}

func (b *PubSub) Subscribe(evType interface{}, evHandler Handler) {
	tt := reflect.TypeOf(evType)
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, found := b.handlers[tt]; !found {
		b.handlers[tt] = []Handler{}
	}

	b.handlers[tt] = append(b.handlers[tt], evHandler)
}
