package main

import (
	"fmt"
	"io"
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
