package main

import (
	"github.com/isayme/go-amqp-reconnect/rabbitmq"
	"github.com/op/go-logging"
	"github.com/streadway/amqp"
)

//const logFormat = `%{time: Jan/02 15:04:05} %{level:.1s} %{shortfile} %{shortfunc}> %{message}`
const logFormat = `%{time: Jan/02 15:04:05} %{level:.1s} %{shortfile} > %{message}`

var (
	cfg        *Config
	log        *logging.Logger
	shaper     *Shaper
	fw         *Firewall
	ppp        *PPPoE
	billing    *Billing
	mqMessages <-chan amqp.Delivery
)

type (
	Task struct {
		Cmd  string `json:"cmd"`
		User string `json:"user"`
		Ips  string `json:"ips"`
		TlId int    `json:"tlid"`
		City string `json:"city"`
	}
)

func eh(err error, msg ...string) {
	if err != nil {
		log.Error(err, msg)
		panic(err)
	}
}

func ehSkip(err error, msg ...string) {
	if err != nil {
		log.Error(err, msg)
	}
}

func RunAtStart() {
	//shaper.UpdateCache()
	//fw.UpdateCache()
	eh(HandleTask(&Task{Cmd: "sync_all"}))
	return
}

func main() {
	mqConn, err := rabbitmq.Dial(cfg.MQ.Url)
	eh(err, "Failed to connect to RabbitMQ")
	defer func() { ehSkip(mqConn.Close()) }()

	var ch *rabbitmq.Channel

	ch, err = mqConn.Channel()
	eh(err, "Failed to open a channel")
	defer func() { ehSkip(ch.Close()) }()

	shaper = NewShaper()
	defer func() { shaper.Disconnect() }()

	fw = NewFirewall()
	defer func() { fw.Disconnect() }()

	ppp = NewPPPoE()
	defer func() { ppp.Disconnect() }()

	billing = NewBilling()
	defer func() { billing.Disconnect() }()

	eh(ch.Qos(1, 0, false), "Failed to set QoS")
	mqMessages, err = ch.Consume(
		cfg.MQ.Queue,
		"mk-daemon",
		false,
		true,
		false,
		true,
		nil,
	)
	eh(err, "Failed to register a consumer")

	RunAtStart()
	log.Info(" [*] Waiting for tasks.")

	forever := make(chan bool)
	go MQmessagesHandler()
	<-forever
}
