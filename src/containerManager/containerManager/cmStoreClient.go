package containerManager

import (
	"encoding/json"

	libDatabox "github.com/toshbrown/lib-go-databox"
)

type CMStore struct {
	Store *libDatabox.CoreStoreClient
}

const slaStoreID = "slaStore"

func NewCMStore(store *libDatabox.CoreStoreClient) *CMStore {

	//setup SLAStore
	store.RegisterDatasource(libDatabox.DataSourceMetadata{
		Description:    "Persistant SLA storage",
		ContentType:    "json",
		Vendor:         "databox",
		DataSourceType: "SLA",
		DataSourceID:   slaStoreID,
		StoreType:      "kv",
		IsActuator:     false,
		Location:       "",
		Unit:           "",
	})

	return &CMStore{Store: store}
}

func (s CMStore) SaveSLA(sla libDatabox.SLA) error {

	payload, err := json.Marshal(sla)
	if err != nil {
		return err
	}

	return s.Store.KVJSONWrite(slaStoreID, sla.Name, payload)

}

func (s CMStore) GetAllSLAs() ([]libDatabox.SLA, error) {

	var slaList []libDatabox.SLA

	keys, err := s.Store.KVJSONListKeys(slaStoreID)
	if err != nil {
		return nil, err
	}

	for _, k := range keys {
		var sla libDatabox.SLA
		payload, err := s.Store.KVJSONRead(slaStoreID, k)
		if err != nil {
			libDatabox.Err("[GetAllSLAs] failed to get  " + slaStoreID + ". " + err.Error())
			continue
		}
		err = json.Unmarshal(payload, &sla)
		if err != nil {
			libDatabox.Err("[GetAllSLAs] failed decode SLA for " + slaStoreID + ". " + err.Error())
			continue
		}
		slaList = append(slaList, sla)
	}

	return slaList, err

}

func (s CMStore) DeleteSLA(name string) error {
	return s.Store.KVJSONDelete(slaStoreID, name)
}

func (s CMStore) ClearSLADatabase() error {
	return s.Store.KVJSONDeleteAll(slaStoreID)
}

func (s CMStore) SavePassword(password string) error {
	return s.Store.KVTextWrite(slaStoreID, "CMPassword", []byte(password))
}

func (s CMStore) LoadPassword() (string, error) {
	password, err := s.Store.KVTextRead(slaStoreID, "CMPassword")
	return string(password), err
}
