// rabbitmq的channel协程, 有conn启动, 可多个

package consumer

import (
	"github.com/streadway/amqp"
	"go.slotsdev.info/server-group/gamelib/log"
	"strconv"
	"time"
)

type (
	// channel的session
	ChSession struct {
		index           int
		name            string // 队列名(前缀+index组成的字符串)
		channel         *amqp.Channel
		delivery        <-chan amqp.Delivery
		notifyChanClose chan *amqp.Error
		done            chan bool
		isReady         bool
	}
)

const (

	// channel exception后重新init的时长
	reInitDelay = 5 * time.Second
)

func NewChSession(prefixName string, index int) *ChSession {
	return &ChSession{
		index: index,
		name:  prefixName + strconv.Itoa(index),
		done:  make(chan bool),
	}
}

func (chS *ChSession) closeChSession() {
	close(chS.done)
	if chS.channel != nil {
		chS.channel.Close()
	}
}

func (chS *ChSession) handleChannel(conn *amqp.Connection) {
FOR1:
	for {
		chS.isReady = false
		err := chS.init(conn)
		//err := errors.New("abc")
		log.Info("Init channel: ", chS.index)

		if err != nil {
			log.Warn("Failed to init channel. Retrying...")
			select {
			case <-chS.done:
				chS.isReady = false
				// 在init错误的时候收到done信息, 直接关掉整个协程
				break FOR1
			case <-time.After(reInitDelay):
				// init错误, 等待n秒继续init
			}
			continue FOR1
		}
	FOR2:
		for {
			select {
			case <-chS.done:
				chS.isReady = false
				log.Info("Done channel : ", chS.index)
				break FOR1

			case <-chS.notifyChanClose:
				// break FOR2, 重新跑FOR1循环
				log.Warn("Notify close channel : ", chS.index, " Rerunning init...")
				// 将chS.pushMap中的所有数据都返回
				break FOR2

			case d := <-chS.delivery:
				log.Debug("d:", string(d.Body))
				err := callback(d.Body)
				if err != nil {
					log.Error("callback err: ", err)
				} else {
					d.Ack(false)
				}
			}
		}
	}
}

// init will initialize channel & declare queue
// 初始化channel和声明queue
func (chS *ChSession) init(conn *amqp.Connection) error {
	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	_, err = ch.QueueDeclare(
		chS.name,
		true,  // Durable， 队列持久化
		false, // Delete when unused
		false, // Exclusive
		false, // No-wait
		nil,   // Arguments
	)
	if err != nil {
		return err
	}

	// 设置qos
	err = ch.Qos(3, 0, false)
	if err != nil {
		return err
	}

	delivery, err := ch.Consume(
		chS.name,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	chS.channel = ch
	chS.notifyChanClose = make(chan *amqp.Error, 10)
	chS.channel.NotifyClose(chS.notifyChanClose)
	chS.delivery = delivery

	chS.isReady = true
	log.Info("Channel setup success!")
	return nil
}
