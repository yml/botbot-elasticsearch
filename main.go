package main

import (
	"flag"

	"github.com/golang/glog"
)

const (
	esAddr  = "localhost:9200"
	esIndex = "botbot"
	esType  = "line"
)

func main() {
	flag.Parse() // We need to capture arguments for glog
	glog.Infoln("Starting botbot-elasticsearch pointing to", esAddr)
	line := struct{ Text, Nick string }{Text: "hello world", Nick: "yml"}
	es := NewElasticsearch(esAddr, esIndex, esType)
	es.Index(&line)
}
