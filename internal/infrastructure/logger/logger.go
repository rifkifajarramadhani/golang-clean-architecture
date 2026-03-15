package logger

import (
	"log"
	"os"
)

var Logger *log.Logger

func Init() {
	os.MkdirAll("logs", os.ModePerm)
	file, err := os.OpenFile(
		"logs/app.log",
		os.O_CREATE|os.O_APPEND|os.O_WRONLY,
		0666,
	)
	if err != nil {
		log.Fatal(err)
	}

	Logger = log.New(file, "APP: ", log.Ldate|log.Ltime|log.Lshortfile)
}
