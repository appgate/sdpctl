package util

import (
	"fmt"
	"net"
	"time"

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
)

func NewHook(protocol, address string, levels []logrus.Level, version int) *Hook {
	formatter := &logrus.JSONFormatter{
		FieldMap:        fieldMap,
		TimestampFormat: time.RFC3339,
	}

	fields := logrus.Fields{
		"@version": fmt.Sprint(version), // API version used
		"@key":     uuid.New(),          // This will be unique for every command being run
	}

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

	conn, err := net.Dial(h.protocol, h.address)
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.Write(bytes)
	if err != nil {
		return err
	}
	return nil
}

func (h *Hook) Levels() []logrus.Level {
	return h.levels
}
