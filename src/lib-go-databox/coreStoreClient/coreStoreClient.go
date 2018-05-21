package coreStoreClient

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	log "databoxerrors"

	arbiterClient "lib-go-databox/arbiterClient"
	databoxTypes "lib-go-databox/types"

	zest "github.com/toshbrown/goZestClient"
)

type CoreStoreClient struct {
	zestC     zest.ZestClient
	arbiter   arbiterClient.ArbiterClient
	request   *http.Client
	zEndpoint string
	dEndpoint string
	TS        TScoreStoreClient_0_3_0
}

func NewCoreStoreClient(databoxRequest *http.Client, arbiterClient arbiterClient.ArbiterClient, serverKeyPath string, storeEndPoint string, enableLogging bool) CoreStoreClient {
	csc := CoreStoreClient{
		arbiter: arbiterClient,
		request: databoxRequest,
	}

	//get the server key
	serverKey, err := ioutil.ReadFile(serverKeyPath)
	if err != nil {
		fmt.Println("Warning:: failed to read ZMQ_PUBLIC_KEY using default value")
		serverKey = []byte("vl6wu0A@XP?}Or/&BR#LSxn>A+}L)p44/W[wXL3<")
	}

	csc.zEndpoint = storeEndPoint
	csc.dEndpoint = strings.Replace(storeEndPoint, ":5555", ":5556", 1)
	csc.zestC, err = zest.New(csc.zEndpoint, csc.dEndpoint, string(serverKey), enableLogging)
	if err != nil {
		fmt.Println("[NewCoreStoreClient] Error zest.New ", err.Error())
	}

	csc.TS = NewTSCoreStoreClient(arbiterClient, csc.zestC)

	return csc
}

func (csc *CoreStoreClient) GetStoreDataSourceCatalogue(href string) (databoxTypes.HypercatRoot, error) {

	fmt.Println("[GetStoreDataSourceCatalogue] ", href)

	target := href + "/cat"
	method := "GET"

	token, err := csc.arbiter.RequestToken(target, method)
	if err != nil {
		return databoxTypes.HypercatRoot{}, err
	}
	log.Debug("[GetStoreDataSourceCatalogue] got Token: " + string(token))

	hypercatJSON, getErr := csc.zestC.Get(string(token), "/cat", "JSON")
	if getErr != nil {
		return databoxTypes.HypercatRoot{}, err
	}
	log.Debug("[GetStoreDataSourceCatalogue] got store cat: " + string(hypercatJSON))
	cat := databoxTypes.HypercatRoot{}
	json.Unmarshal(hypercatJSON, &cat)

	return cat, nil

}

// RegisterDatasource is used by apps and drivers to register datasource in stores they
// own.
func (csc *CoreStoreClient) RegisterDatasource(metadata databoxTypes.DataSourceMetadata) error {

	path := "/cat"

	token, err := csc.arbiter.RequestToken(csc.zEndpoint+path, "POST")
	if err != nil {
		return err
	}
	hypercatJSON, err := csc.dataSourceMetadataToHypercat(metadata, csc.zEndpoint+"/ts/")

	writeErr := csc.zestC.Post(string(token), path, hypercatJSON, "JSON")
	if writeErr != nil {
		csc.arbiter.InvalidateCache(csc.zEndpoint+path, "POST")
		return errors.New("Error writing: " + writeErr.Error())
	}

	return nil
}

