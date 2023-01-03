package consumer

import (
	"encoding/json"
	"errors"
	"go.slotsdev.info/server-group/gamelib/log"
)

type Consumer struct {
	callback func(interface{}) error
}

var (
	GConsumer *Consumer
)

// NewConsumer 新建一个消费者
func NewConsumer(prefixName, addr string, channelNum int, callback func(interface{}) error) {
	GConsumer = &Consumer{
		callback: callback,
	}
	Open(prefixName, addr, channelNum)
}

// 回调，处理数据
//
func callback(data []byte) (err error) {

	// 延迟处理的函数
	defer func() {
		recoverErr := recover()
		if recoverErr != nil {
			log.Error("callback recover err: ", recoverErr)
			// todo 需要通知
			err = errors.New("callback err")
		}
	}()

	s := make([]interface{}, 0)
	err = json.Unmarshal(data, &s)
	if err != nil {
		return err
	}
	for _, v := range s {
		err = GConsumer.callback(v)
		if err != nil {
			return err
		}
	}
	return nil
}
