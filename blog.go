package main

import (
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"encoding/json"
	"errors"

	"github.com/dhamidi/set"
	"github.com/nu7hatch/gouuid"
	"github.com/russross/blackfriday"
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
	allPosts   *AllPostsView
	postDetail *PostDetailView
	posts      *Posts
}

func (app *Application) HandleEvent(event Event) error {
	errs := Errors{}
	if err := app.allPosts.HandleEvent(event); err != nil {
		errs = append(errs, err)
	}

	if err := app.postDetail.HandleEvent(event); err != nil {
		errs = append(errs, err)
	}

	if err := app.posts.HandleEvent(event); err != nil {
		errs = append(errs, err)
	}

	return errs.Return()
}

func (app *Application) HandleCommand(command Command) (Events, error) {
	switch cmd := command.(type) {
	case *PublishPostCommand:
		return app.PublishPost(cmd)
	}

	return nil, nil
}

func (app *Application) PublishPost(cmd *PublishPostCommand) (Events, error) {
	p := &Post{posts: app.posts}
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

type PostDetailPost struct {
	Id        string
	Title     string
	Body      string
	BodyHTML  string
	Published string
}

type PostDetailView struct {
	Posts map[string]*PostDetailPost
}

func (view *PostDetailView) HandleEvent(event Event) error {
	switch evt := event.(type) {
	case *PostPublishedEvent:

		view.AddPost(&PostDetailPost{
			Id:        evt.PostId,
			Title:     evt.Title,
			Body:      evt.Content,
			BodyHTML:  string(blackfriday.MarkdownCommon([]byte(evt.Content))),
			Published: evt.PublishedAt.Format("02 Jan 2006"),
		})
	}

	return nil
}

func (view *PostDetailView) AddPost(post *PostDetailPost) {
	if view.Posts == nil {
		view.Posts = map[string]*PostDetailPost{}
	}

	view.Posts[post.Id] = post
}

func main() {
	app := &Application{
		allPosts:   &AllPostsView{},
		postDetail: &PostDetailView{},
		posts:      NewPosts(),
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
