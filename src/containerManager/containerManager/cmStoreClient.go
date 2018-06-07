package containerManager

import (
	log "databoxlog"
	"encoding/json"
	"lib-go-databox/coreStoreClient"
	databoxTypes "lib-go-databox/types"
)

type CMStore struct {
	Store *coreStoreClient.CoreStoreClient
}

const slaStoreID = "slaStore"

//TODO This is all disabled until core store is updated to 0.0.7 as 0.0.6 has no delete support

func NewCMStore(store *coreStoreClient.CoreStoreClient) *CMStore {

	//setup SLAStore
	store.RegisterDatasource(databoxTypes.DataSourceMetadata{
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

func (s CMStore) SaveSLA(sla databoxTypes.SLA) error {

	payload, err := json.Marshal(sla)
	if err != nil {
		return err
	}

	return s.Store.KVJSONWrite(slaStoreID, sla.Name, payload)

}

func (s CMStore) GetAllSLAs() ([]databoxTypes.SLA, error) {

	var slaList []databoxTypes.SLA

	keys, err := s.Store.KVJSONListKeys(slaStoreID)
	if err != nil {
		return nil, err
	}

	for _, k := range keys {
		var sla databoxTypes.SLA
		payload, err := s.Store.KVJSONRead(slaStoreID, k)
		if err != nil {
			log.Err("[GetAllSLAs] failed to get  " + slaStoreID + ". " + err.Error())
			continue
		}
		err = json.Unmarshal(payload, &sla)
		if err != nil {
			log.Err("[GetAllSLAs] failed decode SLA for " + slaStoreID + ". " + err.Error())
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
