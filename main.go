package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/dhamidi/blog/eventstore"

	// expvar is imported for registering its HTTP handler.
	_ "expvar"
)

func respondWithError(w http.ResponseWriter, err error) {
	switch err {
	case ErrNotFound:
		http.Error(w, err.Error(), http.StatusNotFound)
	default:
		if strings.HasPrefix(err.Error(), "ValidationError") {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			w.Write(renderTemplate("views/validation_error.html", err))
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

	store, err := eventstore.NewOnDisk("_events")
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

	http.HandleFunc("/admin/posts/", func(w http.ResponseWriter, req *http.Request) {
		if !authenticated(w, req) {
			return
		}

		action := ""
		fields := strings.Split(req.URL.Path[len("/admin/posts/"):], "/")
		postId := fields[0]
		if len(fields) > 1 {
			action = fields[1]
		}
		post := app.views.allPosts.ById(postId)

		if post == nil {
			respondWithError(w, ErrNotFound)
		}

		switch req.Method {
		case "GET":
			switch action {
			case "reword":
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.Write(renderTemplate("views/reword_post.html", post))
			default:
				respondWithError(w, ErrNotFound)
			}
		case "POST":
			cmd := &RewordPostCommand{
				PostId:     postId,
				Reason:     req.FormValue("reason"),
				NewContent: req.FormValue("content"),
			}

			if _, err := app.HandleCommand(cmd); err != nil {
				respondWithError(w, err)
			} else {
				http.Redirect(w, req, "/admin", http.StatusSeeOther)
			}
		default:
			http.Error(w, "Only GET is allowed.", http.StatusMethodNotAllowed)
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

	http.HandleFunc("/admin/", func(w http.ResponseWriter, req *http.Request) {
		if !authenticated(w, req) {
			return
		}

		switch req.Method {
		case "GET":
			w.Header().Set("Content-Type", "text/html; charsetf=utf-8")
			w.Write(renderTemplate("views/admin.html", app.views.allPosts))
		default:
			http.Error(w, "Only GET is allowed.", http.StatusMethodNotAllowed)
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

	http.HandleFunc("/sitemap.xml", func(w http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case "GET":
			w.Header().Set("Content-Type", "application/xml; charset=utf-8")
			app.views.sitemap.RenderXML(w)
		default:
			http.Error(w, "Only GET is allowed.", http.StatusMethodNotAllowed)
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
		go http.ListenAndServe(
			os.Getenv("BLOG_HOST"),
			http.RedirectHandler(fmt.Sprintf("https://%s/", os.Getenv("BLOG_TLS_HOST")), http.StatusMovedPermanently),
		)
		log.Fatal(http.ListenAndServeTLS(os.Getenv("BLOG_TLS_HOST"), app.tls.cert, app.tls.key, nil))
	} else {
		log.Fatal(http.ListenAndServe(os.Getenv("BLOG_HOST"), nil))
	}
}
