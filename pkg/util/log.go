package util

import (
	"io"
	"net"
	"os"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Hook struct {
	formatter logrus.Formatter
	levels    []logrus.Level
	fields    logrus.Fields
	protocol  string
	address   string
}

var (
	fieldMap = logrus.FieldMap{
		logrus.FieldKeyTime:  "@time",
		logrus.FieldKeyLevel: "@level",
		logrus.FieldKeyMsg:   "message",
	}
	fields = logrus.Fields{
		"@key": uuid.New(), // This will be unique for every command being run
	}
)

func NewHook(protocol, address string, levels []logrus.Level, formatter logrus.Formatter) *Hook {
	logrus.New()
	return &Hook{
		formatter: formatter,
		fields:    fields,
		protocol:  protocol,
		address:   address,
		levels:    levels,
	}
}

func (h *Hook) Fire(entry *logrus.Entry) error {
	for k, v := range h.fields {
		if _, ok := h.fields[k]; !ok {
			h.fields[k] = v
		}
	}
	for k, v := range h.fields {
		entry.Data[k] = v
	}
	bytes, err := h.formatter.Format(entry)
	if err != nil {
		return err
	}

	var w io.WriteCloser
	if h.protocol != "file" {
		w, err = net.Dial(h.protocol, h.address)
		if err != nil {
			return err
		}
	} else {
		w, err = os.OpenFile(h.address, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if err != nil {
			return err
		}
	}
	defer w.Close()

	_, err = w.Write(bytes)
	if err != nil {
		return err
	}
	return nil
}

func (h *Hook) Levels() []logrus.Level {
	return h.levels
}
