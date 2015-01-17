package main

import (
	"fmt"
	"log"
	"net/http"
)

type Application struct {
	Store *FileStore

	replaying bool

	types struct {
		posts *Posts
	}

	mailer Mailer

	observers []EventHandler

	processors []EventHandler

	views struct {
		allPosts *AllPostsJSONView
	}
}

func (app *Application) Init() error {
	app.Store.RegisterType(&PostPublishedEvent{})
	app.Store.RegisterType(&PostRewordedEvent{})
	app.Store.RegisterType(&PostCommentedEvent{})
	app.Store.RegisterType(&PostCommentAuthenticatedEvent{})

	app.types.posts = &Posts{}
	app.views.allPosts = &AllPostsJSONView{}

	if mailer, err := NewSystemMailer("/usr/sbin/sendmail"); err != nil {
		log.Fatal(err)
	} else {
		app.mailer = mailer
	}

	app.processors = []EventHandler{
		&PostCommentProcessor{
			mailer: app.mailer,
			posts:  app.views.allPosts,
		},
	}

	app.observers = []EventHandler{
		app.types.posts,
		app.views.allPosts,
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

func (app *Application) load(typ Type, id string) (Aggregate, error) {
	aggregate := typ.New()
	events, err := app.Store.LoadStream(id)
	if err != nil {
		return nil, err
	}

	events.ApplyTo(aggregate)

	return aggregate, nil
}

func (app *Application) HandleCommand(command Command) (*Events, error) {
	command.Sanitize()

	switch cmd := command.(type) {
	case *PublishPostCommand:
		return app.publishPost(cmd)
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
	post, err := app.load(app.types.posts, cmd.PostId)
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

func (app *Application) HandleEvent(event Event) error {
	for _, observer := range app.observers {
		if err := observer.HandleEvent(event); err != nil {
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

		app.HandleEvent(event)

		if !app.replaying {
			for _, proc := range app.processors {
				if err := proc.HandleEvent(event); err != nil {
					log.Printf("Application.process: %s\n", err)
				}
			}
		}
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

	http.HandleFunc("/comments/", func(w http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case "GET":
			id := req.URL.Path[len("/comments/"):]
			cmd := &PostAuthenticateCommentCommand{
				PostId:    req.FormValue("post_id"),
				CommentId: id,
			}

			if _, err := app.HandleCommand(cmd); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
			} else {
				w.WriteHeader(http.StatusNoContent)
			}
		default:
			http.Error(w, "Only GET is allowed.", http.StatusMethodNotAllowed)

		}
	})

	http.HandleFunc("/comments", func(w http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case "POST":
			cmd := &CommentOnPostCommand{
				PostId:  req.FormValue("post_id"),
				Author:  req.FormValue("author"),
				Email:   req.FormValue("email"),
				Content: req.FormValue("content"),
			}

			if _, err := app.HandleCommand(cmd); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
			} else {
				w.Header().Set("Location", fmt.Sprintf("/posts/%s", cmd.PostId))
				w.WriteHeader(http.StatusSeeOther)
			}
		default:
			http.Error(w, "Only POST is allowed.", http.StatusMethodNotAllowed)

		}
	})

	http.HandleFunc("/posts", func(w http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case "GET":
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Write(app.views.allPosts.Render())
		case "POST":
			cmd := &PublishPostCommand{
				Title:   req.FormValue("title"),
				Content: req.FormValue("content"),
			}
			if _, err := app.HandleCommand(cmd); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
			} else {
				w.Header().Set("Location", fmt.Sprintf("/posts/%s", cmd.postId))
				w.WriteHeader(http.StatusCreated)
			}
		default:
			http.Error(w, "Only POST is allowed.", http.StatusMethodNotAllowed)
		}
	})
	log.Fatal(http.ListenAndServe(":8000", nil))
}
