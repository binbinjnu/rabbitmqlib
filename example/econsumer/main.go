package main

import (
	"rabbitmqlib/consumer"
	"rabbitmqlib/example/util"
)

func main() {

	addr := "amqp://admin:123456@192.168.146.128:5672"
	consumer.Open("dm_queue_", addr, 2)

	util.WaitClose()

}
