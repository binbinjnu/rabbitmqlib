package main

import (
	"fmt"
	"go.slotsdev.info/server-group/rabbitmqlib/consumer"
	"go.slotsdev.info/server-group/rabbitmqlib/example/util"
)

func main() {
	addr := "amqp://admin:123456@192.168.146.128:5672"
	consumer.NewConsumer("dm_queue_", addr, 2, CallbackShow)
	util.WaitClose()
}

// CallbackShow 回调函数
func CallbackShow(data interface{}) error {
	switch data.(type) {
	case map[string]interface{}:
		m, _ := data.(map[string]interface{})
		for k, v := range m {
			fmt.Println("key is ", k, " value is ", v)
		}
	}
	return nil
}
