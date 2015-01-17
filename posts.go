package main

import "time"

type Posts struct {
	titles map[string]bool

	commentIds map[string]string
}

func (posts *Posts) New() Aggregate {
	return &Post{
		posts:    posts,
		comments: map[string]*PostComment{},
	}
}

func (posts *Posts) HandleEvent(event Event) error {
	if posts.titles == nil {
		posts.titles = map[string]bool{}
	}
	if posts.commentIds == nil {
		posts.commentIds = map[string]string{}
	}

	switch evt := event.(type) {
	case *PostPublishedEvent:
		posts.titles[evt.Title] = true
	case *PostCommentedEvent:
		posts.commentIds[evt.CommentId] = evt.PostId
	case *PostCommentAuthenticatedEvent:
		delete(posts.commentIds, evt.CommentId)
	}

	return nil
}

func (posts *Posts) IdForComment(commentId string) string {
	return posts.commentIds[commentId]
}

func (posts *Posts) UniqueTitle(title string) bool {
	return posts.titles[title] != true
}

type Post struct {
	posts    *Posts
	id       string
	content  string
	comments map[string]*PostComment
}

type PostComment struct {
	id            string
	authenticated bool
}

func (post *Post) HandleEvent(event Event) error {
	switch evt := event.(type) {
	case *PostPublishedEvent:
		post.id = evt.PostId
		post.content = evt.Content
	case *PostRewordedEvent:
		post.content = evt.RewordedContent
	case *PostCommentedEvent:
		comment := &PostComment{
			id:            evt.CommentId,
			authenticated: false,
		}
		if evt.CommentId == "" {
			comment.id = Id()
			evt.CommentId = comment.id
			comment.authenticated = true
		}
		post.comments[evt.CommentId] = comment
	case *PostCommentAuthenticatedEvent:
		post.comments[evt.CommentId].authenticated = true
	}

	return nil
}

func (post *Post) HandleCommand(command Command) (*Events, error) {
	switch cmd := command.(type) {
	case *PublishPostCommand:
		return post.publish(cmd)
	case *RewordPostCommand:
		return post.reword(cmd)
	case *CommentOnPostCommand:
		return post.comment(cmd)
	case *PostAuthenticateCommentCommand:
		return post.authenticateComment(cmd)
	}

	return NoEvents, nil
}

func (post *Post) publish(cmd *PublishPostCommand) (*Events, error) {
	verr := ValidationError{}
	if cmd.Title == "" {
		verr.Add("Title", ErrEmpty)
	}
	if cmd.Content == "" {
		verr.Add("Content", ErrEmpty)
	}

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
	verr := ValidationError{}
	if cmd.NewContent == "" {
		verr.Add("Content", ErrEmpty)
	}

	if cmd.NewContent == post.content {
		return NoEvents, verr.Return()
	}

	return ListOfEvents(&PostRewordedEvent{
		PostId:          cmd.PostId,
		RewordedContent: cmd.NewContent,
		RewordedAt:      time.Now(),
	}), nil
}

func (post *Post) authenticateComment(cmd *PostAuthenticateCommentCommand) (*Events, error) {
	verr := ValidationError{}
	comment := post.comments[cmd.CommentId]
	if comment == nil {
		verr.Add("Comment", ErrNotFound)
	} else if comment.authenticated {
		verr.Add("Comment", ErrAlreadyAuthenticated)
	}

	return ListOfEvents(&PostCommentAuthenticatedEvent{
		PostId:          post.id,
		CommentId:       cmd.CommentId,
		AuthenticatedAt: time.Now(),
	}), verr.Return()
}

func (post *Post) comment(cmd *CommentOnPostCommand) (*Events, error) {
	verr := ValidationError{}
	if cmd.Content == "" {
		verr.Add("Content", ErrEmpty)
	}
	if cmd.Author == "" {
		verr.Add("Author", ErrEmpty)
	}
	if cmd.Email == "" {
		verr.Add("Email", ErrEmpty)
	}

	return ListOfEvents(&PostCommentedEvent{
		PostId:      post.id,
		CommentId:   Id(),
		Content:     cmd.Content,
		AuthorName:  cmd.Author,
		AuthorEmail: cmd.Email,
		CommentedAt: time.Now(),
	}), verr.Return()
}

func (post *Post) uniqueTitle(title string) bool {
	return post.posts.UniqueTitle(title)
}
