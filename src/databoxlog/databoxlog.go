package databoxlog

import (
	"encoding/json"
	corestore "lib-go-databox/coreStoreClient"
	databoxtype "lib-go-databox/types"
	"log"
	"runtime"
	"strconv"
)

type Logger struct {
	Store *corestore.CoreStoreClient
}

type LogEntries struct {
	Msg  string `json:"msg"`
	Type string `json:"type"`
}

type Logs []LogEntries

func New(store *corestore.CoreStoreClient) (*Logger, error) {

	dsmd := databoxtype.DataSourceMetadata{
		Description:    "container manager logs",
		ContentType:    "aplication/json",
		Vendor:         "databox",
		DataSourceType: "databox-logs",
		DataSourceID:   "cmlogs",
		StoreType:      "tsblob",
		IsActuator:     false,
		Unit:           "",
		Location:       "",
	}

	err := store.RegisterDatasource(dsmd)
	ChkErr(err)

	return &Logger{
		Store: store,
	}, nil
}

func (l Logger) Info(msg string) {
	Info(msg)
	err := l.Store.TSBlobWrite("cmlogs", []byte("{\"log\":"+strconv.Quote(msg)+",\"type\":\"INFO\"}"))
	ChkErr(err)
}
func (l Logger) Warn(msg string) {
	Warn(msg)
	err := l.Store.TSBlobWrite("cmlogs", []byte("{\"log\":"+strconv.Quote(msg)+",\"type\":\"WARN\"}"))
	ChkErr(err)
}
func (l Logger) Err(msg string) {
	Err(msg)
	err := l.Store.TSBlobWrite("cmlogs", []byte("{\"log\":"+strconv.Quote(msg)+",\"type\":\"ERROR\"}"))
	ChkErr(err)
}
func (l Logger) Debug(msg string) {
	Debug(msg)
	err := l.Store.TSBlobWrite("cmlogs", []byte("{\"log\":"+strconv.Quote(msg)+",\"type\":\"DEBUG\"}"))
	ChkErr(err)
}

func (l Logger) ChkErr(err error) {
	if err == nil {
		return
	}
	if debug == true {
		Err(err.Error())
		l.Err(err.Error())
	}
}

func (l Logger) GetLastNLogEntries(n int) Logs {

	var logs Logs
	data, err := l.Store.TSBlobLastN("cmlogs", n)
	l.ChkErr(err)
	json.Unmarshal(data, &logs)

	return logs
}

func (l Logger) GetLastNLogEntriesRaw(n int) []byte {

	data, err := l.Store.TSBlobLastN("cmlogs", n)
	l.ChkErr(err)
	return data

}

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
	log.SetFlags(log.Ldate | log.Ltime)
	_, file, line, ok := runtime.Caller(3)
	if !ok {
		file = "???"
		line = 0
	}

	log.Println(file + " L" + strconv.Itoa(line) + ":" + msg)
}

func Debug(msg string) {
	if debug == true {
		log.SetPrefix("[DEBUG]")
		log.SetFlags(log.LstdFlags)
		log.Println(msg)
	}

}
