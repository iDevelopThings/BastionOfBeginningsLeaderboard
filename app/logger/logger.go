package logger

import (
	config "github.com/go-ozzo/ozzo-config"
	log "github.com/go-ozzo/ozzo-log"
)

var Logger = log.NewLogger()

func Emergency(format string, a ...interface{}) { Logger.Emergency(format, a...) }
func Alert(format string, a ...interface{})     { Logger.Alert(format, a...) }
func Critical(format string, a ...interface{})  { Logger.Critical(format, a...) }
func Error(format string, a ...interface{})     { Logger.Error(format, a...) }
func Warning(format string, a ...interface{})   { Logger.Warning(format, a...) }
func Notice(format string, a ...interface{})    { Logger.Notice(format, a...) }
func Info(format string, a ...interface{})      { Logger.Info(format, a...) }
func Debug(format string, a ...interface{})     { Logger.Debug(format, a...) }

func Init(c *config.Config) {
	c.Register("ConsoleTarget", log.NewConsoleTarget)
	c.Register("FileTarget", log.NewFileTarget)

	if err := c.Configure(Logger, "Logger"); err != nil {
		panic(err)
	}

	if err := Logger.Open(); err != nil {
		panic(err)
	}

	Debug("Logger initialized")
}
