package main

import "time"

type PostRewordedEvent struct {
	PostId          string
	RewordedContent string
	RewordedAt      time.Time
}

func (event *PostRewordedEvent) Tag() string         { return "posts.reworded" }
func (event *PostRewordedEvent) AggregateId() string { return event.PostId }

type PostPublishedEvent struct {
	PostId      string
	Title       string
	Content     string
	PublishedAt time.Time
}

func (event *PostPublishedEvent) Tag() string {
	return "post.published"
}

func (event *PostPublishedEvent) AggregateId() string {
	return event.PostId
}

type PostCommentedEvent struct {
	PostId      string
	AuthorName  string
	Content     string
	CommentedAt time.Time
}

func (event *PostCommentedEvent) Tag() string         { return "post.commented" }
func (event *PostCommentedEvent) AggregateId() string { return event.PostId }
