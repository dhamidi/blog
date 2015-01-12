package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func (view *AllPostsView) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	data, err := json.MarshalIndent(view, "", "  ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	fmt.Fprintf(w, "%s", data)
}

func (view *PostDetailView) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "GET":
		view.showPost(w, req)
	default:
		http.Error(w, fmt.Sprintf("%s not supported.", req.Method), http.StatusMethodNotAllowed)
	}
}

func (view *PostDetailView) showPost(w http.ResponseWriter, req *http.Request) {
	postId := req.URL.Path[len("/posts/"):]
	post, err := view.postById(postId)
	if err == ErrNotFound {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	enc := json.NewEncoder(w)
	if err := enc.Encode(post); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (view *PostDetailView) postById(id string) (*PostDetailPost, error) {
	post, found := view.Posts[id]
	if !found {
		return nil, ErrNotFound
	}

	return post, nil
}

func (app *Application) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "POST":
		cmd := &PublishPostCommand{
			Title:   req.FormValue("title"),
			Content: req.FormValue("content"),
		}
		if _, err := app.HandleCommand(cmd); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		} else {
			w.WriteHeader(http.StatusCreated)
		}
	default:
		http.Error(w, fmt.Sprintf("%s not supported.", req.Method), http.StatusMethodNotAllowed)
	}
}
