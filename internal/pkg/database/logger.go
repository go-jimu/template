package database

import (
	"context"
	"fmt"
	"log/slog"

	"xorm.io/xorm/log"
)

type XormSlog struct {
	logger *slog.Logger
	isShow bool
	off    bool
}

var _ log.Logger = (*XormSlog)(nil)

func NewXormSlog(logger *slog.Logger) *XormSlog {
	logger = logger.With(slog.String("module", "xorm"))
	return &XormSlog{logger: logger, isShow: true}
}

func (l *XormSlog) Debug(v ...any) {
	l.logger.Debug(fmt.Sprint(v...))
}

func (l *XormSlog) Debugf(format string, v ...any) {
	l.logger.Debug(fmt.Sprintf(format, v...))
}

func (l *XormSlog) Error(v ...any) {
	l.logger.Error(fmt.Sprint(v...))
}

func (l *XormSlog) Errorf(format string, v ...any) {
	l.logger.Error(fmt.Sprintf(format, v...))
}

func (l *XormSlog) Info(v ...any) {
	l.logger.Info(fmt.Sprint(v...))
}

func (l *XormSlog) Infof(format string, v ...any) {
	l.logger.Info(fmt.Sprintf(format, v...))
}

func (l *XormSlog) Warn(v ...any) {
	l.logger.Warn(fmt.Sprint(v...))
}

func (l *XormSlog) Warnf(format string, v ...any) {
	l.logger.Warn(fmt.Sprintf(format, v...))
}

func (l *XormSlog) Level() log.LogLevel {
	if l.off {
		return log.LOG_OFF
	}

	for _, level := range []slog.Level{slog.LevelError, slog.LevelWarn, slog.LevelInfo, slog.LevelDebug} {
		if l.logger.Enabled(context.TODO(), level) {
			switch level {
			case slog.LevelDebug:
				return log.LOG_DEBUG
			case slog.LevelInfo:
				return log.LOG_INFO
			case slog.LevelWarn:
				return log.LOG_WARNING
			case slog.LevelError:
				return log.LOG_ERR
			}
		}
	}
	return log.LOG_UNKNOWN
}

func (l *XormSlog) SetLevel(level log.LogLevel) {
	l.logger.Warn("cannot set level for slog after created")
}

func (l *XormSlog) ShowSQL(show ...bool) {
	l.isShow = show[0]
}

func (l *XormSlog) IsShowSQL() bool {
	return l.isShow
}
