package internal

import (
	"log"
	"os"
)

//Logger ...
type Logger struct {
	logger *log.Logger
}

//NewDefaultLogger ...
func NewDefaultLogger() *Logger {
	logger := &Logger{
		logger: log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lmicroseconds),
	}
	return logger
}

//Debug ...
func (l *Logger) Debug(message string) {
	l.logger.Println("DEBUG: " + message)
}

//Info ...
func (l *Logger) Info(message string) {
	l.logger.Println("INFO: " + message)
}

//Error ...
func (l *Logger) Error(err error, message string) {
	finalMsg := "ERROR: "
	if message != "" {
		finalMsg += "msg: " + message + " | err: "
	}
	finalMsg += err.Error()
	l.logger.Println(finalMsg)
}
