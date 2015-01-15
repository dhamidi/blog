package main

import "github.com/nu7hatch/gouuid"

type Aggregate interface {
	EventHandler
	CommandHandler
}

type CommandHandler interface {
	When(Command) (*Events, error)
}

type EventHandler interface {
	Apply(Event) error
}

type Event interface {
	Tag() string
	AggregateId() string
}
type EventList struct {
	items []Event
}

type Events EventList

var NoEvents = &Events{items: []Event{}}

func ListOfEvents(events ...Event) *Events {
	return &Events{items: events}
}

func (e *Events) Len() int { return len(e.items) }
func (e *Events) Append(events ...Event) *Events {
	e.items = append(e.items, events...)
	return e
}
func (e *Events) Items() []Event {
	return e.items
}

func (e *Events) ApplyTo(agg EventHandler) {
	for _, event := range e.items {
		agg.Apply(event)
	}
}

type Command interface {
	Validate() ValidationError
}

func Id() string {
	id, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}
	return id.String()
}
