package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/nu7hatch/gouuid"
)

var (
	ErrNotUnique = errors.New("not unique")
	ErrEmpty     = errors.New("empty")
)

func Id() string {
	id, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}
	return id.String()
}

type ValidationError map[string][]error

func (verr ValidationError) Error() string {
	out := bytes.NewBufferString("")

	fmt.Fprintf(out, "ValidationError:\n")
	for field, errors := range verr {
		fmt.Fprintf(out, "  %s: %v\n", field, errors)
	}

	return out.String()
}

func (verr ValidationError) Add(key string, err error) ValidationError {
	verr[key] = append(verr[key], err)
	return verr
}

func (verr ValidationError) Len() int {
	return len(verr)
}

func (verr ValidationError) Return() error {
	if verr.Len() == 0 {
		return nil
	} else {
		return verr
	}
}

type Aggregate interface {
	When(Command) (*Events, error)
	Apply(Event) error
}

type Event interface {
	Tag() string
	AggregateId() string
}
type EventList struct {
	items []Event
}

type Events EventList

var NoEvents *Events

func ListOfEvents(events ...Event) *Events {
	return &Events{items: events}
}

func (e *Events) Len() int { return len(e.items) }
func (e *Events) Append(events ...Event) *Events {
	e.items = append(e.items, events...)
	return e
}
func (e *Events) Items() []Event {
	return e.items
}

func (e *Events) ApplyTo(agg Aggregate) {
	for _, event := range e.items {
		agg.Apply(event)
	}
}

type Command interface {
	Validate() error
}

type PostAggregate struct {
	titles map[string]bool
}

type PublishPostCommand struct {
	Title   string
	Content string
}

func (cmd *PublishPostCommand) Validate() error {
	cmd.Title = strings.TrimSpace(cmd.Title)
	cmd.Content = strings.TrimSpace(cmd.Content)

	verr := ValidationError{}
	if cmd.Title == "" {
		verr.Add("Title", ErrEmpty)
	}
	if cmd.Content == "" {
		verr.Add("Content", ErrEmpty)
	}

	return verr.Return()
}

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

func (post *PostAggregate) When(command Command) (*Events, error) {
	switch cmd := command.(type) {
	case *PublishPostCommand:
		return post.publish(cmd)
	}

	return NoEvents, nil
}

func (post *PostAggregate) Apply(event Event) error {
	if post.titles == nil {
		post.titles = map[string]bool{}
	}

	switch evt := event.(type) {
	case *PostPublishedEvent:
		post.titles[evt.Title] = true
	}

	return nil
}

func (post *PostAggregate) publish(cmd *PublishPostCommand) (*Events, error) {
	if post.uniqueTitle(cmd.Title) {
		return ListOfEvents(&PostPublishedEvent{
			PostId:      Id(),
			Title:       cmd.Title,
			Content:     cmd.Content,
			PublishedAt: time.Now(),
		}), nil
	} else {
		return NoEvents, ValidationError{"Title": []error{ErrNotUnique}}
	}
}

func (post *PostAggregate) uniqueTitle(title string) bool {
	return !post.titles[title]
}

type FileStore struct {
	dir     string
	typeMap map[string]reflect.Type
	lock    *sync.RWMutex
}

type eventOnFile struct {
	StoredAt *time.Time
	Type     string
	Event    json.RawMessage
}

func NewFileStore(dir string) (*FileStore, error) {
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(dir, 0755)
		}

		if err != nil {
			return nil, err
		}
	}

	return &FileStore{
		dir:     dir,
		typeMap: map[string]reflect.Type{},
		lock:    &sync.RWMutex{},
	}, nil
}

func (fs *FileStore) RegisterType(event Event) {
	fs.typeMap[event.Tag()] = reflect.TypeOf(event)
}

func (fs *FileStore) LoadAll() (*Events, error) {
	return fs.load(fs.filenamesForStream("all"))
}

func (fs *FileStore) LoadStream(id string) (*Events, error) {
	return fs.load(fs.filenamesForStream(id))
}

func (fs *FileStore) filenamesForStream(id string) []string {
	dirname := filepath.Join(fs.dir, id)
	dir, err := os.Open(dirname)
	if err != nil {
		log.Printf("FileStore: %s\n", err)
		return []string{}
	}

	fnames := []string{}
	if names, err := dir.Readdirnames(0); err != nil {
		log.Printf("FileStore: %s\n", err)
		return []string{}
	} else {
		for _, name := range names {
			fnames = append(fnames, filepath.Join(dirname, name))
		}
	}

	sort.Strings(fnames)

	return fnames
}

func (fs *FileStore) load(filenames []string) (*Events, error) {
	events := []Event{}
	for _, fname := range filenames {
		if event, err := fs.loadEvent(fname); err != nil {
			return NoEvents, fmt.Errorf("FileStore: %s\n", err)
		} else {
			events = append(events, event)
		}
	}

	return &Events{items: events}, nil
}

func (fs *FileStore) loadEvent(fname string) (Event, error) {
	msg := eventOnFile{}
	file, err := os.Open(fname)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	dec := json.NewDecoder(file)
	if err := dec.Decode(&msg); err != nil {
		return nil, err
	}

	event := fs.eventForType(msg.Type)
	err = json.Unmarshal([]byte(msg.Event), event)
	if err != nil {
		return nil, err
	} else {
		return event, nil
	}
}

func (fs *FileStore) eventForType(typename string) Event {
	typ, ok := fs.typeMap[typename]
	if !ok {
		panic(fmt.Errorf("FileStore: type %q not registered.", typename))
	}

	return reflect.New(typ.Elem()).Interface().(Event)
}

func (fs *FileStore) Store(event Event) error {
	now := time.Now().UTC()

	eventData, err := json.Marshal(event)
	if err != nil {
		return err
	}

	eventMsg := json.RawMessage(eventData)
	msg := &eventOnFile{StoredAt: &now, Type: event.Tag(), Event: eventMsg}

	data, err := json.MarshalIndent(msg, "", "  ")
	if err != nil {
		return err
	}

	fs.lock.Lock()
	defer fs.lock.Unlock()

	if err := fs.storeForAll(now, data); err != nil {
		return err
	}

	return fs.storeForAggregate(now, event.AggregateId(), data)
}

func (fs *FileStore) storeForAll(now time.Time, data []byte) error {
	return fs.storeForAggregate(now, "all", data)
}

func (fs *FileStore) storeForAggregate(now time.Time, id string, data []byte) error {
	nowStr := fmt.Sprintf("%d", now.UnixNano())
	dirname := filepath.Join(fs.dir, id)
	fname := filepath.Join(dirname, nowStr)

	os.MkdirAll(dirname, 0755)

	out, err := os.OpenFile(fname, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("FileStore: %s\n", fname)
	}
	defer out.Close()

	if _, err := io.Copy(out, bytes.NewReader(data)); err != nil {
		return err
	} else {
		return nil
	}
}

func main() {
	store, err := NewFileStore("_events")
	if err != nil {
		log.Fatal(err)
	}

	store.RegisterType(&PostPublishedEvent{})

	events, err := store.LoadAll()
	if err != nil {
		log.Fatal(err)
	}

	for _, event := range events.Items() {
		fmt.Printf("%#v\n", event)
	}
}
