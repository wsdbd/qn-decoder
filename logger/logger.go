package logger

import (
  "log"
  "os"
  "fmt"
)

var logger *log.Logger

func init() {
  logger = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
}

func Println(args ...interface{}) {
  logger.Println(args...)
}

func Errorf(f string, v ...interface{}) {
	logger.Println(fmt.Sprintf(f, v...))
}
