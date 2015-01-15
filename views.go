package main

import (
	"encoding/json"
	"log"
)

type AllPostsPost struct {
	Id      string
	Title   string
	Content string
}

type AllPostsJSONView struct {
	Collection []*AllPostsPost
}

func (view *AllPostsJSONView) Apply(event Event) error {
	switch evt := event.(type) {
	case *PostPublishedEvent:
		view.addPost(evt)
	}

	return nil
}

func (view *AllPostsJSONView) addPost(evt *PostPublishedEvent) {
	view.Collection = append(view.Collection, &AllPostsPost{
		Id:      evt.PostId,
		Title:   evt.Title,
		Content: evt.Content,
	})
}

func (view *AllPostsJSONView) Render() []byte {
	data, err := json.MarshalIndent(view, "", "  ")
	if err != nil {
		log.Printf("AllPostsJSONView.Render: %s\n", err)
		return []byte{}
	}

	return data
}
