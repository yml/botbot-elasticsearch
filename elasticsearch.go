package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/golang/glog"
)

type Elasticsearch struct {
	addr    string
	esIndex string
	esType  string //prefix by es because type is a builtin
}

func NewElasticsearch(addr, esIndex, esType string) *Elasticsearch {
	return &Elasticsearch{addr: addr, esIndex: esIndex, esType: esType}
}

func (es *Elasticsearch) newRequest(method string, body io.Reader) (*http.Request, error) {
	urlStr := fmt.Sprintf("http://%s/%s/%s", es.addr, es.esIndex, es.esType)
	glog.V(3).Infoln("URL string used to build the request to elasticsearch:", urlStr)
	req, err := http.NewRequest(method, urlStr, body)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Accept", "application/json")
	return req, err
}

func (es *Elasticsearch) Do(method string, v interface{}) (*http.Response, error) {
	client := &http.Client{}
	b, err := json.Marshal(v)
	if err != nil {
		glog.Errorln("An error occured while marshiling to json:", err)
		return nil, err
	}
	glog.V(3).Infoln("json:", string(b))

	req, err := es.newRequest(method, bytes.NewReader(b))
	if err != nil {
		glog.Errorln("An error occured while bulding elasticsearch request:", err)
		return nil, err
	}
	return client.Do(req)

}

func (es *Elasticsearch) Index(v interface{}) (body []byte, err error) {
	resp, err := es.Do("POST", v)
	if err != nil {
		glog.Errorln("An error occured while doing the request to elasticsearch:", err)
		return nil, err
	}
	defer resp.Body.Close()
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		glog.Errorln("An error occured while reading the response from elasticsearch", err)
	}
	glog.V(3).Infoln("Response from :", string(body))
	return body, err
}
