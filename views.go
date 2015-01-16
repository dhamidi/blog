package main

import (
	"encoding/json"
	"log"
)

type AllPostsComment struct {
	Author  string
	Content string
}

type AllPostsPost struct {
	Id       string
	Title    string
	Content  string
	Comments []*AllPostsComment
}

type AllPostsJSONView struct {
	Collection []*AllPostsPost
}

func (view *AllPostsJSONView) HandleEvent(event Event) error {
	switch evt := event.(type) {
	case *PostPublishedEvent:
		view.addPost(evt)
	case *PostCommentedEvent:
		view.addCommentToPost(evt)
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

func (view *AllPostsJSONView) addCommentToPost(evt *PostCommentedEvent) {
	comment := &AllPostsComment{
		Author:  evt.AuthorName,
		Content: evt.Content,
	}

	for _, post := range view.Collection {
		if post.Id == evt.PostId {
			post.Comments = append(post.Comments, comment)
			return
		}
	}
}
