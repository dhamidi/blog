package main

import (
	"fmt"
	"log"
)

type Application struct {
	Store *FileStore

	replaying bool

	types struct {
		posts *Posts
	}

	observers []EventHandler
}

func (app *Application) Init() error {
	app.Store.RegisterType(&PostPublishedEvent{})
	app.Store.RegisterType(&PostRewordedEvent{})
	app.Store.RegisterType(&PostCommentedEvent{})

	app.types.posts = &Posts{}

	app.observers = []EventHandler{
		app.types.posts,
	}

	return app.replayState()
}

func (app *Application) replayState() error {
	events, err := app.Store.LoadAll()
	if err != nil {
		return fmt.Errorf("Application.replayState: %s\n", err)
	}

	app.replaying = true
	err = app.process(events)
	app.replaying = false
	return err
}

func (app *Application) load(aggregate EventHandler, id string) error {
	events, err := app.Store.LoadStream(id)
	if err != nil {
		return err
	}

	events.ApplyTo(aggregate)

	return nil
}

func (app *Application) PublishPost(cmd *PublishPostCommand) error {
	cmd.postId = Id()
	post := app.types.posts.New()
	events, err := post.When(cmd)

	if err != nil {
		return err
	} else {
		return app.process(events)
	}
}

func (app *Application) RewordPost(cmd *RewordPostCommand) error {
	post := app.types.posts.New()
	if err := app.load(post, cmd.PostId); err != nil {
		return fmt.Errorf("Application.load: %s\n", err)
	}

	events, err := post.When(cmd)
	if err != nil {
		return err
	} else {
		return app.process(events)
	}
}

func (app *Application) CommentOnPost(cmd *CommentOnPostCommand) error {
	post := app.types.posts.New()
	if err := app.load(post, cmd.PostId); err != nil {
		return fmt.Errorf("Application.load: %s\n", err)
	}

	events, err := post.When(cmd)
	if err != nil {
		return err
	} else {
		return app.process(events)
	}
}

func (app *Application) Apply(event Event) error {
	for _, observer := range app.observers {
		if err := observer.Apply(event); err != nil {
			log.Printf("Application.Apply: %s\nWhile processing:\n%#v\n", err, event)
		}
	}

	return nil
}

func (app *Application) process(events *Events) error {
	for _, event := range events.Items() {
		log.Printf("Application.process: %#v\n", event)

		if !app.replaying {
			if err := app.Store.Store(event); err != nil {
				log.Printf("Application.process: %s\n", err)
			} else {
				log.Printf("Application.process: event stored\n")
			}
		}

		app.Apply(event)
	}

	return nil
}

func main() {
	store, err := NewFileStore("_events")
	if err != nil {
		log.Fatal(err)
	}

	app := Application{Store: store}
	if err := app.Init(); err != nil {
		log.Fatal(err)
	}

	if err := app.PublishPost(&PublishPostCommand{
		Title:   "hello",
		Content: "world",
	}); err != nil {
		if verr, ok := err.(ValidationError); ok {
			log.Println(verr)
		} else {
			log.Fatal(err)
		}
	}
}
