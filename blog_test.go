package main

import (
	"fmt"
	"os"
	"reflect"
	"runtime"
	"testing"
	"time"
)

type mockEventHandler struct {
	Events Events
}

func (handler *mockEventHandler) HandleEvent(event Event) error {
	handler.Events = append(handler.Events, event)
	return nil
}

func assertRequired(t *testing.T, err error, field string) {
	if err == nil {
		t.Fatalf("%sexpected an error.", assertContext())
	}

	if verr, ok := err.(*ValidationError); !ok {
		t.Fatalf("Expected ValidationError, got %#v", err)
	} else {
		if msg := verr.Get(field); msg.Error() != "empty" {
			t.Fatalf("expected %q, got %q", "empty", msg)
		}
	}
}

func assertContext() string {
	_, file, line, ok := runtime.Caller(2)
	if ok {
		pwd, err := os.Getwd()
		if err != nil {
			panic(err)
		}

		return fmt.Sprintf(".%s:%d: ", file[len(pwd):], line)
	} else {
		return ""
	}
}

func assertEqual(t *testing.T, actual, expected interface{}) {
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("%sexpected %#v, got %#v", assertContext(), expected, actual)
	}
}

func TestPost_Publish_ReturnsEvent(t *testing.T) {
	post := &Post{}
	events, err := post.Publish("hello", "world")
	if err != nil {
		t.Fatal(err)
	}

	if published, ok := events[0].(*PostPublishedEvent); !ok {
		t.Fatalf("Unexpected event: %#v\n", events[0])
	} else {
		if published.Content != "world" {
			t.Fatalf("Expected content %q, got %q", "world", published.Content)
		}

		if published.Title != "hello" {
			t.Fatalf("Expected title %q, got %q", "hello", published.Title)
		}
	}
}

func TestPost_Publish_RequiresTitle(t *testing.T) {
	post := &Post{}
	_, err := post.Publish("", "world")
	assertRequired(t, err, "Title")
}

func TestPost_Publish_RequiresContent(t *testing.T) {
	post := &Post{}
	_, err := post.Publish("hello", "")
	assertRequired(t, err, "Content")
}

func TestPost_Publish_StripsSpacesFromTitle(t *testing.T) {
	post := &Post{}
	events, err := post.Publish("  hello  ", "  world  ")
	if err != nil {
		t.Fatal(err)
	}

	published := events[0].(*PostPublishedEvent)
	assertEqual(t, published.Title, "hello")
}

func TestPost_Publish_StripsSpacesFromContent(t *testing.T) {
	post := &Post{}
	events, err := post.Publish("hello", "  world  ")
	if err != nil {
		t.Fatal(err)
	}

	published := events[0].(*PostPublishedEvent)
	assertEqual(t, published.Content, "world")
}

func TestApplication_HandleCommand_PublishesPost(t *testing.T) {
	app := &Application{}
	handler := &mockEventHandler{}

	RunCommand(&PublishPostCommand{
		Title:   "  hello  ",
		Content: "  world  ",
	}, app, handler)

	_ = handler.Events[0].(*PostPublishedEvent)
}
func TestAllPostsView_SortsByPublishedAtDescending(t *testing.T) {
	view := &AllPostsView{}
	ids := []string{Id(), Id()}
	dates := []time.Time{
		time.Date(2015, 1, 1, 10, 0, 0, 0, time.UTC),
		time.Date(2015, 1, 2, 10, 0, 0, 0, time.UTC),
	}

	events := Events{
		&PostPublishedEvent{
			PostId:      ids[0],
			Title:       "First post",
			PublishedAt: dates[0],
		},
		&PostPublishedEvent{
			PostId:      ids[1],
			Title:       "Second post",
			PublishedAt: dates[1],
		},
	}

	for _, event := range events {
		view.HandleEvent(event)
	}

	assertEqual(t, view.Posts, AllPosts{
		{Id: ids[1], Title: "Second post", Published: "02 Jan 2015", publishedAt: dates[1]},
		{Id: ids[0], Title: "First post", Published: "01 Jan 2015", publishedAt: dates[0]},
	})
}
