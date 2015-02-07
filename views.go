package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday"
)

type ChangeItem struct {
	At          time.Time
	ChangedAt   string
	Description string
}

type RecentChanges []*ChangeItem

func (c RecentChanges) Len() int           { return len(c) }
func (c RecentChanges) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }
func (c RecentChanges) Less(i, j int) bool { return c[i].At.After(c[j].At) }

type AllPostsComment struct {
	Id          string
	Author      string
	Content     string
	ContentHTML template.HTML
	Created     string

	createdAt time.Time
}

type AllPostsPost struct {
	Id        string
	Title     string
	Published string

	Content     string
	ContentHTML template.HTML

	Excerpt     string
	ExcerptHTML template.HTML

	Url  *url.URL
	Slug string

	Preview bool

	Comments []*AllPostsComment

	Changes []*ChangeItem

	allComments map[string]*AllPostsComment
	publishedAt time.Time
}

func (post *AllPostsPost) Len() int { return len(post.Comments) }
func (post *AllPostsPost) Swap(i, j int) {
	post.Comments[i], post.Comments[j] = post.Comments[j], post.Comments[i]
}
func (post *AllPostsPost) Less(i, j int) bool {
	return post.Comments[i].createdAt.Before(post.Comments[j].createdAt)
}

func (post *AllPostsPost) addComment(comment *AllPostsComment) {
	post.allComments[comment.Id] = comment
}

func (post *AllPostsPost) authenticateComment(id string) {
	comment := post.allComments[id]
	if comment != nil {
		post.Comments = append(post.Comments, comment)
		sort.Sort(post)
	}
}

func (post *AllPostsPost) createExcerpt() {
	excerptEnd := strings.Index(post.Content, "\n\n")
	if excerptEnd != -1 {
		post.Excerpt = post.Content[:excerptEnd]
	}

	excerptEndHTML := strings.Index(string(post.ContentHTML), "</p>")
	if excerptEndHTML != -1 {
		post.ExcerptHTML = post.ContentHTML[:excerptEndHTML]
	}
}

func (post *AllPostsPost) setContent(content string) *AllPostsPost {
	post.Content = content
	post.ContentHTML = textToHTML(content, false)

	post.createExcerpt()

	return post
}

func (post *AllPostsPost) addToHistory(occurred time.Time, description string) *AllPostsPost {
	post.Changes = append(post.Changes, &ChangeItem{
		At:          occurred,
		ChangedAt:   occurred.Format("02 Jan 2006, 15:04:05 MST"),
		Description: description,
	})

	sort.Sort(RecentChanges(post.Changes))

	return post
}

func (post *AllPostsPost) RenderHTML() []byte {
	if post.Preview {
		return renderTemplate("views/post_preview.html", post)
	} else {
		return renderTemplate("views/post.html", post)
	}
}

type AllPostsView struct {
	Collection []*AllPostsPost

	allPosts       map[string]*AllPostsPost
	allPostsBySlug map[string]*AllPostsPost
}

func (view *AllPostsView) Len() int { return len(view.Collection) }
func (view *AllPostsView) Swap(i, j int) {
	view.Collection[i], view.Collection[j] = view.Collection[j], view.Collection[i]
}
func (view *AllPostsView) Less(i, j int) bool {
	return view.Collection[j].publishedAt.Before(view.Collection[i].publishedAt)
}

func (view *AllPostsView) HandleEvent(event Event) error {
	switch evt := event.(type) {
	case *PostPublishedEvent:
		view.addPost(evt)
	case *PostRewordedEvent:
		view.rewordPost(evt)
	case *PostCommentedEvent:
		view.addCommentToPost(evt)
	case *PostCommentAuthenticatedEvent:
		view.authenticateComment(evt)
	}

	return nil
}

func (view *AllPostsView) addPost(evt *PostPublishedEvent) {
	if view.allPosts == nil {
		view.allPosts = map[string]*AllPostsPost{}
	}
	if view.allPostsBySlug == nil {
		view.allPostsBySlug = map[string]*AllPostsPost{}
	}

	slug := slugify(evt.Title)
	post := &AllPostsPost{
		Id:          evt.PostId,
		Title:       evt.Title,
		Content:     evt.Content,
		ContentHTML: textToHTML(evt.Content, false),
		Comments:    []*AllPostsComment{},
		Published:   evt.PublishedAt.Format("02 Jan 2006"),
		Slug:        slug,
		Url:         &url.URL{Path: "/posts/" + slug + ".html"},
		Preview:     false,

		allComments: map[string]*AllPostsComment{},
		publishedAt: evt.PublishedAt,
	}
	post.createExcerpt()

	view.allPosts[evt.PostId] = post
	view.allPostsBySlug[slug] = post
	view.Collection = append(view.Collection, post)
	sort.Sort(view)
}

func (view *AllPostsView) rewordPost(event *PostRewordedEvent) {
	post := view.ById(event.PostId)
	if post == nil {
		return
	}

	post.setContent(event.RewordedContent)

	if event.Reason != "" {
		post.addToHistory(event.RewordedAt, event.Reason)
	}
}

func (view *AllPostsView) ById(id string) *AllPostsPost {
	return view.allPosts[id]
}

func (view *AllPostsView) BySlug(slug string) *AllPostsPost {
	return view.allPostsBySlug[slug]
}

