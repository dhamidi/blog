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
