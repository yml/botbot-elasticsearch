package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/golang/glog"
)

const (
	esHost  = "localhost"
	esPort  = "9200"
	esIndex = "botbot"
	esType  = "line"
)

type Elasticsearch struct {
	host    string
	port    string
	esIndex string
	esType  string //prefix by es because type is a builtin
}

func NewElasticsearch(host, port, esIndex, esType string) *Elasticsearch {
	return &Elasticsearch{host: host, port: port, esIndex: esIndex, esType: esType}
}

func (es *Elasticsearch) newRequest(method string, body io.Reader) (*http.Request, error) {
	urlStr := fmt.Sprintf("http://%s:%s/%s/%s/1", es.host, es.port, es.esIndex, es.esType)
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

	req, err := es.newRequest("PUT", bytes.NewReader(b))
	if err != nil {
		glog.Errorln("An error occured while bulding elasticsearch request:", err)
		return nil, err
	}
	return client.Do(req)

}

func (es *Elasticsearch) Index(v interface{}) (err error) {
	resp, err := es.Do("PUT", v)
	if err != nil {
		glog.Errorln("An error occured while doing the request to elasticsearch:", err)
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	glog.V(3).Infoln("Response from :", string(body))
	return err
}

func main() {
	flag.Parse() // We need to capture arguments for glog
	glog.Infoln("Starting botbot-elasticsearch", esHost, esPort)
	line := struct{ Text, Nick string }{Text: "hello world", Nick: "yml"}
	es := NewElasticsearch(esHost, esPort, esIndex, esType)
	es.Index(&line)
}
