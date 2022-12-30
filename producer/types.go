package producer

const (
	// 消息推送的状态
	DATA_PUSH_FAIL         = iota // 推送失败
	DATA_PUSH_SUCCESS             // 推送成功
	DATA_PUSH_ACK_FAIL            // 推送确认失败
	DATA_PUSH_ACK_SUCCESS         // 推送确认成功
	DATA_PUSH_CHAN_CLOSE          // 收到chS.notifyChanClose
	DATA_PUSH_SESSION_DONE        // 收到chS.done
)

type (
	msgSt struct {
		id       uint64
		msg      []byte
		respChan chan *respSt // mqchannel反馈
	}

	respSt struct {
		id        uint64
		pushState int   // 推送状态
		err       error // todo 可以加上错误信息
	}
)
