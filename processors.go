package main

import (
	"fmt"
	"net/url"
)

type PostCommentProcessor struct {
	mailer Mailer
	posts  *AllPostsJSONView
}

func (proc *PostCommentProcessor) HandleEvent(event Event) error {
	switch evt := event.(type) {
	case *PostCommentedEvent:
		return proc.authenticateComment(evt)
	}

	return nil
}

func (proc *PostCommentProcessor) authenticateComment(evt *PostCommentedEvent) error {
	post := proc.posts.ById(evt.PostId)
	link := &url.URL{
		Scheme: "http",
		Host:   "localhost:8000",
		Path:   "/comments/" + evt.CommentId,
	}

	body := fmt.Sprintf(`Subject: Authenticate your comment

Hello %s,

Please authenticate your comment on

    %q

By clicking this link:

    %s`, evt.AuthorName, post.Title, link)

	return proc.mailer.SendMessage(&MailMessage{
		To:   []string{evt.AuthorEmail},
		Body: []byte(body),
	})
}
