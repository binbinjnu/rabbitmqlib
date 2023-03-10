// rabbitmq的channel(queue)协程, 有conn启动, 可多个
// 支持重连, 定时监测队列消息数量, 数量多时设置标记位

package producer

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
		volume          int    // 队列容量限制
		channel         *amqp.Channel
		notifyChanClose chan *amqp.Error
		notifyConfirm   chan amqp.Confirmation // 有缓冲队列, 需要足够生产者接受消息速度，暂定2000
		msgChan         chan *msgSt            // 有缓冲队列, 暂定2000, 足够一次性推2000条以内的日志过来
		pushCount       uint64                 // 推送计数器, 与amqp.Confirmation.DeliveryTag一致
		pushMap         map[uint64]uint64      // map[DeliveryTag]msg.id
		done            chan bool
		isThrottling    bool // 是否限流 (定时判断对应的mq的queue的消息数量, 超了则限流)
		isReady         bool
	}
)

const (

	// channel exception后重新init的时长
	reInitDelay = 5 * time.Second

	// 计算队列消息数时间间隔
	volumeCountTick = 5 * time.Second
)

func NewChSession(prefixName string, index, queueVolume int) *ChSession {
	return &ChSession{
		index:     index,
		name:      prefixName + strconv.Itoa(index),
		volume:    queueVolume,
		done:      make(chan bool),
		msgChan:   make(chan *msgSt, 2000),
		pushCount: 0,
		pushMap:   make(map[uint64]uint64),
	}
}

func (chS *ChSession) closeChSession() {
	if chS != nil {
		if chS.done != nil {
			close(chS.done)
		}
		for {
			if chS.isReady == false {
				log.Debug("chS ", chS.index, " closed")
				break
			}
		}
		if chS.channel != nil {
			chS.channel.Close()
		}
	}
}

// 判断是否限流
func (chS *ChSession) IsThrottling() bool {
	return chS.isThrottling
}

func (chS *ChSession) emptyPushMap(pushState int) {
	// 将chS.pushMap中的所有数据都返回
	for _, msgId := range chS.pushMap {
		channelResp(msgId, pushState)
	}
	chS.pushMap = make(map[uint64]uint64)
	chS.pushCount = 0
}

func (chS *ChSession) handleChannel(conn *amqp.Connection) {
	volumeCountTicker := time.NewTicker(volumeCountTick)
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
				log.Info("Done channel : ", chS.index)
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
				// 将chS.pushMap中的所有数据都返回
				chS.emptyPushMap(DATA_PUSH_SESSION_DONE)
				break FOR1

			case <-chS.notifyChanClose:
				// break FOR2, 重新跑FOR1循环
				log.Warn("Notify close channel : ", chS.index, " Rerunning init...")
				// 将chS.pushMap中的所有数据都返回
				chS.emptyPushMap(DATA_PUSH_CHAN_CLOSE)
				break FOR2

			case msg := <-chS.msgChan:
				//log.Debug("Channel: ", chS.index, " receive msg:", string(msg.msg))
				// 发送消息
				if !chS.isReady || chS.isThrottling { // 没准备好 或 限流
					// 继续FOR2循环
					//log.Println("Index: ", chS.index, "not ready or throttling")
					channelResp(msg.id, DATA_PUSH_FAIL)
					continue FOR2
				}
				// 处理error
				err := chS.channel.Publish(
					"",
					chS.name,
					false,
					false,
					amqp.Publishing{
						ContentType:  "text/plain",
						DeliveryMode: amqp.Persistent, // 消息持久化，就算重启也不会丢失
						Body:         msg.msg,
					},
				)
				if err != nil {
					log.Warn("Channel: ", chS.index, "publish err: ", err)
					channelResp(msg.id, DATA_PUSH_FAIL)
					continue FOR2
				}
				log.Debug("Channel: ", chS.index, "publish success")
				chS.pushCount++
				chS.pushMap[chS.pushCount] = msg.id
				channelResp(msg.id, DATA_PUSH_SUCCESS)

			case confirm := <-chS.notifyConfirm:
				log.Debug("index:", chS.index, "confirm:", confirm)
				msgId := chS.pushMap[confirm.DeliveryTag]
				// todo 在rabbitmq控制台手动删除queue,有大概率收到notifyConfirm信息
				// 		内容未amqp.Confirmation{DeliveryTag: 0, Ack: false}
				//		对此忽略DeliveryTag为0的信息
				//		后续查明原因
				if confirm.DeliveryTag != 0 && msgId != 0 {
					// 消息确认
					if confirm.Ack {
						channelResp(msgId, DATA_PUSH_ACK_SUCCESS)
					} else {
						channelResp(msgId, DATA_PUSH_ACK_FAIL)
					}
					delete(chS.pushMap, confirm.DeliveryTag)
				}

			case <-volumeCountTicker.C:
				// 继续handle的for循环
				queue, err := chS.channel.QueueInspect(chS.name)
				//err := errors.New("haha")
				if err != nil {
					// 继续FOR2循环
					continue FOR2
				}
				if queue.Messages >= chS.volume && chS.isThrottling == false {
					// 超出容量限制且未限流
					// todo 计算队列消息数, 如果超过, 设标记位, 一直到水位下降
					// todo 添加通知, 看是否使用回调函数
					log.Warn("Channel:", chS.index, " change to true message num is ", queue.Messages)
					chS.isThrottling = true
					continue FOR2
				} else if queue.Messages <= (chS.volume/2) && chS.isThrottling == true {
					// 低于容量且已限流
					log.Warn("Channel:", chS.index, " change to false message num is ", queue.Messages)
					chS.isThrottling = false
					continue FOR2
				}
			}
		}
	}
}

// init will initialize channel & declare queue
// 初始化channel和声明queue
// todo 是否使用exchange
func (chS *ChSession) init(conn *amqp.Connection) error {
	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	err = ch.Confirm(false)
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

	chS.channel = ch
	chS.notifyChanClose = make(chan *amqp.Error, 10)
	chS.notifyConfirm = make(chan amqp.Confirmation, 2000)
	chS.channel.NotifyClose(chS.notifyChanClose)
	chS.channel.NotifyPublish(chS.notifyConfirm)

	chS.isReady = true
	log.Info("Channel setup success!")
	return nil
}
