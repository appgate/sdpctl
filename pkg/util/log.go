package util

import (
	"fmt"
	"io"
	"net"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Hook struct {
	formatter logrus.Formatter
	levels    []logrus.Level
	fields    logrus.Fields
	writer    io.Writer
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

func NewHook(protocol, address string, levels []logrus.Level, formatter logrus.Formatter) (*Hook, error) {
	w, err := net.Dial(protocol, address)
	if err != nil {
		return nil, err
	}

	return &Hook{
		writer:    w,
		formatter: formatter,
		fields:    fields,
		levels:    levels,
	}, nil
}

func (h *Hook) Fire(entry *logrus.Entry) error {
	if h.writer == nil {
		return fmt.Errorf("no socket connection present")
	}

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

	_, err = h.writer.Write(bytes)
	if err != nil {
		return err
	}
	return nil
}

func (h *Hook) Levels() []logrus.Level {
	return h.levels
}
