package main

import "time"

type PostRewordedEvent struct {
	PostId          string
	Reason          string
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
	CommentId   string
	AuthorName  string
	AuthorEmail string
	Content     string
	CommentedAt time.Time
}

func (event *PostCommentedEvent) Tag() string         { return "post.commented" }
func (event *PostCommentedEvent) AggregateId() string { return event.PostId }

type PostCommentAuthenticatedEvent struct {
	CommentId       string
	PostId          string
	AuthenticatedAt time.Time
}

func (event *PostCommentAuthenticatedEvent) Tag() string         { return "post.comment_authenticated" }
func (event *PostCommentAuthenticatedEvent) AggregateId() string { return event.PostId }
