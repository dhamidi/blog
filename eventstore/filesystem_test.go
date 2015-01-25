package eventstore_test

import (
	"os"
	"reflect"
	"testing"

	"github.com/dhamidi/blog/eventstore"
)

type E map[string]interface{}

func (e E) Tag() string {
	return "event"
}

func (e E) AggregateId() string {
	if id, ok := e["aggregate_id"]; ok {
		return id.(string)
	} else {
		return "any"
	}
}

func assertEquals(t *testing.T, a, b interface{}) {
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("%10s:\n  %#v\n%10s:\n  %#v\n",
			"Expected", b,
			"Got", a,
		)
	}
}

func withTestStore(t *testing.T, test func(store eventstore.Store)) {
	store, err := eventstore.NewOnDisk("_test_store")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll("_test_store")

	test(store)
}

func TestOnDisk_LoadAll_DoesNotReturnAnError(t *testing.T) {
	withTestStore(t, func(store eventstore.Store) {
		_, err := store.LoadAll()
		if err != nil {
			t.Fatal(err)
		}
	})
}

func TestOnDisk_LoadStream_ReturnsErrorForUnknownStream(t *testing.T) {
	withTestStore(t, func(store eventstore.Store) {
		_, err := store.LoadStream("does-not-exist")
		if !eventstore.IsNotFound(err) {
			t.Fatal("Wrong error: %s\n", err)
		}
	})
}

func TestOnDisk_LoadStream_ReturnsStoredEventsInOrder(t *testing.T) {
	withTestStore(t, func(store eventstore.Store) {
		events := []eventstore.Event{
			&E{
				"aggregate_id": "aggregate-id-1",
				"data":         "a",
			},
			&E{
				"aggregate_id": "aggregate-id-2",
				"data":         "b",
			},
			&E{
				"aggregate_id": "aggregate-id-1",
				"data":         "c",
			},
		}

		store.RegisterType(&E{})

		for _, event := range events {
			if err := store.Store(event); err != nil {
				t.Fatal(err)
			}
		}

		loadedEvents, err := store.LoadStream("aggregate-id-1")
		if err != nil {
			t.Fatal(err)
		}

		assertEquals(t, loadedEvents, []eventstore.Event{events[0], events[2]})
	})
}

func TestOnDisk_LoadAll_ReturnsAllStoredEventsInOrder(t *testing.T) {
	withTestStore(t, func(store eventstore.Store) {
		events := []eventstore.Event{
			&E{
				"aggregate_id": "aggregate-id-1",
				"data":         "a",
			},
			&E{
				"aggregate_id": "aggregate-id-2",
				"data":         "b",
			},

			&E{
				"aggregate_id": "aggregate-id-1",
				"data":         "c",
			},
		}

		store.RegisterType(&E{})

		for _, event := range events {
			if err := store.Store(event); err != nil {
				t.Fatal(err)
			}
		}

		loadedEvents, err := store.LoadAll()
		if err != nil {
			t.Fatal(err)
		}

		assertEquals(t, loadedEvents, events)
	})
}
