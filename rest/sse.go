package rest

import (
	"fmt"
	"github.com/golang/glog"
	"net/http"
	"sync"
)

func (this *engine) StreamServerEvents(resp http.ResponseWriter, req *http.Request,
	contentType, eventType, key string, source <-chan interface{}) error {

	sc, new := this.getSseChannel(contentType, eventType, key)
	if new {
		go func() {
			// connect the source
			for {
				if m, open := <-source; !open {
					glog.V(100).Infoln("Closing channel:", sc.Key)
					sc.Stop()
					this.deleteSseChannel(key)
					return
				} else {
					sc.messages <- m
				}
			}
		}()
	}
	sc.ServeHTTP(resp, req)
	return nil
}

func (this *engine) Stop() {
	for _, s := range this.sseChannels {
		s.stop <- 1
	}
}

func (this *engine) deleteSseChannel(key string) {
	this.lock.Lock()
	defer this.lock.Unlock()
	delete(this.sseChannels, key)
	glog.V(100).Infoln("Engine deleted channel", key, "c=", len(this.sseChannels))
}

func (this *engine) getSseChannel(contentType, eventType, key string) (*sseChannel, bool) {
	this.lock.Lock()
	defer this.lock.Unlock()

	if c, has := this.sseChannels[key]; has {
		return c, false
	} else {
		c = new(sseChannel).Init()
		c.engine = this
		c.ContentType = contentType
		c.EventType = eventType
		this.sseChannels[key] = c
		c.Start()
		return c, true
	}
}

type event_client chan interface{}

type sseChannel struct {
	Key string

	ContentType string
	EventType   string

	engine *engine
	lock   sync.Mutex

	// Send to this to stop
	stop chan int

	clients map[event_client]int

	// Channel into which new clients can be pushed
	newClients chan event_client

	// Channel into which disconnected clients should be pushed
	defunctClients chan event_client

	// Channel into which messages are pushed to be broadcast out
	// to attahed clients.
	messages chan interface{}
}

func (this *sseChannel) Init() *sseChannel {
	this.stop = make(chan int)
	this.clients = make(map[event_client]int)
	this.newClients = make(chan event_client)
	this.defunctClients = make(chan event_client)
	this.messages = make(chan interface{})
	return this
}

func (this *sseChannel) Stop() {
	glog.V(100).Infoln("Stopping channel", this.Key)

	if this.stop == nil {
		glog.V(100).Infoln("Stopped.")
		return
	}

	this.lock.Lock()
	defer this.lock.Unlock()

	glog.V(100).Infoln("Sending stop")
	this.stop <- 1
	close(this.stop)
	this.stop = nil

	glog.V(100).Infoln("Closing messages")
	close(this.messages)
	// // stop all clients
	// for c, _ := range this.clients {
	// 	glog.V(100).Infoln("Closing event client channels")
	// 	c <- nil
	// 	close(c)
	// }

	// TODO - send via channel to engine?
	this.engine.deleteSseChannel(this.Key)
}

func (this *sseChannel) Start() *sseChannel {
	go func() {
		for {
			select {

			case s := <-this.newClients:
				this.lock.Lock()
				this.clients[s] = 1
				this.lock.Unlock()
				glog.V(100).Infoln("Added new client:", s)

			case s := <-this.defunctClients:
				this.lock.Lock()
				delete(this.clients, s)
				this.lock.Unlock()
				close(s)
				glog.V(100).Infoln("Removed client:", s)

			case <-this.stop:
				this.Stop()
				return
			default:
				msg, open := <-this.messages
				if !open || msg == nil {
					for s, _ := range this.clients {
						glog.V(100).Infoln("Stopping client", s)
						s <- nil
					}
					glog.V(100).Infoln("Stopping channel", this.Key)
					this.Stop()
					return
				}
				// There is a new message to send.  For each
				// attached client, push the new message
				// into the client's message channel.
				for s, _ := range this.clients {
					s <- msg
				}
			}
		}
	}()
	return this
}

// TODO - return and disconnect client
func (this *sseChannel) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	// Make sure that the writer supports flushing.
	f, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	// Create a new channel, over which the broker can
	// send this client messages.
	messageChan := make(event_client)

	// Add this client to the map of those that should
	// receive updates
	this.newClients <- messageChan

	// Listen to the closing of the http connection via the CloseNotifier
	notify := w.(http.CloseNotifier).CloseNotify()
	go func() {
		<-notify
		// Remove this client from the map of attached clients
		// when `EventHandler` exits.
		this.defunctClients <- messageChan
		glog.V(100).Infoln("HTTP connection just closed.")
	}()

	// Set the headers related to event streaming.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	for {

		// Read from our messageChan.
		msg, open := <-messageChan

		if !open || msg == nil {
			// If our messageChan was closed, this means that the client has
			// disconnected.
			glog.V(100).Infoln("Messages stopped.. Closing http connection")
			break
		}

		switch this.ContentType {
		case "application/json":
			fmt.Fprintf(w, "event: %s\n", this.EventType)
			fmt.Fprint(w, "data: ")
			json_marshaler(this.ContentType, w, &msg, no_header)
			fmt.Fprint(w, "\n\n")
		case "text/plain":
			fmt.Fprintf(w, "%s\n", msg)
		default:
			if m, ok := marshalers[this.ContentType]; ok {
				fmt.Fprintf(w, "event: %s\n", this.EventType)
				fmt.Fprint(w, "data: ")
				m(this.ContentType, w, &msg, no_header)
				fmt.Fprint(w, "\n\n")
			}
		}

		// Flush the response.  This is only possible if
		// the repsonse supports streaming.
		f.Flush()
	}

	// Done.
	glog.V(100).Infoln("Finished HTTP request at ", r.URL.Path, "num_channels=", len(this.engine.sseChannels))
}
