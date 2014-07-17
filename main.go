package main

import (
	"encoding/json"
	"flag"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/BotBotMe/botbot-bot/common"
	"github.com/BotBotMe/botbot-bot/line"
	"github.com/golang/glog"
)

const (
	esAddr  = "localhost:9200"
	esIndex = "botbot"
	esType  = "line"
)

func listen(wg sync.WaitGroup, queue common.Queue, quit chan struct{}) chan line.Line {

	lines := make(chan line.Line)
	go func(lines chan line.Line, wg sync.WaitGroup) {
		wg.Add(1)
		for {
			select {
			case <-quit:
				wg.Done()
				glog.V(2).Infoln("Quiting the listen goroutine because we received a quit chan")
				return
			default:
				// Blocking on lpop for 5 s
				_, val, err := queue.Blpop([]string{"q"}, 5)
				if err != nil {
					glog.Errorln("An error occured while blpop on 'q'", err)
				}
				var l line.Line
				err = json.Unmarshal(val, &l)
				if err != nil && err.Error() == "unexpected end of JSON input" {
					glog.V(3).Infoln("blpop timeout", err)
				} else if err != nil {
					glog.Errorln("An error occured while unmarshalling the line")
				} else {
					glog.V(2).Infoln("line", l)
					lines <- l
				}
			}
		}
	}(lines, wg)
	return lines
}

func EsStore(l line.Line) {
	es := NewElasticsearch(esAddr, esIndex, esType)
	es.Index(&l)
}

func Ping(l line.Line) {
	if strings.Contains(l.Content, "ping") {
		glog.Infoln("Pong")
	}
}

func Debug(l line.Line) {
	glog.V(3).Infoln("[Debug] line.Command", l.Command, "line.Content", l.Content)
}

func main() {
	var wg sync.WaitGroup
	flag.Parse() // We need to capture arguments for glog
	glog.Infoln("Starting botbot-elasticsearch pointing to", esAddr)

	// Channels declaration
	sigs := make(chan os.Signal, 1)
	quit := make(chan struct{})

	// listen for syscall to cleanly terminate goroutine
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	plugins := []func(line.Line){EsStore, Ping, Debug}
	queue := common.NewRedisQueue()

	lines := listen(wg, queue, quit)
	condition := true
	for condition {
		select {
		case l := <-lines:
			for _, p := range plugins {
				p(l)
			}
		case <-sigs:
			glog.Infoln("Closting botbot-elasticsearch")
			close(quit)
			wg.Wait()
			condition = false
			break
		}
	}
}
