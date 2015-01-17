package main

import "strings"

type RewordPostCommand struct {
	PostId     string
	NewContent string
}

func (cmd *RewordPostCommand) Sanitize() {
	cmd.NewContent = strings.TrimSpace(cmd.NewContent)
}

type PublishPostCommand struct {
	Title   string
	Content string

	postId string
}

func (cmd *PublishPostCommand) Sanitize() {
	cmd.Title = strings.TrimSpace(cmd.Title)
	cmd.Content = strings.TrimSpace(cmd.Content)

	if cmd.postId == "" {
		cmd.postId = Id()
	}
}

type CommentOnPostCommand struct {
	PostId  string
	Content string
	Author  string
	Email   string
}

func (cmd *CommentOnPostCommand) Sanitize() {
	cmd.Content = strings.TrimSpace(cmd.Content)
	cmd.Author = strings.TrimSpace(cmd.Author)
	cmd.Email = strings.TrimSpace(cmd.Email)
}

type PostAuthenticateCommentCommand struct {
	PostId    string
	CommentId string
}

func (cmd *PostAuthenticateCommentCommand) Sanitize() {}
