package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
)

type EventsInFileSystem struct {
	root  string
	byId  map[string]Events
	all   Events
	bySeq string
	enc   *gob.Encoder
	log   *os.File
}

func NewEventsInFileSystem(dir string) (*EventsInFileSystem, error) {
	result := &EventsInFileSystem{
		root:  dir,
		byId:  map[string]Events{},
		bySeq: filepath.Join(dir, "seq"),
		all:   Events{},
	}

	if err := result.Init(); err != nil {
		return nil, fmt.Errorf("EventsInFileSystem: %s", err.Error())
	}

	return result, nil
}

func (store *EventsInFileSystem) Init() error {
	if info, err := os.Stat(store.root); err != nil {
		return err
	} else if !info.IsDir() {
		return fmt.Errorf("%q needs to be a directory.", store.root)
	}

	if file, err := os.OpenFile(
		store.bySeq,
		os.O_CREATE|os.O_APPEND|os.O_WRONLY|os.O_SYNC,
		0644,
	); err != nil {
		return err
	} else {
		store.log = file
		store.enc = gob.NewEncoder(store.log)
	}

	return nil
}

func (store *EventsInFileSystem) AllEvents() (Events, error) {
	return store.all, nil
}

func (store *EventsInFileSystem) EventsForStream(streamId string) (Events, error) {
	return store.byId[streamId], nil
}

func (store *EventsInFileSystem) Register(value interface{}) {
	gob.Register(value)
}

func (store *EventsInFileSystem) LoadHistory() error {
	history, err := ioutil.ReadFile(store.bySeq)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	dec := gob.NewDecoder(bytes.NewReader(history))

	for {
		var result Event

		err = dec.Decode(&result)
		if err == io.EOF {
			return nil
		}

		if err != nil {
			return err
		}

		store.saveEventForStream(result)
		store.all = append(store.all, result)
	}
}

func (store *EventsInFileSystem) HandleEvent(event Event) error {
	streamId := event.EventStreamId()
	if streamId != "" {
		if err := store.saveEventForStream(event); err != nil {
			return err
		}
	}

	if err := store.appendToLog(event); err != nil {
		store.removeEventForStream(event)
		return err
	}

	return nil
}

func (store *EventsInFileSystem) appendToLog(event Event) error {
	if err := store.enc.Encode(&event); err != nil {
		return err
	}

	store.all = append(store.all, event)

	if err := store.log.Sync(); err != nil {
		return err
	}

	return nil
}

func (store *EventsInFileSystem) saveEventForStream(event Event) error {
	streamId := event.EventStreamId()
	store.byId[streamId] = append(store.byId[streamId], event)
	return nil
}

func (store *EventsInFileSystem) removeEventForStream(event Event) error {
	streamId := event.EventStreamId()
	for index, storedEvent := range store.byId[streamId] {
		if reflect.DeepEqual(storedEvent, event) {
			store.byId[streamId] = append(
				store.byId[streamId][0:index],
				store.byId[streamId][index+1:]...,
			)
			return nil
		}
	}

	return nil
}
