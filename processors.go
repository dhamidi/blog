package main

import (
	"fmt"
	"net/url"
	"os"
)

type PostCommentProcessor struct {
	mailer Mailer
	posts  *AllPostsView
	useTls bool
}

func (proc *PostCommentProcessor) HandleEvent(event Event) error {
	switch evt := event.(type) {
	case *PostCommentedEvent:
		return proc.authenticateComment(evt)
	}

	return nil
}

func (proc *PostCommentProcessor) authenticateComment(evt *PostCommentedEvent) error {
	scheme := "https"
	host := os.Getenv("BLOG_PROXY")
	if host == "" {
		if !proc.useTls {
			scheme = "http"
			host = os.Getenv("BLOG_HOST")
		} else {
			host = os.Getenv("BLOG_TLS_HOST")
		}
	}

	post := proc.posts.ById(evt.PostId)
	link := &url.URL{
		Scheme: scheme,
		Host:   host,
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
