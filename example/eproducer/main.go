package main

import (
	"log"
	"rabbitmqlib/example/util"
	"rabbitmqlib/producer"
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

	addr := "amqp://admin:123456@192.168.146.128:5672"
	err := producer.NewProducer("dm_queue_", addr, 2, 100)
	if err != nil {
		log.Println("err:", err)
		return
	}

	log.Println("before time sleep")
	time.Sleep(10 * time.Second)
	log.Println("after time sleep")
	for i := 0; i < 10; i++ {
		producer.SendMsg(map[string]interface{}{"int": i, "string": "abc"})
		time.Sleep(3 * time.Second)
	}
	//mymq.Send([]byte("abc"))
	//mymq.Send([]byte("efg"))
	//mymq.Send([]byte("hij"))
	//mymq.Close()
	util.WaitClose()
	producer.CloseProducer()
	time.Sleep(5 * time.Second)

}