//dataSourceMetadataToHypercat converts a DataSourceMetadata instance to json for registering a data source
func (csc *CoreStoreClient) dataSourceMetadataToHypercat(metadata databoxTypes.DataSourceMetadata, endPoint string) ([]byte, error) {

	if metadata.Description == "" ||
		metadata.ContentType == "" ||
		metadata.Vendor == "" ||
		metadata.DataSourceType == "" ||
		metadata.DataSourceID == "" ||
		metadata.StoreType == "" {

		return nil, errors.New("Missing required metadata")
	}

	cat := databoxTypes.HypercatItem{}
	cat.ItemMetadata = append(cat.ItemMetadata, databoxTypes.RelValPair{Rel: "urn:X-hypercat:rels:hasDescription:en", Val: metadata.Description})
	cat.ItemMetadata = append(cat.ItemMetadata, databoxTypes.RelValPair{Rel: "urn:X-hypercat:rels:isContentType", Val: metadata.ContentType})
	cat.ItemMetadata = append(cat.ItemMetadata, databoxTypes.RelValPair{Rel: "urn:X-databox:rels:hasVendor", Val: metadata.Vendor})
	cat.ItemMetadata = append(cat.ItemMetadata, databoxTypes.RelValPair{Rel: "urn:X-databox:rels:hasType", Val: metadata.DataSourceType})
	cat.ItemMetadata = append(cat.ItemMetadata, databoxTypes.RelValPair{Rel: "urn:X-databox:rels:hasDatasourceid", Val: metadata.DataSourceID})
	cat.ItemMetadata = append(cat.ItemMetadata, databoxTypes.RelValPair{Rel: "urn:X-databox:rels:hasStoreType", Val: metadata.StoreType})

	if metadata.IsActuator {
		cat.ItemMetadata = append(cat.ItemMetadata, databoxTypes.RelValPairBool{Rel: "urn:X-databox:rels:isActuator", Val: true})
	}

	if metadata.Location != "" {
		cat.ItemMetadata = append(cat.ItemMetadata, databoxTypes.RelValPair{Rel: "urn:X-databox:rels:hasLocation", Val: metadata.Location})
	}

	if metadata.Unit != "" {
		cat.ItemMetadata = append(cat.ItemMetadata, databoxTypes.RelValPair{Rel: "urn:X-databox:rels:hasUnit", Val: metadata.Unit})
	}

	cat.Href = endPoint + metadata.DataSourceID

	return json.Marshal(cat)

}

// HypercatToDataSourceMetadata is a helper function to convert the hypercat description of a datasource to a DataSourceMetadata instance
// Also returns the store url for this data source.
func (csc *CoreStoreClient) HypercatToDataSourceMetadata(hypercatDataSourceDescription string) (databoxTypes.DataSourceMetadata, string, error) {
	dm := databoxTypes.DataSourceMetadata{}

	hc := databoxTypes.HypercatItem{}
	err := json.Unmarshal([]byte(hypercatDataSourceDescription), &hc)
	if err != nil {
		return dm, "", err
	}

	for _, pair := range hc.ItemMetadata {
		vals := pair.(map[string]interface{})
		if vals["rel"].(string) == "urn:X-hypercat:rels:hasDescription:en" {
			dm.Description = vals["val"].(string)
			continue
		}
		if vals["rel"].(string) == "urn:X-hypercat:rels:isContentType" {
			dm.ContentType = vals["val"].(string)
			continue
		}
		if vals["rel"].(string) == "urn:X-databox:rels:hasVendor" {
			dm.Vendor = vals["val"].(string)
			continue
		}
		if vals["rel"].(string) == "urn:X-databox:rels:hasType" {
			dm.DataSourceType = vals["val"].(string)
			continue
		}
		if vals["rel"].(string) == "urn:X-databox:rels:hasDatasourceid" {
			dm.DataSourceID = vals["val"].(string)
			continue
		}
		if vals["rel"].(string) == "urn:X-databox:rels:hasStoreType" {
			dm.StoreType = vals["val"].(string)
			continue
		}
		if vals["rel"].(string) == "urn:X-databox:rels:isActuator" {
			dm.IsActuator = vals["val"].(bool)
			continue
		}
		if vals["rel"].(string) == "urn:X-databox:rels:hasLocation" {
			dm.Location = vals["val"].(string)
			continue
		}
		if vals["rel"].(string) == "urn:X-databox:rels:hasUnit" {
			dm.Unit = vals["val"].(string)
			continue
		}

	}

	url, getStoreURLErr := GetStoreURLFromDsHref(hc.Href)

	return dm, url, getStoreURLErr
}

// GetStoreURLFromDsHref extracts the base store url from the href provied in the hypercat descriptions.
func GetStoreURLFromDsHref(href string) (string, error) {

	u, err := url.Parse(href)
	if err != nil {
		return "", err
	}

	return u.Scheme + "://" + u.Host, nil

}
