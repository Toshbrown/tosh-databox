package databoxerrors

import (
	"log"
)

const debug = true

func ChkErr(err error) {
	if err == nil {
		return
	}
	if debug == true {
		Err(err.Error())
	}
}

func ChkErrFatal(err error) {
	if err == nil {
		return
	}
	log.Fatal("[ERROR]" + err.Error())
}

func Info(msg string) {
	log.SetPrefix("[INFO]")
	log.SetFlags(log.LstdFlags)
	log.Println(msg)
}

func Warn(msg string) {
	log.SetPrefix("[WARNING]")
	log.SetFlags(log.LstdFlags)
	log.Println(msg)
}

func Err(msg string) {
	log.SetPrefix("[ERROR]")
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.Println(msg)
}

func Debug(msg string) {
	if debug == true {
		log.SetPrefix("[DEBUG]")
		log.SetFlags(log.LstdFlags)
		log.Println(msg)
	}

}
