package eventstore

// Event defines the operations necessary for storing an event.
type Event interface {
	// Tag should return a string uniquely identifying the concrete
	// type of this event.  This string used for example when
	// deserializing events.
	Tag() string

	// AggregateId returns the id of the aggregate this event
	// belongs to.  This can be any string except "all".
	AggregateId() string
}

var (
	NoEvents = []Event{}
)

// Store defines the operations any event store needs to support.
type Store interface {
	// LoadAll loads all events from the store.  Usually this method
	// is called once on the event store, namely when the whole
	// application state needs to be reconstructed.
	LoadAll() ([]Event, error)

	// LoadStream loads all events belonging to the stream
	// identified by id.  The id "all" identifies a special stream
	// comprising all events.  Any error returned is of type
	// *StorageError.
	LoadStream(id string) ([]Event, error)

	// Store writes an event to the store.  The event is stored in a
	// stream identified by event.AggregateId().
	Store(event Event) error

	// RegisterType adds the concrete type of event to an internal
	// index used for deserialization.
	RegisterType(event Event)
}
