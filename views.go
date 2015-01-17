package main

import (
	"encoding/json"
	"log"
	"sort"
	"time"
)

type AllPostsComment struct {
	Id      string
	Author  string
	Content string

	created time.Time
}

type AllPostsPost struct {
	Id       string
	Title    string
	Content  string
	Comments []*AllPostsComment

	allComments map[string]*AllPostsComment
}

func (post *AllPostsPost) Len() int { return len(post.Comments) }
func (post *AllPostsPost) Swap(i, j int) {
	post.Comments[i], post.Comments[j] = post.Comments[j], post.Comments[i]
}
func (post *AllPostsPost) Less(i, j int) bool {
	return post.Comments[i].created.Before(post.Comments[j].created)
}

func (post *AllPostsPost) addComment(comment *AllPostsComment) {
	post.allComments[comment.Id] = comment
}

func (post *AllPostsPost) authenticateComment(id string) {
	comment := post.allComments[id]
	if comment != nil {
		post.Comments = append(post.Comments, comment)
		sort.Sort(post)
	}
}

type AllPostsJSONView struct {
	Collection []*AllPostsPost

	allPosts map[string]*AllPostsPost
}

func (view *AllPostsJSONView) HandleEvent(event Event) error {
	switch evt := event.(type) {
	case *PostPublishedEvent:
		view.addPost(evt)
	case *PostCommentedEvent:
		view.addCommentToPost(evt)
	case *PostCommentAuthenticatedEvent:
		view.authenticateComment(evt)
	}

	return nil
}

func (view *AllPostsJSONView) addPost(evt *PostPublishedEvent) {
	if view.allPosts == nil {
		view.allPosts = map[string]*AllPostsPost{}
	}

	post := &AllPostsPost{
		Id:      evt.PostId,
		Title:   evt.Title,
		Content: evt.Content,

		allComments: map[string]*AllPostsComment{},
	}

	view.allPosts[evt.PostId] = post
	view.Collection = append(view.Collection, post)
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
		Id:      evt.CommentId,
		Author:  evt.AuthorName,
		Content: evt.Content,

		created: evt.CommentedAt,
	}

	post := view.allPosts[evt.PostId]
	post.addComment(comment)
}

func (view *AllPostsJSONView) authenticateComment(evt *PostCommentAuthenticatedEvent) {
	post := view.allPosts[evt.PostId]
	post.authenticateComment(evt.CommentId)
}
