package main

import (
	"log"
	"os"
	"testing"
)

func init() {
	if err := os.MkdirAll("_test/store", 0755); err != nil {
		log.Fatal(err)
	}
}

func TestEventsInFileSystem_restoresHistory(t *testing.T) {
	os.Remove("_test/store/seq")
	store, err := NewEventsInFileSystem("_test/store")
	store.Register(&PostPublishedEvent{})

	if err != nil {
		t.Fatal(err)
	}

	if err := store.HandleEvent(&PostPublishedEvent{
		PostId:  "post-1",
		Title:   "foo",
		Content: "bar",
	}); err != nil {
		t.Fatal(err)
	}

	if err := store.LoadHistory(); err != nil {
		t.Fatal(err)
	}

	events := store.byId["post-1"]
	if events == nil {
		t.Fatal("No events found for stream post-1")
	}

	_, ok := events[0].(*PostPublishedEvent)
	if !ok {
		t.Fatalf("Not a PostPublishedEvent: %#v\n", events[0])
	}
}
