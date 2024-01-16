package logger

import (
	// "errors"

	"os"
	"strings"

	"github.com/rs/zerolog"
)

var (
	_logger zerolog.Logger
)

type LogItem struct {
	event *zerolog.Event
}

func Init(env string) {
	// zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	_logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
	if env != "local" {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
}

func newLogItem(event *zerolog.Event) *LogItem {
	return &LogItem{
		event: event,
	}
}

func Info() *LogItem {
	return newLogItem(_logger.Info())
}

func Debug() *LogItem {
	return newLogItem(_logger.Debug())
}

func Warn() *LogItem {
	return newLogItem(_logger.Warn())
}

func Error() *LogItem {
	return newLogItem(_logger.Error())
}

func Panic() *LogItem {
	return newLogItem(_logger.Panic())
}

// data
func (l *LogItem) D(key string, data interface{}) *LogItem {
	l.event = l.event.Interface(key, data)
	return l
}

// error
func (l *LogItem) E(err error) *LogItem {
	l.event = l.event.Err(err)
	return l
}

// write
func (l *LogItem) W(messages ...string) {
	if len(messages) == 0 {
		l.event.Send()
	} else {
		msg := strings.Join(messages, " ")
		l.event.Msg(msg)
	}
}

// write format
func (l *LogItem) Wf(message string, a ...interface{}) {
	l.event.Msgf(message, a...)
}