func (view *AllPostsView) addCommentToPost(evt *PostCommentedEvent) {
	comment := &AllPostsComment{
		Id:          evt.CommentId,
		Author:      evt.AuthorName,
		Content:     evt.Content,
		Created:     evt.CommentedAt.Format("02 Jan 2006 15:04"),
		ContentHTML: textToHTML(evt.Content, true),

		createdAt: evt.CommentedAt,
	}

	post := view.allPosts[evt.PostId]
	post.addComment(comment)
}

func (view *AllPostsView) authenticateComment(evt *PostCommentAuthenticatedEvent) {
	post := view.allPosts[evt.PostId]
	post.authenticateComment(evt.CommentId)
}

func (view *AllPostsView) approveCommentViewFor(postId, commentId string) (*ApproveCommentView, error) {
	post := view.allPosts[postId]
	if post == nil {
		return nil, ErrNotFound
	}

	comment := post.allComments[commentId]
	if comment == nil {
		return nil, ErrNotFound
	}

	return &ApproveCommentView{
		Post:    post,
		Comment: comment,
	}, nil
}

func (view *AllPostsView) RenderHTML() []byte {
	return renderTemplate("views/all_posts.html", view)
}

type ApproveCommentView struct {
	Post    *AllPostsPost
	Comment *AllPostsComment
}

func (view *ApproveCommentView) RenderHTML() []byte {
	return renderTemplate("views/approve_comment.html", view)
}

type SitemapURL struct {
	XMLName    xml.Name `xml:"url"`
	Loc        string   `xml:"loc"`
	Lastmod    string   `xml:"lastmod,omitempty"`
	Changefreq string   `xml:"changefreq,omitempty"`
	Priority   string   `xml:"priority,omitempty"`
}

type Sitemap struct {
	allPosts *AllPostsView
	baseUrl  *url.URL
	Urls     []*SitemapURL
}

func NewSitemap(allPosts *AllPostsView) *Sitemap {
	host := os.Getenv("BLOG_TLS_HOST")
	if host == "" {
		host = "http://" + os.Getenv("BLOG_HOST")
	} else {
		host = "https://" + host
	}

	if host == "" {
		log.Fatal("BLOG_TLS_HOST or BLOG_HOST need to be set.")
	}

	baseUrl, err := url.Parse(host)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Sitemap: host=%q, baseUrl=%#v\n", host, baseUrl)
	return &Sitemap{
		allPosts: allPosts,
		baseUrl:  baseUrl,
		Urls:     []*SitemapURL{},
	}
}

func (sitemap *Sitemap) HandleEvent(event Event) error {
	switch evt := event.(type) {
	case *PostPublishedEvent:
		post := sitemap.allPosts.ById(evt.PostId)
		url := &SitemapURL{
			Loc: sitemap.baseUrl.ResolveReference(post.Url).String(),
		}
		sitemap.Urls = append(sitemap.Urls, url)
	}

	return nil
}

func (sitemap *Sitemap) RenderXML(w io.Writer) error {
	fmt.Fprintf(w, "%s\n%s\n",
		xml.Header,
		`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`,
	)
	enc := xml.NewEncoder(w)
	for _, url := range sitemap.Urls {
		if err := enc.Encode(url); err != nil {
			log.Printf("sitemap: %s\n", err)
		}
	}
	fmt.Fprintf(w, "</urlset>\n")
	return nil
}

var blackfridayExtensions = (blackfriday.EXTENSION_NO_INTRA_EMPHASIS |
	blackfriday.EXTENSION_TABLES |
	blackfriday.EXTENSION_FENCED_CODE |
	blackfriday.EXTENSION_AUTOLINK |
	blackfriday.EXTENSION_SPACE_HEADERS |
	blackfriday.EXTENSION_FOOTNOTES |
	blackfriday.EXTENSION_HEADER_IDS |
	blackfriday.EXTENSION_AUTO_HEADER_IDS)

type userContentRenderer struct {
	blackfriday.Renderer
}

func (renderer *userContentRenderer) Image(out *bytes.Buffer, link []byte, title []byte, alt []byte) {
	renderer.Link(out, link, title, alt)
}

var (
	blackfridayAuthor = blackfriday.HtmlRenderer(
		blackfriday.HTML_USE_XHTML|
			blackfriday.HTML_USE_SMARTYPANTS|
			blackfriday.HTML_SMARTYPANTS_FRACTIONS|
			blackfriday.HTML_SMARTYPANTS_LATEX_DASHES,
		"",
		"",
	)
	blackfridayUgc = &userContentRenderer{
		Renderer: blackfridayAuthor,
	}
)

func textToHTML(text string, userGenerated bool) template.HTML {
	renderer := blackfridayAuthor
	if userGenerated {
		renderer = blackfridayUgc
	}

	data := blackfriday.Markdown([]byte(text), renderer, blackfridayExtensions)

	if userGenerated {
		data = bluemonday.UGCPolicy().SanitizeBytes(data)
	}

	return template.HTML(data)
}

var slugReplacer = strings.NewReplacer(
	" ", "-",
	"\n", ";",
)

func slugify(str string) string {
	return slugReplacer.Replace(strings.TrimSpace(strings.ToLower(str)))
}

func renderTemplate(name string, data interface{}) []byte {
	tmpl, err := template.ParseFiles("views/layout.html", name)
	if err != nil {
		return []byte(err.Error())
	}

	out := bytes.NewBufferString("")

	if err := tmpl.ExecuteTemplate(out, "layout.html", data); err != nil {
		return []byte(err.Error())
	} else {
		return out.Bytes()
	}
}
