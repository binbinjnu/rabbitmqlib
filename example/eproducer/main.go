package main

import (
	"go.slotsdev.info/server-group/gamelib/log"
	"go.slotsdev.info/server-group/rabbitmqlib/example/util"
	"go.slotsdev.info/server-group/rabbitmqlib/producer"
	"strconv"
	"time"
)

func main() {
	//if st == nil {
	//	fmt.Println("is nil ")
	//}
	//name := "duomi_queue"
	//addr := "amqp://admin:123456@192.168.146.128:5672"
	//mq := sconn.New(name, addr)
	//message := []byte("message")
	//time.Sleep(time.Second * 3)
	//mq.Push(message)

	addr := "amqp://dmsoft:dmsoft123456@192.168.99.105:5672"
	err := producer.NewProducer("dm_queue_", addr, 2, 100)
	if err != nil {
		log.Debug("err:", err)
		return
	}
	defer producer.CloseProducer()

	go send()

	util.WaitClose()
}

func send() {
	log.Debug("before time sleep")
	time.Sleep(5 * time.Second)
	log.Debug("after time sleep")
	for i := 0; i < 100; i++ {
		log.Debug("send msg i: ", i)
		producer.SendMsg(
			map[string]interface{}{
				"table":     "test_log",
				"id":        i,
				"string":    "index_" + strconv.Itoa(i),
				"timestamp": time.Now().Unix(),
				"datetime":  time.Now().Format("2006-01-02 15:04:05"),
			})
	}
}
