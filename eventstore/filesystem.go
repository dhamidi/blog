package eventstore

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"sync"
	"time"
)

type fileStore struct {
	dir     string
	typeMap map[string]reflect.Type
	lock    *sync.RWMutex
}

type eventOnFile struct {
	StoredAt *time.Time
	Type     string
	Event    json.RawMessage
}

func NewOnDisk(dir string) (Store, error) {
	streamDir := filepath.Join(dir, "all")

	if _, err := os.Stat(streamDir); err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(streamDir, 0755)
		}

		if err != nil {
			return nil, &StorageError{
				Op:          "OnDisk",
				Stream:      "all",
				Err:         ErrInternal,
				InternalErr: err,
			}
		}
	}

	return &fileStore{
		dir:     dir,
		typeMap: map[string]reflect.Type{},
		lock:    &sync.RWMutex{},
	}, nil
}

func (fs *fileStore) RegisterType(event Event) {
	fs.typeMap[event.Tag()] = reflect.TypeOf(event)
}

func (fs *fileStore) LoadAll() ([]Event, error) {
	return fs.LoadStream("all")
}

func (fs *fileStore) LoadStream(id string) ([]Event, error) {
	filenames, err := fs.filenamesForStream(id)
	streamDir := filepath.Join(fs.dir, id)

	if _, err := os.Stat(streamDir); os.IsNotExist(err) {
		return NoEvents, &StorageError{
			Op:     "LoadStream",
			Stream: id,
			Err:    ErrNotFound,
		}
	}

	if err != nil {
		return NoEvents, &StorageError{
			Op:          "LoadStream",
			Stream:      id,
			Err:         ErrInternal,
			InternalErr: err,
		}
	}

	return fs.load(filenames)
}

func (fs *fileStore) filenamesForStream(id string) ([]string, error) {
	dirname := filepath.Join(fs.dir, id)
	dir, err := os.Open(dirname)
	if err != nil {
		return []string{}, err
	}

	fnames := []string{}
	if names, err := dir.Readdirnames(0); err != nil {
		return []string{}, err
	} else {
		for _, name := range names {
			fnames = append(fnames, filepath.Join(dirname, name))
		}
	}

	sort.Strings(fnames)

	return fnames, nil
}

func (fs *fileStore) load(filenames []string) ([]Event, error) {
	events := []Event{}
	for _, fname := range filenames {
		if event, err := fs.loadEvent(fname); err != nil {
			return NoEvents, err
		} else {
			events = append(events, event)
		}
	}

	return events, nil
}

func (fs *fileStore) loadEvent(fname string) (Event, error) {
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

func (fs *fileStore) eventForType(typename string) Event {
	typ, ok := fs.typeMap[typename]
	if !ok {
		panic(fmt.Errorf("type %q not registered.", typename))
	}

	return reflect.New(typ.Elem()).Interface().(Event)
}

func (fs *fileStore) Store(event Event) error {
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

func (fs *fileStore) storeForAll(now time.Time, data []byte) error {
	return fs.storeForAggregate(now, "all", data)
}

func (fs *fileStore) storeForAggregate(now time.Time, id string, data []byte) error {
	nowStr := fmt.Sprintf("%d", now.UnixNano())
	dirname := filepath.Join(fs.dir, id)
	fname := filepath.Join(dirname, nowStr)

	if _, err := os.Stat(dirname); os.IsNotExist(err) {
		os.MkdirAll(dirname, 0755)
	}

	out, err := os.OpenFile(fname, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, bytes.NewReader(data)); err != nil {
		return err
	} else {
		return nil
	}
}
