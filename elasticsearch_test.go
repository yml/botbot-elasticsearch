package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/glog"
)

// SetGlogFlags walk around a glog issue and force it to log to stderr.
// It need to be called at the beginning of each test.
func SetGlogFlags() {
	flag.Set("logtostderr", "true")
	flag.Set("v", "4")
}

func mockElasticsearchServer(t *testing.T) *httptest.Server {
	expectedURL := "/botbot/line"
	expectedBody := `{"Text":"hello world","Nick":"yml"}`
	result := `{"_index":"botbot","_type":"line","_id":"B8K2BW9yRkWPEkD48gKmrg","_version":1,"created":true}`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.String() != expectedURL {
			t.Error("Expected", expectedURL, "received", r.URL)
		}
		body, _ := ioutil.ReadAll(r.Body)
		if string(body) != expectedBody {
			t.Error("Expected", expectedBody, "received", string(body))
		}
		glog.Infoln("[Test] Body received", string(body))
		glog.Infoln("[Test] r.URL", r.URL)
		fmt.Fprintln(w, result)
	}))
	return ts
}

func Test_ElasticsearchDo(t *testing.T) {
	SetGlogFlags()
	esTs := mockElasticsearchServer(t)
	defer esTs.Close()
	addr := esTs.Listener.Addr().String()
	glog.Infoln("[Test] esTs.URL", addr)
	es := NewElasticsearch(addr, "botbot", "line")
	line := struct{ Text, Nick string }{Text: "hello world", Nick: "yml"}
	body, err := es.Index(line)
	if err != nil {
		t.Fatal("An error occured while indexing the line", err)
	}
	glog.Infoln("[Test] Return by the mockElasticsearchServer", string(body))
}
