package main

import "strings"

type RewordPostCommand struct {
	PostId     string
	NewContent string
}

func (cmd *RewordPostCommand) Validate() ValidationError {
	verr := ValidationError{}

	cmd.NewContent = strings.TrimSpace(cmd.NewContent)
	if cmd.NewContent == "" {
		verr.Add("Content", ErrEmpty)
	}

	return verr
}

type PublishPostCommand struct {
	Title   string
	Content string

	postId string
}

func (cmd *PublishPostCommand) Validate() ValidationError {
	cmd.Title = strings.TrimSpace(cmd.Title)
	cmd.Content = strings.TrimSpace(cmd.Content)

	verr := ValidationError{}
	if cmd.Title == "" {
		verr.Add("Title", ErrEmpty)
	}
	if cmd.Content == "" {
		verr.Add("Content", ErrEmpty)
	}

	return verr
}

type CommentOnPostCommand struct {
	PostId  string
	Content string
	Author  string
}

func (cmd *CommentOnPostCommand) Validate() ValidationError {
	verr := ValidationError{}
	cmd.Content = strings.TrimSpace(cmd.Content)
	cmd.Author = strings.TrimSpace(cmd.Author)

	if cmd.Content == "" {
		verr.Add("Content", ErrEmpty)
	}
	if cmd.Author == "" {
		verr.Add("Author", ErrEmpty)
	}

	return verr
}
