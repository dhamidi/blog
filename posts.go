package main

import "time"

type Posts struct {
	titles map[string]bool
}

func (posts *Posts) New() *Post {
	return &Post{posts: posts}
}

func (posts *Posts) Apply(event Event) error {
	if posts.titles == nil {
		posts.titles = map[string]bool{}
	}

	switch evt := event.(type) {
	case *PostPublishedEvent:
		posts.titles[evt.Title] = true
	}

	return nil
}

func (posts *Posts) UniqueTitle(title string) bool {
	return posts.titles[title] != true
}

type Post struct {
	posts   *Posts
	id      string
	content string
}

func (post *Post) Apply(event Event) error {
	switch evt := event.(type) {
	case *PostPublishedEvent:
		post.id = evt.PostId
		post.content = evt.Content
	case *PostRewordedEvent:
		post.content = evt.RewordedContent
	}

	return nil
}

func (post *Post) When(command Command) (*Events, error) {
	switch cmd := command.(type) {
	case *PublishPostCommand:
		return post.publish(cmd)
	case *RewordPostCommand:
		return post.reword(cmd)
	}

	return NoEvents, nil
}

func (post *Post) publish(cmd *PublishPostCommand) (*Events, error) {
	verr := cmd.Validate()

	if !post.uniqueTitle(cmd.Title) {
		verr.Add("Title", ErrNotUnique)
	}

	if err := verr.Return(); err != nil {
		return NoEvents, err
	} else {
		return ListOfEvents(&PostPublishedEvent{
			PostId:      Id(),
			Title:       cmd.Title,
			Content:     cmd.Content,
			PublishedAt: time.Now(),
		}), nil
	}
}

func (post *Post) reword(cmd *RewordPostCommand) (*Events, error) {
	verr := cmd.Validate()

	if cmd.NewContent == post.content {
		return NoEvents, verr.Return()
	}

	return ListOfEvents(&PostRewordedEvent{
		PostId:          cmd.PostId,
		RewordedContent: cmd.NewContent,
		RewordedAt:      time.Now(),
	}), nil
}

func (post *Post) uniqueTitle(title string) bool {
	return post.posts.UniqueTitle(title)
}
