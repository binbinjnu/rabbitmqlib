// rabbitmq的链接, 单协程
// 支持重连, 开启channel(queue)的协程

package producer

import (
	"github.com/streadway/amqp"
	"log"
	"time"
)

type (
	// conn的session
	ConnSession struct {
		prefixName      string
		connection      *amqp.Connection
		channelNum      int
		channelMap      map[int]*ChSession
		notifyConnClose chan *amqp.Error
		done            chan bool
		isReady         bool
	}
)

var (
	GConnSession *ConnSession
)

const (
	// connection的重连时长
	reconnectDelay = 5 * time.Second
)

// 开启
func Open(prefixName, addr string, channelNum, queueVolume int) {
	GConnSession = &ConnSession{
		prefixName: prefixName,
		channelNum: channelNum,
		channelMap: make(map[int]*ChSession),
		done:       make(chan bool),
	}
	go GConnSession.handleConn(addr, queueVolume)
}

func (connS *ConnSession) closeConnSession() {
	close(connS.done)
	if connS.connection != nil {
		connS.connection.Close()
	}
}

func (connS *ConnSession) handleConn(addr string, queueVolume int) {
FOR1:
	for {
		connS.isReady = false
		log.Println("Attempting to connect")
		conn, err := connS.connect(addr)
		if err != nil {
			log.Println("Failed to connect. Retrying...")
			select {
			case <-connS.done:
				connS.isReady = false
				log.Println("Conn done in FOR1!")
				// 在conn错误的时候,收到done的信息, 直接关闭整个协程
				break FOR1
			case <-time.After(reconnectDelay):
				// conn错误, 等待n秒继续conn
			}
			continue FOR1
		}
		// conn正确
		// 重新分配channelMap
		connS.channelMap = make(map[int]*ChSession)
		// 建立n个channel并绑定
		for i := 0; i < connS.channelNum; i++ {
			chSession := NewChSession(connS.prefixName, i, queueVolume)
			go chSession.handleChannel(conn)
			connS.channelMap[i] = chSession
		}
		connS.isReady = true

	FOR2:
		for {
			select {
			case <-connS.done:
				connS.isReady = false
				log.Println("Conn done in FOR2!")
				// 运行过程中, 收到done消息, 直接关掉整个协程
				break FOR1
			case <-connS.notifyConnClose:
				// 运行过程中, 收到ConnClose的消息, 重新跑FOR1循环进行重连
				log.Println("Conn closed. Rerunning conn...")
				// 关掉旧的channel
				for _, v := range connS.channelMap {
					v.closeChSession()
				}
				break FOR2
			}
		}
	}
}

// connect will create a new AMQP connection
// 建立新的AMQP连接
func (connS *ConnSession) connect(addr string) (*amqp.Connection, error) {
	//return nil, errors.New("haha")
	conn, err := amqp.Dial(addr)
	if err != nil {
		return nil, err
	}

	connS.connection = conn
	connS.notifyConnClose = make(chan *amqp.Error)
	connS.connection.NotifyClose(connS.notifyConnClose)

	log.Println("Conn setup success!")

	return conn, nil
}
