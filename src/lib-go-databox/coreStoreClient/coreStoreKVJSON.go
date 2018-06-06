package coreStoreClient

import (
	"encoding/json"
	"errors"
)

// KVJSONWrite Write will add data to the key value data store.
func (csc *CoreStoreClient) KVJSONWrite(dataSourceID string, key string, payload []byte) error {

	path := "/kv/" + dataSourceID + "/" + key

	return csc.write(path, payload)

}

// KVJSONRead will read the vale store at under tha key
// return data is a JSON object of the format {"timestamp":213123123,"data":[data-written-by-driver]}
func (csc *CoreStoreClient) KVJSONRead(dataSourceID string, key string) ([]byte, error) {

	path := "/kv/" + dataSourceID + "/" + key

	return csc.read(path)

}

// KVJSONListKeys returns an array of key registed under the dataSourceID
func (csc *CoreStoreClient) KVJSONListKeys(dataSourceID string) ([]string, error) {

	path := "/kv/" + dataSourceID + "/keys"

	data, err := csc.read(path)
	if err != nil {
		return []string{}, err
	}

	var keysArray []string

	err = json.Unmarshal(data, &keysArray)
	if err != nil {
		return []string{}, errors.New("KVJSONListKeys: Error decoding data. " + err.Error())
	}
	return keysArray, nil

}

// KVJSONDelete will delete the value store under that key
func (csc *CoreStoreClient) KVJSONDelete(dataSourceID string, key string) error {

	return errors.New("KVJSONDelete is not implicated yet github.com/toshbrown/goZestClient needs updating")

}

// KVJSONDeleteAll will delete the values store under the datasource
func (csc *CoreStoreClient) KVJSONDeleteAll(dataSourceID string) error {

	return errors.New("KVJSONDeleteAll is not implicated yet github.com/toshbrown/goZestClient needs updating")

}

func (csc *CoreStoreClient) KVJSONObserve(dataSourceID string) (<-chan []byte, error) {

	path := "/kv/" + dataSourceID

	return csc.observe(path)

}

func (csc *CoreStoreClient) KVJSONObserveKey(dataSourceID string, key string) (<-chan []byte, error) {

	path := "/KV/" + dataSourceID + "/" + key

	return csc.observe(path)

}
