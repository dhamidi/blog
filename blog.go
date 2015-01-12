package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"encoding/json"
	"errors"

	"github.com/dhamidi/set"
	"github.com/nu7hatch/gouuid"
)

type Event interface {
	EventStreamId() string
}
type Command interface{}

type Events []Event

type CommandHandler interface {
	HandleCommand(Command) (Events, error)
}

type EventHandler interface {
	HandleEvent(Event) error
}

type Errors []error

func (err Errors) Error() string {
	messages := []string{}
	for _, e := range err {
		messages = append(messages, e.Error())
	}

	data, _ := json.MarshalIndent(messages, "", "  ")

	return string(data)
}

func (err Errors) MarshalJSON() ([]byte, error) {
	return []byte(err.Error()), nil
}

func (err Errors) Return() error {
	if len(err) == 0 {
		return nil
	} else {
		return err
	}
}

type ValidationError struct {
	fields map[string]Errors
}

func NewValidationError() *ValidationError {
	return &ValidationError{
		fields: map[string]Errors{},
	}
}

func (err *ValidationError) Get(field string) error {
	errors := err.fields[field]
	if len(errors) == 0 {
		return nil
	}

	return errors[0]
}

func (err *ValidationError) Put(field string, value error) *ValidationError {
	err.fields[field] = append(err.fields[field], value)
	return err
}

func (err *ValidationError) Return() error {
	if len(err.fields) == 0 {
		return nil
	} else {
		return err
	}
}

func (err *ValidationError) Error() string {
	data, er := json.MarshalIndent(err.fields, "", "  ")
	if er != nil {
		panic(er)
	}

	return string(data)
}

type Posts struct {
	titles *set.String
}

func NewPosts() *Posts {
	return &Posts{titles: &set.String{}}
}

func (p *Posts) HandleEvent(event Event) error {
	switch evt := event.(type) {
	case *PostPublishedEvent:
		p.titles.Add(evt.Title)
	}

	return nil
}

func (p *Posts) UniqueTitle(title string) bool {
	return !p.titles.Contains(title)
}

type Post struct {
	posts *Posts
}

type PostPublishedEvent struct {
	PostId      string
	Title       string
	Content     string
	PublishedAt time.Time
}

func (p *PostPublishedEvent) EventStreamId() string { return p.PostId }

func Id() string {
	id, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}
	return id.String()
}

func (p *Post) Publish(title, content string) (Events, error) {
	validation := NewValidationError()

	title = strings.TrimSpace(title)
	content = strings.TrimSpace(content)

	if len(title) == 0 {
		validation.Put("Title", errors.New("empty"))
	}

	if len(content) == 0 {
		validation.Put("Content", errors.New("empty"))
	}

	if !p.posts.UniqueTitle(title) {
		validation.Put("Title", errors.New("exists"))
	}

	return Events{
		&PostPublishedEvent{
			PostId:      Id(),
			Content:     content,
			Title:       title,
			PublishedAt: time.Now(),
		},
	}, validation.Return()
}

type PublishPostCommand struct {
	Title   string
	Content string
}

type Application struct {
	allPosts      *AllPostsView
	postDetail    *PostDetailView
	posts         *Posts
	eventStore    *EventsInFileSystem
	eventHandlers []EventHandler
}

func (app *Application) Init() error {
	app.allPosts = &AllPostsView{}
	app.postDetail = &PostDetailView{}
	app.posts = NewPosts()
	eventStore, err := NewEventsInFileSystem("_events")
	if err != nil {
		return fmt.Errorf("Application.Init: %s", err)
	}
	eventStore.Register(&PostPublishedEvent{})
	app.eventStore = eventStore

	app.eventHandlers = []EventHandler{
		app.allPosts,
		app.postDetail,
		app.posts,
	}

	return app.restoreState()
}

func (app *Application) restoreState() error {
	if err := app.eventStore.LoadHistory(); err != nil {
		return fmt.Errorf("Application.Init: %s", err)
	}

	state, err := app.eventStore.AllEvents()
	if err != nil {
		return fmt.Errorf("Application.Init: %s", err)
	}
	for _, event := range state {
		if err := app.HandleEvent(event); err != nil {
			return err
		}
	}

	return nil
}

func (app *Application) HandleEvent(event Event) error {
	errs := Errors{}
	for _, handler := range app.eventHandlers {
		if err := handler.HandleEvent(event); err != nil {
			errs = append(errs, err)
		}
	}

	return errs.Return()
}

func (app *Application) HandleCommand(command Command) (Events, error) {
	var events Events
	var err error

	errs := Errors{}

	switch cmd := command.(type) {
	case *PublishPostCommand:
		events, err = app.PublishPost(cmd)
	}

	if err != nil {
		return nil, err
	}

	for _, event := range events {
		if err := app.eventStore.HandleEvent(event); err != nil {
			errs = append(errs, err)
		}

		if err := app.HandleEvent(event); err != nil {
			errs = append(errs, err)
		}
	}

	return events, errs.Return()
}

func (app *Application) PublishPost(cmd *PublishPostCommand) (Events, error) {
	p := &Post{posts: app.posts}
	return p.Publish(cmd.Title, cmd.Content)
}

func main() {

	app := &Application{}
	if err := app.Init(); err != nil {
		log.Fatal(err)
	}

	http.Handle("/posts/", app.postDetail)
	http.HandleFunc("/posts", func(w http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case "GET":
			app.allPosts.ServeHTTP(w, req)
		default:
			app.ServeHTTP(w, req)
		}
	})

	log.Fatal(http.ListenAndServe(":8000", nil))
}
