package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type Mailer interface {
	SendMessage(*MailMessage) error
}

type MailMessage struct {
	To   []string
	Body []byte
}

type FileMailer struct {
	dir string
}

func NewFileMailer(dir string) (*FileMailer, error) {
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		return nil, err
	} else {
		if err := os.MkdirAll(dir, 0755); !os.IsExist(err) {
			return nil, err
		}
	}

	return &FileMailer{dir: dir}, nil
}

func (mailer *FileMailer) SendMessage(msg *MailMessage) error {
	data, err := json.MarshalIndent(msg, "", "  ")
	if err != nil {
		return err
	}

	fname := filepath.Join(mailer.dir, fmt.Sprintf("%d", time.Now().UnixNano()))
	return ioutil.WriteFile(fname, data, 0644)
}

type SystemMailer string

func NewSystemMailer(pathToSendmail string) (SystemMailer, error) {
	return SystemMailer(pathToSendmail), nil
}

func (sendmail SystemMailer) SendMessage(msg *MailMessage) error {
	cmd := exec.Command(string(sendmail), msg.To...)
	cmd.Stdin = bytes.NewReader(msg.Body)
	return cmd.Run()
}
