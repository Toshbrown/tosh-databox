package coreStoreClient

import (
	"encoding/json"
	"errors"
	"strconv"
)

// TSBlobWrite will add data to the times series data store. Data will be time stamped at insertion (format ms since 1970)
func (csc *CoreStoreClient) TSBlobWrite(dataSourceID string, payload []byte) error {

	path := "/ts/blob/" + dataSourceID

	return csc.write(path, payload)

}

// TSBlobWriteAt will add data to the times series data store. Data will be time stamped with the timstamp provided in the
// timstamp paramiter (format ms since 1970)
func (csc *CoreStoreClient) TSBlobWriteAt(dataSourceID string, timstamp int64, payload []byte) error {

	path := "/ts/blob/" + dataSourceID + "/at/"

	token, err := csc.Arbiter.RequestToken(csc.ZEndpoint+path+"*", "POST")
	if err != nil {
		return err
	}

	path = path + strconv.FormatInt(timstamp, 10)

	err = csc.ZestC.Post(string(token), path, payload, "JSON")
	if err != nil {
		csc.Arbiter.InvalidateCache(csc.ZEndpoint+path+"*", "POST")
		return errors.New("Error writing: " + err.Error())
	}

	return nil

}

//TSBlobLatest will retrieve the last entry stored at the requested datasource ID
// return data is a byte array contingin JSON of the format
// {"timestamp":213123123,"data":[data-written-by-driver]}
func (csc *CoreStoreClient) TSBlobLatest(dataSourceID string) ([]byte, error) {

	path := "/ts/blob/" + dataSourceID + "/latest"

	return csc.read(path)

}

// TSBlobEarliest will retrieve the first entry stored at the requested datasource ID
// return data is a byte array contingin JSON of the format
// {"timestamp":213123123,"data":[data-written-by-driver]}
func (csc *CoreStoreClient) TSBlobEarliest(dataSourceID string) ([]byte, error) {

	path := "/ts/blob/" + dataSourceID + "/earliest"

	token, err := csc.Arbiter.RequestToken(csc.ZEndpoint+path, "GET")
	if err != nil {
		return []byte(""), err
	}

	resp, getErr := csc.ZestC.Get(string(token), path, "JSON")
	if getErr != nil {
		csc.Arbiter.InvalidateCache(csc.ZEndpoint+path, "GET")
		return []byte(""), errors.New("Error getting earliest data: " + getErr.Error())
	}

	return resp, nil

}

// LastN will retrieve the last N entries stored at the requested datasource ID
// return data is a byte array contingin JSON of the format
// {"timestamp":213123123,"data":[data-written-by-driver]}
func (csc *CoreStoreClient) TSBlobLastN(dataSourceID string, n int) ([]byte, error) {

	path := "/ts/blob/" + dataSourceID + "/last/" + strconv.Itoa(n)

	token, err := csc.Arbiter.RequestToken(csc.ZEndpoint+path, "GET")
	if err != nil {
		return []byte(""), err
	}

	resp, getErr := csc.ZestC.Get(string(token), path, "JSON")
	if getErr != nil {
		csc.Arbiter.InvalidateCache(csc.ZEndpoint+path, "GET")
		return []byte(""), errors.New("Error getting latest data: " + getErr.Error())
	}

	return resp, nil

}

// FirstN will retrieve the first N entries stored at the requested datasource ID
// return data is a byte array contingin JSON of the format
// {"timestamp":213123123,"data":[data-written-by-driver]}
func (csc *CoreStoreClient) TSBlobFirstN(dataSourceID string, n int) ([]byte, error) {

	path := "/ts/blob/" + dataSourceID + "/first/" + strconv.Itoa(n)

	token, err := csc.Arbiter.RequestToken(csc.ZEndpoint+path, "GET")
	if err != nil {
		return []byte(""), err
	}

	resp, getErr := csc.ZestC.Get(string(token), path, "JSON")
	if getErr != nil {
		csc.Arbiter.InvalidateCache(csc.ZEndpoint+path, "GET")
		return []byte(""), errors.New("Error getting latest data: " + getErr.Error())
	}

	return resp, nil

}

// TSBlobSince will retrieve all entries since the requested timestamp (ms since unix epoch)
// return data is a byte array contingin JSON of the format
// {"timestamp":213123123,"data":[data-written-by-driver]}
func (csc *CoreStoreClient) TSBlobSince(dataSourceID string, sinceTimeStamp int64) ([]byte, error) {

	path := "/ts/blob/" + dataSourceID + "/since/" + strconv.FormatInt(sinceTimeStamp, 10)

	token, err := csc.Arbiter.RequestToken(csc.ZEndpoint+path, "GET")
	if err != nil {
		return []byte(""), err
	}

	resp, getErr := csc.ZestC.Get(string(token), path, "JSON")
	if getErr != nil {
		csc.Arbiter.InvalidateCache(csc.ZEndpoint+path, "GET")
		return []byte(""), errors.New("Error getting latest data: " + getErr.Error())
	}

	return resp, nil

}

// TSBlobRange will retrieve all entries between  formTimeStamp and toTimeStamp timestamp in ms since unix epoch
// return data is a byte array contingin JSON of the format
// {"timestamp":213123123,"data":[data-written-by-driver]}
func (csc *CoreStoreClient) TSBlobRange(dataSourceID string, formTimeStamp int64, toTimeStamp int64) ([]byte, error) {

	path := "/ts/blob/" + dataSourceID + "/range/" + strconv.FormatInt(formTimeStamp, 10) + "/" + strconv.FormatInt(toTimeStamp, 10)

	token, err := csc.Arbiter.RequestToken(csc.ZEndpoint+path, "GET")
	if err != nil {
		return []byte(""), err
	}

	resp, getErr := csc.ZestC.Get(string(token), path, "JSON")
	if getErr != nil {
		csc.Arbiter.InvalidateCache(csc.ZEndpoint+path, "GET")
		return []byte(""), errors.New("Error getting latest data: " + getErr.Error())
	}

	return resp, nil

}

func (csc *CoreStoreClient) Length(dataSourceID string) (int, error) {
	path := "/ts/blob/" + dataSourceID + "/length"

	token, err := csc.Arbiter.RequestToken(csc.ZEndpoint+path, "GET")
	if err != nil {
		return 0, err
	}

	resp, getErr := csc.ZestC.Get(string(token), path, "JSON")
	if getErr != nil {
		csc.Arbiter.InvalidateCache(csc.ZEndpoint+path, "GET")
		return 0, errors.New("Error getting latest data: " + getErr.Error())
	}

	type legnthResult struct {
		Length int `json:"length"`
	}

	var val legnthResult
	err = json.Unmarshal(resp, &val)
	if err != nil {
		return 0, err
	}

	return val.Length, nil
}

// TSBlobObserve allows you to get notifications when a new value is written by a driver
// the returned chan receives chan []byte continging json of the
// form {"TimestampMS":213123123,"Json":byte[]}
func (csc *CoreStoreClient) TSBlobObserve(dataSourceID string) (<-chan []byte, error) {

	path := "/ts/blob/" + dataSourceID

	token, err := csc.Arbiter.RequestToken(csc.ZEndpoint+path, "GET")
	if err != nil {
		return nil, err
	}

	payloadChan, getErr := csc.ZestC.Observe(string(token), path, "JSON", 0)
	if getErr != nil {
		csc.Arbiter.InvalidateCache(csc.ZEndpoint+path, "GET")
		return nil, errors.New("Error observing: " + getErr.Error())
	}

	return payloadChan, nil

}
