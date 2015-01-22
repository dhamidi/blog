package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	// expvar is imported for registering its HTTP handler.

	_ "expvar"
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
		allPosts *AllPostsView
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
	post := app.types.posts.New()
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

func respondWithError(w http.ResponseWriter, err error) {
	switch err {
	case ErrNotFound:
		http.Error(w, err.Error(), http.StatusNotFound)
	default:
		if strings.HasPrefix(err.Error(), "ValidationError") {
			http.Error(w, err.Error(), http.StatusBadRequest)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func authenticated(w http.ResponseWriter, req *http.Request) bool {
	user, pass, ok := req.BasicAuth()
	if !ok || !validUser(user, pass) {
		w.Header().Set("WWW-Authenticate", "Basic realm=\"Administration\"")
		http.Error(w, "Login required.", http.StatusUnauthorized)
		return false
	}

	return true
}

func validUser(user, pass string) bool {
	expectedUser := os.Getenv("BLOG_ADMIN_USER")
	expectedPass := os.Getenv("BLOG_ADMIN_PASS")

	if expectedUser == "" || expectedPass == "" {
		return false
	}

	return user == expectedUser && pass == expectedPass
}

func main() {
	assetServer := http.FileServer(http.Dir("assets"))

	store, err := NewFileStore("_events")
	if err != nil {
		log.Fatal(err)
	}

	app := Application{Store: store}
	if err := app.Init(); err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/comments/", func(w http.ResponseWriter, req *http.Request) {
		commentId := req.URL.Path[len("/comments/"):]
		postId := app.types.posts.IdForComment(commentId)

		switch req.Method {
		case "GET":
			view, err := app.views.allPosts.approveCommentViewFor(postId, commentId)
			if err != nil {
				respondWithError(w, err)
			} else {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.Write(view.RenderHTML())
			}
		case "POST":
			cmd := &PostAuthenticateCommentCommand{
				CommentId: commentId,
			}

			if _, err := app.HandleCommand(cmd); err != nil {
				respondWithError(w, err)
			} else {
				post := app.views.allPosts.ById(postId)
				w.Header().Set("Location", post.Url.String())
				w.WriteHeader(http.StatusSeeOther)
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
				respondWithError(w, err)
			} else {
				post := app.views.allPosts.ById(cmd.PostId)
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.Write(renderTemplate("views/comment_received.html", map[string]interface{}{
					"Post":  post,
					"Email": cmd.Email,
				}))
			}
		default:
			http.Error(w, "Only POST is allowed.", http.StatusMethodNotAllowed)

		}
	})

	http.HandleFunc("/posts.html", func(w http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case "GET":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(app.views.allPosts.RenderHTML())
		default:
			http.Error(w, "Only GET is allowed.", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/posts/", func(w http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case "GET":
			fields := strings.Split(req.URL.Path, "/")
			postSlug := strings.Replace(fields[len(fields)-1], ".html", "", 1)
			view := app.views.allPosts.BySlug(postSlug)
			if view == nil {
				respondWithError(w, ErrNotFound)
			} else {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.Write(view.RenderHTML())
			}
		default:
			http.Error(w, "Only GET is allowed.", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/admin/posts/new", func(w http.ResponseWriter, req *http.Request) {
		if !authenticated(w, req) {
			return
		}
		switch req.Method {
		case "GET":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(renderTemplate("views/publish_post.html", nil))
		default:
			http.Error(w, "Only GET is allowed.", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/admin/posts/preview", func(w http.ResponseWriter, req *http.Request) {
		if !authenticated(w, req) {
			return
		}

		switch req.Method {
		case "POST":
			cmd := &PreviewPostCommand{
				PublishPostCommand: &PublishPostCommand{
					Title:   req.FormValue("title"),
					Content: req.FormValue("content"),
				},
			}
			if _, err := app.HandleCommand(cmd); err != nil {
				respondWithError(w, err)
			} else {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.Write(cmd.view.RenderHTML())
			}
		default:
			http.Error(w, "Only POST is allowed.", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/admin/posts", func(w http.ResponseWriter, req *http.Request) {
		if !authenticated(w, req) {
			return
		}

		switch req.Method {
		case "POST":
			cmd := &PublishPostCommand{
				Title:   req.FormValue("title"),
				Content: req.FormValue("content"),
			}
			if _, err := app.HandleCommand(cmd); err != nil {
				respondWithError(w, err)
			} else {
				http.Redirect(w, req, "/posts.html", http.StatusSeeOther)
			}
		default:
			http.Error(w, "Only POST is allowed.", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/posts", func(w http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case "GET":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(app.views.allPosts.RenderHTML())
		default:
			http.Error(w, "Only GET,POST is allowed.", http.StatusMethodNotAllowed)
		}
	})

	http.Handle("/index.html", http.RedirectHandler("/posts.html", http.StatusSeeOther))
	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		if app.tls.enabled && req.URL.Scheme == "http" {
			req.URL.Scheme = "https"
			http.Redirect(w, req, "/", http.StatusSwitchingProtocols)
			return
		}

		if req.URL.Path == "/" {
			http.Redirect(w, req, "/posts.html", http.StatusSeeOther)
		} else {
			assetServer.ServeHTTP(w, req)
		}
	})

	if app.tls.enabled {
		log.Fatal(http.ListenAndServeTLS(os.Getenv("BLOG_HOST"), app.tls.cert, app.tls.key, nil))
	} else {
		log.Fatal(http.ListenAndServe(os.Getenv("BLOG_HOST"), nil))
	}
}
