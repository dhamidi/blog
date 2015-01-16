package main

import "github.com/nu7hatch/gouuid"

type Aggregate interface {
	EventHandler
	CommandHandler
}

type Type interface {
	New() Aggregate
}

type CommandHandler interface {
	HandleCommand(Command) (*Events, error)
}

type EventHandler interface {
	HandleEvent(Event) error
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
		agg.HandleEvent(event)
	}
}

type Command interface {
	Sanitize()
}

func Id() string {
	id, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}
	return id.String()
}
