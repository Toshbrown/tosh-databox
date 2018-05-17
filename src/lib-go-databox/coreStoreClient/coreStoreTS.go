package coreStoreClient

import (
	"errors"
	"strconv"

	arbiterClient "lib-go-databox/arbiterClient"

	zest "github.com/toshbrown/goZestClient"
)

type TScoreStoreClient_0_3_0 interface {
	// Write  will be timestamped with write time in ms since the unix epoch by the store
	TSWrite(dataSourceID string, payload []byte) error
	// WriteAt will be timestamped with timestamp provided in ms since the unix epoch
	TSWriteAt(dataSourceID string, timestamp int64, payload []byte) error
	// Read the latest value.
	// return data is a JSON object of the format {"timestamp":213123123,"data":[data-written-by-driver]}
	TSLatest(dataSourceID string) ([]byte, error)
	// Read the earliest value.
	// return data is a JSON object of the format {"timestamp":213123123,"data":[data-written-by-driver]}
	TSEarliest(dataSourceID string) ([]byte, error)
	// Read the last N values.
	// return data is an array of JSON objects of the format {"timestamp":213123123,"data":[data-written-by-driver]}
	TSLastN(dataSourceID string, n int) ([]byte, error)
	// Read the first N values.
	// return data is an array of JSON objects of the format {"timestamp":213123123,"data":[data-written-by-driver]}
	TSFirstN(dataSourceID string, n int) ([]byte, error)
	// Read values written after the provided timestamp in in ms since the unix epoch.
	// return data is an array of JSON objects of the format {"timestamp":213123123,"data":[data-written-by-driver]}
	TSSince(dataSourceID string, sinceTimeStamp int64) ([]byte, error)
	// Read values written between the start timestamp and end timestamp in in ms since the unix epoch.
	// return data is an array of JSON objects of the format {"timestamp":213123123,"data":[data-written-by-driver]}
	TSRange(dataSourceID string, formTimeStamp int64, toTimeStamp int64) ([]byte, error)
	// Get notifications when a new value is written
	// the returned chan receives valuse of the form {"timestamp":213123123,"data":[data-written-by-driver]}
	TSObserve(dataSourceID string) (<-chan []byte, error)
}

type TSCoreStoreClient struct {
	zestC         zest.ZestClient
	arbiterClient arbiterClient.ArbiterClient
	zEndpoint     string
	dEndpoint     string
}

// NewJSONTimeSeriesClient returns a new jSONTimeSeriesClient to enable interaction with a time series data store in JSON format
// reqEndpoint is provided in the DATABOX_ZMQ_ENDPOINT environment varable to databox apps and drivers.
func NewTSCoreStoreClient(arbiterClient arbiterClient.ArbiterClient, zestClient zest.ZestClient) TScoreStoreClient_0_3_0 {

	tsc := TSCoreStoreClient{}
	tsc.zEndpoint = zestClient.Endpoint
	tsc.dEndpoint = zestClient.DealerEndpoint
	tsc.zestC = zestClient
	tsc.arbiterClient = arbiterClient

	return tsc
}

// Write will add data to the times series data store. Data will be time stamped at insertion (format ms since 1970)
func (tsc TSCoreStoreClient) TSWrite(dataSourceID string, payload []byte) error {

	path := "/ts/" + dataSourceID

	token, err := tsc.arbiterClient.RequestToken(tsc.zEndpoint+path, "POST")
	if err != nil {
		return err
	}

	err = tsc.zestC.Post(string(token), path, payload, "JSON")
	if err != nil {
		tsc.arbiterClient.InvalidateCache(tsc.zEndpoint+path, "POST")
		return errors.New("Error writing: " + err.Error())
	}

	return nil

}

// WriteAt will add data to the times series data store. Data will be time stamped with the timstamp provided in the
// timstamp paramiter (format ms since 1970)
func (tsc TSCoreStoreClient) TSWriteAt(dataSourceID string, timstamp int64, payload []byte) error {

	path := "/ts/" + dataSourceID + "/at/"

	token, err := tsc.arbiterClient.RequestToken(tsc.zEndpoint+path+"*", "POST")
	if err != nil {
		return err
	}

	path = path + strconv.FormatInt(timstamp, 10)

	err = tsc.zestC.Post(string(token), path, payload, "JSON")
	if err != nil {
		tsc.arbiterClient.InvalidateCache(tsc.zEndpoint+path+"*", "POST")
		return errors.New("Error writing: " + err.Error())
	}

	return nil

}

//Latest will retrieve the last entry stored at the requested datasource ID
// return data is a JSON object of the format {"timestamp":213123123,"data":[data-written-by-driver]}
func (tsc TSCoreStoreClient) TSLatest(dataSourceID string) ([]byte, error) {

	path := "/ts/" + dataSourceID + "/latest"

	token, err := tsc.arbiterClient.RequestToken(tsc.zEndpoint+path, "GET")
	if err != nil {
		return []byte(""), err
	}

	resp, getErr := tsc.zestC.Get(string(token), path, "JSON")
	if getErr != nil {
		tsc.arbiterClient.InvalidateCache(tsc.zEndpoint+path, "GET")
		return []byte(""), errors.New("Error getting latest data: " + getErr.Error())
	}

	return resp, nil

}

