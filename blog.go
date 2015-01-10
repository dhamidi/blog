package main

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"encoding/json"
	"errors"

	"github.com/nu7hatch/gouuid"
)

type Event interface{}
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
		messages = append(messages, fmt.Sprintf("%q", e.Error()))
	}

	return "[" + strings.Join(messages, ", ") + "]"
}

func (err Errors) Return() error {
	if len(err) == 0 {
		return nil
	} else {
		return err
	}
}

type ValidationError struct {
	fields map[string][]error
}

func NewValidationError() *ValidationError {
	return &ValidationError{
		fields: map[string][]error{},
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

type Post struct{}

type PostPublishedEvent struct {
	PostId      string
	Title       string
	Content     string
	PublishedAt time.Time
}

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

type Application struct{}

func (app *Application) HandleCommand(command Command) (Events, error) {
	switch cmd := command.(type) {
	case *PublishPostCommand:
		return app.PublishPost(cmd)
	}

	return nil, nil
}

func (app *Application) PublishPost(cmd *PublishPostCommand) (Events, error) {
	p := &Post{}
	return p.Publish(cmd.Title, cmd.Content)
}

func RunCommand(cmd Command, target CommandHandler, processor EventHandler) error {
	events, err := target.HandleCommand(cmd)
	if err != nil {
		return err
	}

	errors := Errors{}
	for _, event := range events {
		if err := processor.HandleEvent(event); err != nil {
			log.Printf("Error: %s\nWhile processing: %#v\n", err.Error(), event)
			errors = append(errors, err)
		}
	}

	return errors.Return()
}

type AllPostsPost struct {
	Id          string
	Title       string
	Published   string
	publishedAt time.Time
}

type AllPosts []*AllPostsPost

func (all AllPosts) Len() int           { return len(all) }
func (all AllPosts) Swap(i, j int)      { all[i], all[j] = all[j], all[i] }
func (all AllPosts) Less(i, j int) bool { return all[i].publishedAt.After(all[j].publishedAt) }

func (all AllPosts) GoString() string {
	posts := []string{}
	for _, post := range all {
		posts = append(posts, fmt.Sprintf("%#v", *post))
	}

	return fmt.Sprintf("%v", posts)
}

type AllPostsView struct {
	Posts AllPosts
}

func (view *AllPostsView) HandleEvent(event Event) error {
	switch evt := event.(type) {
	case *PostPublishedEvent:
		view.AddPost(&AllPostsPost{
			Id:          evt.PostId,
			Title:       evt.Title,
			Published:   evt.PublishedAt.Format("02 Jan 2006"),
			publishedAt: evt.PublishedAt,
		})
	}
	return nil
}

func (view *AllPostsView) AddPost(post *AllPostsPost) {
	view.Posts = append(view.Posts, post)
	sort.Sort(view.Posts)
}

func main() {

}
