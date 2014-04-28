package tally

import (
	"github.com/garyburd/redigo/redis"
	"github.com/golang/glog"
)

type SubscriberSettings struct {
	RedisUrl       string
	RedisChannel   string
	MaxQueueLength int
}

type TallySubscriber interface {
	Channel() <-chan interface{}
	Start()
	Queue(queue string, inbound <-chan interface{})
	Stop()
	Close()
}

type tallySubscriberImpl struct {
	settings  SubscriberSettings
	queue     redis.Conn
	subscribe redis.Conn
	channel   chan interface{}
	stop      chan bool
}

func InitSubscriber(settings SubscriberSettings) (impl *tallySubscriberImpl, err error) {
	subscribe, err := redis.Dial("tcp", settings.RedisUrl)
	if err != nil {
		glog.Warningln("error-connect-redis-subscribe", settings, err)
		return
	}
	queue, err := redis.Dial("tcp", settings.RedisUrl)
	if err != nil {
		glog.Warningln("error-connect-redis-push-queue", settings, err)
		return
	}
	return &tallySubscriberImpl{
		settings:  settings,
		stop:      make(chan bool),
		queue:     queue,
		subscribe: subscribe,
	}, nil
}

func (this *tallySubscriberImpl) Channel() <-chan interface{} {
	if this.channel == nil {
		this.channel = make(chan interface{})
	}
	return this.channel
}

func (this *tallySubscriberImpl) Queue(queue string, inbound <-chan interface{}) {
	go func() {
		var queueLength int = 0
		var err error
		for {
			select {
			case message := <-inbound:
				if queueLength > this.settings.MaxQueueLength {
					glog.Warningln("queue-length-exceeds-limit", this.settings.MaxQueueLength, queueLength)
					// drop the message
					continue
				}
				queueLength, err = redis.Int(this.queue.Do("LPUSH", queue, message))
				if err != nil {
					glog.Warningln("error-lpush", queue, err, this.settings)
				}
			case stop := <-this.stop:
				if stop {
					return
				}
			}
		}
	}()
}

func (this *tallySubscriberImpl) Start() {
	psc := redis.PubSubConn{this.subscribe}
	err := psc.Subscribe(this.settings.RedisChannel)
	if err != nil {
		glog.Warningln("cannot-subscribe-channel", this.settings.RedisChannel, this.settings)
	}

	go func() {
		for {
			switch message := psc.Receive().(type) {
			case redis.Message:
				if this.channel != nil {
					this.channel <- message.Data
				}
			case redis.Subscription:
				glog.Infoln("subscription", message.Channel, "kind", message.Kind, "count", message.Count)
			case error:
				glog.Warningln("error-from-subscribed-channel", message)
				return
			}
		}
	}()
}

func (this *tallySubscriberImpl) Stop() {
	this.stop <- true
}

func (this *tallySubscriberImpl) Close() {
	this.subscribe.Close()
	this.queue.Close()
}
