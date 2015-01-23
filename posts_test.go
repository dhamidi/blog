package main_test

import (
	"testing"

	"github.com/dhamidi/blog"
)

func TestPost_Publish_RequiresFields(t *testing.T) {
	posts := &main.Posts{}
	post := posts.New()
	_, err := post.HandleCommand(&main.PublishPostCommand{
		Title:   "",
		Content: "",
	})

	if err == nil {
		t.Fatal("Expected an error.")
	}

	verr := err.(main.ValidationError)
	if verr.Get("Title") != main.ErrEmpty {
		t.Fatalf("Title not %s", main.ErrEmpty)
	}

	if verr.Get("Content") != main.ErrEmpty {
		t.Fatalf("Content not %s", main.ErrEmpty)
	}
}

func TestPost_Publish_RequiresUniqueTitle(t *testing.T) {
	posts := &main.Posts{}
	posts.HandleEvent(&main.PostPublishedEvent{
		Title: "post-title",
	})

	post := posts.New()
	_, err := post.HandleCommand(&main.PublishPostCommand{
		Title:   "post-title",
		Content: "post-content",
	})

	if err == nil {
		t.Fatal("Expected an error.")
	}

	verr := err.(main.ValidationError)
	if verr.Get("Title") != main.ErrNotUnique {
		t.Fatalf("Duplicate title allowed.")
	}
}

func TestPost_Comment(t *testing.T) {
	published := &main.PostPublishedEvent{
		PostId:  main.Id(),
		Title:   "post-title",
		Content: "post-content",
	}
	posts := &main.Posts{}
	posts.HandleEvent(published)
	post := posts.New()
	post.HandleEvent(published)
	events, err := post.HandleCommand(&main.CommentOnPostCommand{
		PostId:  published.PostId,
		Author:  "author",
		Email:   "author@example.com",
		Content: "comment-content",
	})

	if err != nil {
		t.Fatal(err)
	}

	commented := events.Items()[0].(*main.PostCommentedEvent)
	if commented.PostId != published.PostId {
		t.Fatalf("Expected same post id on event. Got: %s", commented.PostId)
	}
}

func TestPost_Comment_RequiresExistingPost(t *testing.T) {
	posts := &main.Posts{}
	post := posts.New()

	postId := main.Id()
	_, err := post.HandleCommand(&main.CommentOnPostCommand{
		PostId:  postId,
		Author:  "author",
		Email:   "author@example.com",
		Content: "comment-content",
	})

	if err == nil {
		t.Fatal("Expected an error.")
	}

	verr := err.(main.ValidationError)
	if perr := verr.Get("Post"); perr != main.ErrNotFound {
		t.Fatalf("Expected post to be %s, got %s", main.ErrNotFound, perr)
	}
}
