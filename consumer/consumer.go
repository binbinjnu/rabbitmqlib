package consumer

import (
	"encoding/json"
	"errors"
	"go.slotsdev.info/server-group/gamelib/log"
)

type Consumer struct {
	callback    func(interface{}) error
	connSession *ConnSession
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
		callback:    callback,
		connSession: NewConnSession(prefixName, addr, channelNum),
	}
	go GConsumer.connSession.handleConn()
	return nil
}

// CloseConsumer 关闭消费者
func CloseConsumer() {
	if GConsumer.connSession == nil {
		return
	}
	for _, v := range GConsumer.connSession.channelMap {
		v.closeChSession()
	}
	GConsumer.connSession.closeConnSession()
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
		log.Warn("json unmarshal err, data:", string(data))
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
