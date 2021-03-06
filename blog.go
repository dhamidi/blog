package main

import (
	"fmt"
	"log"
	"os"

	"github.com/dhamidi/blog/eventstore"
)

type Application struct {
	Store eventstore.Store

	replaying bool

	types struct {
		posts *Posts
	}

	mailer Mailer

	observers []EventHandler

	processors []EventHandler

	views struct {
		allPosts *AllPostsView
		sitemap  *Sitemap
	}

	tls struct {
		enabled bool
		cert    string
		key     string
	}
}

func (app *Application) Init() error {
	app.Store.RegisterType(&PostPublishedEvent{})
	app.Store.RegisterType(&PostRewordedEvent{})
	app.Store.RegisterType(&PostCommentedEvent{})
	app.Store.RegisterType(&PostCommentAuthenticatedEvent{})

	app.types.posts = &Posts{}
	app.views.allPosts = &AllPostsView{}
	app.views.sitemap = NewSitemap(app.views.allPosts)

	if mailer, err := NewSystemMailer("/usr/sbin/sendmail"); err != nil {
		log.Fatal(err)
	} else {
		app.mailer = mailer
	}

	app.tls.key, app.tls.cert = os.Getenv("BLOG_TLS_KEY"), os.Getenv("BLOG_TLS_CERT")
	app.tls.enabled = app.tls.key != "" && app.tls.cert != ""

	app.processors = []EventHandler{
		&PostCommentProcessor{
			mailer: app.mailer,
			posts:  app.views.allPosts,
			useTls: app.tls.enabled,
		},
	}

	app.observers = []EventHandler{
		app.types.posts,
		app.views.allPosts,
		app.views.sitemap,
	}

	return app.replayState()
}

func (app *Application) replayState() error {
	events, err := app.Store.LoadAll()
	if err != nil {
		return fmt.Errorf("Application.replayState: %s\n", err)
	}

	app.replaying = true
	for _, event := range events {
		app.HandleEvent(event)
	}
	app.replaying = false
	return err
}

func (app *Application) load(typ Type, id string) (Aggregate, error) {
	aggregate := typ.New()
	events, err := app.Store.LoadStream(id)

	if err != nil {
		return nil, err
	}

	for _, event := range events {
		aggregate.HandleEvent(event)
	}

	return aggregate, nil
}

func (app *Application) HandleCommand(command Command) (*Events, error) {
	command.Sanitize()

	switch cmd := command.(type) {
	case *PublishPostCommand:
		return app.publishPost(cmd)
	case *PreviewPostCommand:
		return app.previewPost(cmd)
	case *RewordPostCommand:
		return app.rewordPost(cmd)
	case *CommentOnPostCommand:
		return app.commentOnPost(cmd)
	case *PostAuthenticateCommentCommand:
		return app.authenticateComment(cmd)
	}

	return NoEvents, nil
}

func (app *Application) publishPost(cmd *PublishPostCommand) (*Events, error) {
	post := app.types.posts.New()
	events, err := post.HandleCommand(cmd)

	if err != nil {
		return NoEvents, err
	} else {
		return events, app.process(events)
	}
}

func (app *Application) rewordPost(cmd *RewordPostCommand) (*Events, error) {
	post, err := app.load(app.types.posts, cmd.PostId)

	if err == ErrNotFound {
		return NoEvents, err
	}
	if err != nil {
		return NoEvents, fmt.Errorf("Application.load: %s\n", err)
	}

	events, err := post.HandleCommand(cmd)
	if err != nil {
		return NoEvents, err
	} else {
		return events, app.process(events)
	}
}

func (app *Application) commentOnPost(cmd *CommentOnPostCommand) (*Events, error) {
	post, err := app.load(app.types.posts, cmd.PostId)

	if err == ErrNotFound {
		return NoEvents, err
	}
	if err != nil {
		return NoEvents, fmt.Errorf("Application.load: %s\n", err)
	}

	events, err := post.HandleCommand(cmd)
	if err != nil {
		return NoEvents, err
	} else {
		return events, app.process(events)
	}
}

func (app *Application) authenticateComment(cmd *PostAuthenticateCommentCommand) (*Events, error) {
	cmd.postId = app.types.posts.IdForComment(cmd.CommentId)
	if cmd.postId == "" {
		return NoEvents, ErrNotFound
	}

	post, err := app.load(app.types.posts, cmd.postId)
	if err != nil {
		return NoEvents, fmt.Errorf("Application.load: %s\n", err)
	}

	events, err := post.HandleCommand(cmd)
	if err != nil {
		return NoEvents, err
	} else {
		return events, app.process(events)
	}
}

func (app *Application) previewPost(cmd *PreviewPostCommand) (*Events, error) {
	posts := &Posts{}
	post := posts.New()
	view := &AllPostsView{}
	events, err := post.HandleCommand(cmd.PublishPostCommand)
	if err != nil {
		return NoEvents, err
	}
	if err := view.HandleEvent(events.Items()[0]); err != nil {
		return NoEvents, err
	}

	viewPost := view.Collection[0]
	viewPost.Preview = true

	cmd.view = viewPost

	return events, nil
}

func (app *Application) HandleEvent(event Event) error {
	log.Printf("Application.HandleEvent: %#v\n", event)

	if err := app.storeEvent(event); err != nil {
		log.Printf("Application.HandleEvent: %s\n", err)
		return err
	}

	if err := app.notifyObservers(event); err != nil {
		return err
	}

	if err := app.notifyProcessors(event); err != nil {
		return err
	}

	return nil
}

func (app *Application) storeEvent(event Event) error {
	if !app.replaying {
		if err := app.Store.Store(event); err != nil {
			log.Printf("Application.storeEvent: %s\n", err)
			return err
		} else {
			log.Printf("Application.storeEvent: event stored\n")
		}
	}

	return nil
}

func (app *Application) notifyObservers(event Event) error {
	for _, observer := range app.observers {
		if err := observer.HandleEvent(event); err != nil {
			log.Printf("Application.notifyObservers: %s\nWhile processing:\n%#v\n", err, event)
			return err
		}
	}

	return nil
}

func (app *Application) notifyProcessors(event Event) error {
	if !app.replaying {
		for _, proc := range app.processors {
			if err := proc.HandleEvent(event); err != nil {
				log.Printf("Application.notifyProcessors: %s\n", err)
				return err
			}
		}
	}

	return nil
}

func (app *Application) process(events *Events) error {
	events.ApplyTo(app)
	return nil
}
