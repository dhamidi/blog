package main

import (
	"fmt"
	"sort"
	"time"

	"github.com/russross/blackfriday"
)

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
