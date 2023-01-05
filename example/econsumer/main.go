package main

import (
	"fmt"
	"go.slotsdev.info/server-group/gamelib/log"
	"go.slotsdev.info/server-group/rabbitmqlib/consumer"
	"go.slotsdev.info/server-group/rabbitmqlib/example/util"
	"reflect"
	"strings"
)

type st struct {
}

var valueOfSt = reflect.ValueOf(&st{})

func main() {
	addr := "amqp://dmsoft:dmsoft123456@192.168.99.105:5672"
	consumer.NewConsumer("dm_queue_", addr, 2, Control)
	util.WaitClose()
}

func Control(data interface{}) error {
	switch data.(type) {
	case map[string]interface{}:
		m, _ := data.(map[string]interface{})
		log.Debug("m table is ", m["table"])
		log.Debug("m reflect value is ", reflect.ValueOf(m))
		f := valueOfSt.MethodByName(case2Camel(m["table"].(string)))
		f.Call([]reflect.Value{reflect.ValueOf(m)})
	}
	return nil
}

func case2Camel(name string) string {
	name = strings.Replace(name, "_", " ", -1)
	name = strings.Title(name)
	return strings.Replace(name, " ", "", -1)
}

func (st *st) TestLog(data map[string]interface{}) error {
	log.Debug("data:", data)
	return nil
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