// Earliest will retrieve the first entry stored at the requested datasource ID
// return data is a JSON object of the format {"timestamp":213123123,"data":[data-written-by-driver]}
func (tsc TSCoreStoreClient) TSEarliest(dataSourceID string) ([]byte, error) {

	path := "/ts/" + dataSourceID + "/earliest"

	token, err := tsc.arbiterClient.RequestToken(tsc.zEndpoint+path, "GET")
	if err != nil {
		return []byte(""), err
	}

	resp, getErr := tsc.zestC.Get(string(token), path, "JSON")
	if getErr != nil {
		tsc.arbiterClient.InvalidateCache(tsc.zEndpoint+path, "GET")
		return []byte(""), errors.New("Error getting earliest data: " + getErr.Error())
	}

	return resp, nil

}

// LastN will retrieve the last N entries stored at the requested datasource ID
// return data is an array of JSON objects of the format {"timestamp":213123123,"data":[data-written-by-driver]}
func (tsc TSCoreStoreClient) TSLastN(dataSourceID string, n int) ([]byte, error) {

	path := "/ts/" + dataSourceID + "/last/" + strconv.Itoa(n)

	token, err := tsc.arbiterClient.RequestToken(tsc.zEndpoint+path, "GET")
	if err != nil {
		return []byte(""), err
	}

	resp, getErr := tsc.zestC.Get(string(token), path, "JSON")
	if getErr != nil {
		tsc.arbiterClient.InvalidateCache(tsc.zEndpoint+path, "GET")
		return []byte(""), errors.New("Error getting latest data: " + getErr.Error())
	}

	return resp, nil

}

// FirstN will retrieve the first N entries stored at the requested datasource ID
// return data is an array of JSON objects of the format {"timestamp":213123123,"data":[data-written-by-driver]}
func (tsc TSCoreStoreClient) TSFirstN(dataSourceID string, n int) ([]byte, error) {

	path := "/ts/" + dataSourceID + "/first/" + strconv.Itoa(n)

	token, err := tsc.arbiterClient.RequestToken(tsc.zEndpoint+path, "GET")
	if err != nil {
		return []byte(""), err
	}

	resp, getErr := tsc.zestC.Get(string(token), path, "JSON")
	if getErr != nil {
		tsc.arbiterClient.InvalidateCache(tsc.zEndpoint+path, "GET")
		return []byte(""), errors.New("Error getting latest data: " + getErr.Error())
	}

	return resp, nil

}

//Since will retrieve all entries since the requested timestamp (ms since unix epoch)
// return data is a JSON object of the format {"timestamp":213123123,"data":[data-written-by-driver]}
func (tsc TSCoreStoreClient) TSSince(dataSourceID string, sinceTimeStamp int64) ([]byte, error) {

	path := "/ts/" + dataSourceID + "/since/" + strconv.FormatInt(sinceTimeStamp, 10)

	token, err := tsc.arbiterClient.RequestToken(tsc.zEndpoint+path, "GET")
	if err != nil {
		return []byte(""), err
	}

	resp, getErr := tsc.zestC.Get(string(token), path, "JSON")
	if getErr != nil {
		tsc.arbiterClient.InvalidateCache(tsc.zEndpoint+path, "GET")
		return []byte(""), errors.New("Error getting latest data: " + getErr.Error())
	}

	return resp, nil

}

// Range will retrieve all entries between  formTimeStamp and toTimeStamp timestamp in ms since unix epoch
// return data is a JSON object of the format {"timestamp":213123123,"data":[data-written-by-driver]}
func (tsc TSCoreStoreClient) TSRange(dataSourceID string, formTimeStamp int64, toTimeStamp int64) ([]byte, error) {

	path := "/ts/" + dataSourceID + "/range/" + strconv.FormatInt(formTimeStamp, 10) + "/" + strconv.FormatInt(toTimeStamp, 10)

	token, err := tsc.arbiterClient.RequestToken(tsc.zEndpoint+path, "GET")
	if err != nil {
		return []byte(""), err
	}

	resp, getErr := tsc.zestC.Get(string(token), path, "JSON")
	if getErr != nil {
		tsc.arbiterClient.InvalidateCache(tsc.zEndpoint+path, "GET")
		return []byte(""), errors.New("Error getting latest data: " + getErr.Error())
	}

	return resp, nil

}

func (tsc TSCoreStoreClient) TSObserve(dataSourceID string) (<-chan []byte, error) {

	path := "/ts/" + dataSourceID

	token, err := tsc.arbiterClient.RequestToken(tsc.zEndpoint+path, "GET")
	if err != nil {
		return nil, err
	}

	payloadChan, getErr := tsc.zestC.Observe(string(token), path, "JSON", 0)
	if getErr != nil {
		tsc.arbiterClient.InvalidateCache(tsc.zEndpoint+path, "GET")
		return nil, errors.New("Error observing: " + getErr.Error())
	}

	return payloadChan, nil

}
