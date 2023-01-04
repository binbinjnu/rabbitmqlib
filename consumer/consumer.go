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
func NewConsumer(prefixName, addr string, channelNum int, callback func(interface{}) error) error {
	// 需要判断channelNum和queueVolume
	if channelNum < 1 {
		return errors.New("channel num require a minimum of 1")
	}
	if GConsumer != nil {
		return errors.New("consumer already start")
	}
	GConsumer = &Consumer{
		callback: callback,
	}
	Open(prefixName, addr, channelNum)
	return nil
}

// CloseConsumer 关闭消费者
func CloseConsumer() {
	if GConnSession == nil {
		return
	}
	for _, v := range GConnSession.channelMap {
		v.closeChSession()
	}
	GConnSession.closeConnSession()
	GConsumer = nil
	return
}

// 回调，处理数据
//
func callback(data []byte) (err error) {
	var v interface{} = nil

	// 延迟处理的函数
	defer func() {
		recoverErr := recover()
		if recoverErr != nil {
			log.Error("callback recover err: ", recoverErr)
			log.Warn("callback recover value: ", v)
			// todo 需要通知
			err = errors.New("callback err")
		}
	}()

	s := make([]interface{}, 0)
	err = json.Unmarshal(data, &s)
	if err != nil {
		return err
	}
	for _, v = range s {
		err = GConsumer.callback(v)
		if err != nil {
			return err
		}
	}
	return nil
}
