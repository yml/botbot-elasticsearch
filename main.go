package main

import (
	"bytes"
	"flag"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strconv"
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

// Context require to Do the PluginResponse
type Context struct {
	Storage *Elasticsearch
	Queue   common.Queue
}

// PluginAction represents an action
type PluginAction int

// All this operation
const (
	NOOP PluginAction = iota
	STORE
	RESPOND
	REMEMBER
)

func (a PluginAction) String() string {
	switch a {
	case NOOP:
		return "noop"
	case STORE:
		return "Store"
	case RESPOND:
		return "Respond"
	case REMEMBER:
		return "Remember"
	default:
		return "Undefined"
	}
}

// Do implements all the PluginAction
func (a PluginAction) Do(b []byte, ctx *Context) error {
	switch a {
	case NOOP:
		return nil
	case STORE:
		client := &http.Client{}
		req, err := ctx.Storage.newRequest("POST", bytes.NewReader(b))
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if glog.V(3) {
			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			glog.Infoln(string(body))
		}
		return err
	case RESPOND:
		return ctx.Queue.Lpush("bot", b)
	default:
		return nil
	}
}

// PluginResponse is the type returned by a Plugin
type PluginResponse struct {
	Action PluginAction
	Body   []byte
}

func listen(wg sync.WaitGroup, ctx *Context, quit chan struct{}) chan []byte {
	lines := make(chan []byte)
	go func(lines chan []byte, wg sync.WaitGroup) {
		wg.Add(1)
		for {
			select {
			case <-quit:
				wg.Done()
				glog.V(2).Infoln("Quiting the listen goroutine because we received a quit chan")
				return
			default:
				// Blocking on lpop for 5 s
				_, val, err := ctx.Queue.Blpop([]string{"q"}, 5)
				if err != nil && err.Error() == "unexpected end of JSON input" {
					glog.V(3).Infoln("blpop timeout", err)
				} else if err != nil {
					glog.Errorln("An error occured while blpop on 'q'", err)
				} else if !bytes.Equal(val, []byte{}) {
					// Only send val if it is not empty
					lines <- val
				}
			}
		}
	}(lines, wg)
	return lines
}

// Plugin represents a plugin
type Plugin struct {
	Do func([]byte) *PluginResponse
}

// Run starts the plugin and returns the chan you can listen on to get response
func (p *Plugin) Run(out chan *PluginResponse, quit chan struct{}, wg sync.WaitGroup) chan []byte {
	in := make(chan []byte)
	go func(in chan []byte, out chan *PluginResponse, quit chan struct{}, wg sync.WaitGroup) {
		wg.Add(1)
		for {
			select {
			case b := <-in:
				out <- p.Do(b)
			case <-quit:
				wg.Done()
				return
			}
		}

	}(in, out, quit, wg)
	return in
}

// NewPlugin returns a pointer to a plugin
func NewPlugin(fn func([]byte) *PluginResponse) *Plugin {
	return &Plugin{Do: fn}
}

// EsStore is an Elastivsearch plugin func
func EsStore(b []byte) *PluginResponse {
	glog.V(3).Infoln("Elastic search plugin", string(b))
	if bytes.Contains(b, []byte(`,"Command":"PRIVMSG",`)) {
		return &PluginResponse{Action: STORE, Body: b}
	}
	return &PluginResponse{}
}

// Ping is a ping plugin
func Ping(b []byte) *PluginResponse {
	glog.V(3).Infoln("Ping plugin", string(b))
	if b != nil {
		l, err := line.NewFromJSON(b)
		if err != nil {
			glog.Errorln("An error occured while trying to Build the line from the []byte", err)
		}
		if strings.EqualFold(l.Content, l.BotNick+": ping") {
			return &PluginResponse{
				Action: RESPOND,
				Body:   []byte("WRITE " + strconv.Itoa(l.ChatBotId) + " " + l.Channel + " Are you in need of my services, " + l.User + " ?"),
			}
		}
	}
	return &PluginResponse{}
}

// Debug is a debug plugin
func Debug(b []byte) *PluginResponse {
	glog.V(2).Infoln("Debug plugin", string(b))
	return &PluginResponse{Action: NOOP, Body: b}
}

// DoPluginActions listen on the out chan for PluginResponse and perform this action
func DoPluginActions(out chan *PluginResponse, ctx *Context, quit chan struct{}, wg sync.WaitGroup) {
	wg.Add(1)
	for {
		select {
		case <-quit:
			wg.Done()
		case pr := <-out:
			err := pr.Action.Do(pr.Body, ctx)
			if err != nil {
				glog.Errorln("An error occured while performing this action", pr, err)
			}
		}
	}
}

func main() {
	var wg sync.WaitGroup
	flag.Parse() // We need to capture arguments for glog
	glog.Infoln("Starting botbot-elasticsearch pointing to", esAddr)

	// Channels declaration
	sigs := make(chan os.Signal, 1)
	quit := make(chan struct{})
	out := make(chan *PluginResponse)

	// listen for syscall to cleanly terminate goroutine
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	pluginInChan := []chan []byte{
		NewPlugin(EsStore).Run(out, quit, wg),
		NewPlugin(Ping).Run(out, quit, wg),
		NewPlugin(Debug).Run(out, quit, wg)}

	context := &Context{
		Storage: NewElasticsearch(esAddr, esIndex, esType),
		Queue:   common.NewRedisQueue(),
	}

	go DoPluginActions(out, context, quit, wg)

	lines := listen(wg, context, quit)
	conditionFlag := true
	for conditionFlag {
		select {
		case <-sigs:
			close(quit)
			conditionFlag = false
			glog.Infoln("Closing the plugin manager")
			wg.Wait()
		case l := <-lines:
			glog.V(3).Infoln("Line", string(l))
			for _, pic := range pluginInChan {
				pic <- l
			}
		}
	}
}
