// 生产者
// 1. 接受外部调用发送过来的数据时, 累计n条或定时发送
// 2. 失败定时重发或累积m组数据没法同步到mq时则本地写文件
// 3. 定时从文件中获取并重发
// 4. 关闭时需要确保发送或本地写文件

package producer

import (
	"bufio"
	"encoding/json"
	"errors"
	"go.slotsdev.info/server-group/gamelib/log"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"
)

type (
	Producer struct {
		prefixName     string
		addr           string
		channelNum     int
		queueVolume    int
		connSession    *ConnSession
		loop           int               // 循环选择channel的计数
		dataCount      uint64            // 接受到的消息计数器, 可用于生成msgSt的id
		dataChan       chan interface{}  // 有缓冲队列	接收其它地方发送过来的数据(缓冲大小不能太大，确保不会导致生产者阻塞)
		dataBuffer     []interface{}     // interface{}的slice, 缓存其它地方发过来的数据, 定时聚合同步给queue
		toBeConfirmMap map[uint64][]byte // 存储已同步给queue的待确认的数据 key为msgSt的id, 值为json.Marshal(producer.dataBuffer)
		failBuffer     [][]byte          // 缓存queue确认失败或没法发送给queue的数据, 定时重发或写文件 json.Marshal(producer.dataBuffer)格式的切片
		respChan       chan *respSt      // mqchannel反馈
		fileIdSlice    []uint64          // 本地文件id的slice
		maxFileId      uint64            // 最大文件id
		isReady        bool
		done           chan bool
	}
)

const (
	// 文件相对路径
	fileDir = "./rabbitmq_log"
	// 文件名前缀
	fileNamePrefix = "log_"

	// 间隔一定时间发一次或者数量超过msgNumDelay发一次
	// 延迟发送时间间隔
	sendTick = 1 * time.Second
	// 延迟发送数量
	msgNumDelay = 10 // 10

	// 定时检查发送失败的数据, 进行重发或写文件
	resendOrDownTick = 2 * time.Second
	// buffer的数量限制, 即累积超n组数据没法发送给queue时则写文件
	dataJsonBufferLimit = 1000 // 1000

	// 定时检查文件的数据, 搞出来发送给queue
	checkFileTick = 1 * time.Second
)

var (
	GProducer *Producer
)

// NewProducer 新建一个生产者, channelNum最小为1, queueVolume最小为10
func NewProducer(prefixName, addr string, channelNum, queueVolume int) error {
	// 需要判断channelNum和queueVolume
	if channelNum < 1 {
		return errors.New("channel num require a minimum of 1")
	}
	if queueVolume < 20 {
		return errors.New("queue volume require a minimum of 20")
	}
	if GProducer != nil {
		return errors.New("producer already start")
	}
	GProducer = &Producer{
		prefixName:     prefixName,
		addr:           addr,
		channelNum:     channelNum,
		queueVolume:    queueVolume,
		loop:           0,
		dataCount:      0,
		dataChan:       make(chan interface{}, 1000),
		dataBuffer:     make([]interface{}, 0, msgNumDelay),
		toBeConfirmMap: make(map[uint64][]byte),
		failBuffer:     make([][]byte, 0, dataJsonBufferLimit),
		respChan:       make(chan *respSt, 2000),
		fileIdSlice:    make([]uint64, 0),
		maxFileId:      0,
		done:           make(chan bool),
	}
	err := GProducer.initLocalFile()
	if err != nil {
		// 初始化本地file不成功
		return err
	}
	// 开启协程
	go GProducer.handleProducer()
	return nil
}

// CloseProducer 关闭生产者
func CloseProducer() {
	if GProducer == nil {
		return
	}
	if GProducer.connSession == nil {
		return
	}
	for _, v := range GProducer.connSession.channelMap {
		v.closeChSession()
	}
	GProducer.connSession.closeConnSession()
	// 发消息给Producer, 处理手尾
	close(GProducer.done)
	for {
		if GProducer.isReady == false {
			log.Debug("producer closed")
			break
		}
	}
	GProducer = nil
	return
}

// SendMsg 发送数据, 最终结构需要跟业务方定
// 发送数据, 最终结构需要跟业务方定
func SendMsg(data interface{}) {
	GProducer.dataChan <- data
}

// channel返回消息给GProducer
func channelResp(id uint64, pushState int) {
	GProducer.respChan <- &respSt{id: id, pushState: pushState}
}

func (producer *Producer) initLocalFile() error {
	err := os.MkdirAll(fileDir, 0777)
	if err != nil {
		// todo 此处需要报错
		log.Error("mkdir err:", err)
		return err
	}
	fileInfoList, err := ioutil.ReadDir(fileDir)
	if err != nil {
		log.Error("read dir err:", err)
		return err
	}
	for _, v := range fileInfoList {
		// 会分割成 {"", "id值"}, 所以需要去index为1的
		splitSlice := strings.Split(v.Name(), fileNamePrefix)
		fileId, err := strconv.ParseUint(splitSlice[1], 10, 64)
		if err != nil {
			log.Error("file id err:", err)
			return err
		}
		producer.fileIdSlice = append(producer.fileIdSlice, fileId)
		if fileId > producer.maxFileId {
			producer.maxFileId = fileId
		}
	}
	return nil
}

func (producer *Producer) handleProducer() {
	producer.connSession = NewConnSession(producer.prefixName, producer.addr, producer.channelNum, producer.queueVolume)
	// 开启connSession协程
	go producer.connSession.handleConn()
	producer.isReady = true
	sendTicker := time.NewTicker(sendTick)
	resendOrDownTicker := time.NewTicker(resendOrDownTick)
	checkFileTicker := time.NewTicker(checkFileTick)
	for {
		select {
		case <-producer.done:
			log.Debug("done!")
			// 关闭, 需要把数据处理完
			// 1. 将producer.toBeConfirmMap中的数据同步到producer.failBuffer中
			for _, v := range producer.toBeConfirmMap {
				producer.failBuffer = append(producer.failBuffer, v)
			}
			// 清空producer.toBeConfirmMap
			producer.toBeConfirmMap = make(map[uint64][]byte)
			// 2. 将producer.dataBuffer中的数据同步到producer.failBuffer中
			if len(producer.dataBuffer) > 0 {
				log.Debug("delay send! buffer: ", producer.dataBuffer)
				dataJson, _ := json.Marshal(producer.dataBuffer)
				// todo 可以不用对json.Marshal的err处理, 能保证 producer.dataStrBuffer是slice
				producer.failBuffer = append(producer.failBuffer, dataJson)
				// 清空buffer
				producer.dataBuffer = make([]interface{}, 0, msgNumDelay)
			}
			// 3. 将producer.failBuffer中的数据落到本地文件中
			if len(producer.failBuffer) > 0 {
				producer.writeFile()
			}
			producer.isReady = false
			break

		case data := <-producer.dataChan:
			// 放到data缓存中
			producer.dataBuffer = append(producer.dataBuffer, data)
			if len(producer.dataBuffer) >= msgNumDelay {
				log.Debug("now send, buffer")
				producer.flushDataBuffer()
			}

		case resp := <-producer.respChan:
			log.Debug("resp is:", resp)
			if resp.pushState == DATA_PUSH_SUCCESS {
				// push成功,不做任何事情
			} else if resp.pushState == DATA_PUSH_ACK_SUCCESS {
				// 成功, 直接删除dataJsonMap中的数据
				delete(producer.toBeConfirmMap, resp.id)
			} else if resp.pushState == DATA_PUSH_FAIL ||
				resp.pushState == DATA_PUSH_ACK_FAIL ||
				resp.pushState == DATA_PUSH_CHAN_CLOSE ||
				resp.pushState == DATA_PUSH_SESSION_DONE {
				// 各种失败, 放到本地缓存中
				// 判断在dataJsonMap中是否存在, 存在则放到dataJsonBuffer中, 等待重发或写文件
				if dataJson, ok := producer.toBeConfirmMap[resp.id]; ok {
					// 存在
					producer.failBuffer = append(producer.failBuffer, dataJson)
					delete(producer.toBeConfirmMap, resp.id)
				}
			}

		case <-sendTicker.C:
			// 定时清空接收的dataStr缓存
			if len(producer.dataBuffer) > 0 {
				log.Debug("delay send! buffer")
				producer.flushDataBuffer()
			}

		case <-resendOrDownTicker.C:
			// 定时处理发送queue失败的数据
			if len(producer.failBuffer) > 0 {
				log.Debug("delay resend or down! buffer")
				producer.flushFailBuffer()
			}

		case <-checkFileTicker.C:
			// 定时处理本地的文件
			if len(producer.fileIdSlice) > 0 {
				producer.flushOneFile()
			}
		}
	}
}

// 每次只处理1个文件
func (producer *Producer) flushOneFile() {
	if producer.hasChannel() {
		fileName := fileDir + "/" + fileNamePrefix + strconv.FormatUint(producer.fileIdSlice[0], 10)
		producer.fileIdSlice = producer.fileIdSlice[1:]
		file, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE, 0666)
		if err != nil {
			log.Error("文件打开失败", err)
			return
		}
		reader := bufio.NewReader(file)
		for {
			// ReadLine有坑，默认最多读4096个byte
			//dataJson, _, err := reader.ReadLine()
			dataJson, err := reader.ReadBytes('\n')
			if err == io.EOF {
				break
			}
			if dataJson != nil && len(dataJson) > 0 {
				if dataJson[len(dataJson)-1] == '\n' {
					drop := 1
					if len(dataJson) > 1 && dataJson[len(dataJson)-2] == '\r' {
						drop = 2
					}
					dataJson = dataJson[:len(dataJson)-drop]
				}

				chS := producer.chooseChannel()
				if chS == nil {
					// 没有合适的channel, 放到本地缓存中
					producer.failBuffer = append(producer.failBuffer, dataJson)
				} else {
					// 有合适的channel, 发给channel
					producer.sendToQueue(dataJson, chS)
				}
			}
		}
		// 及时关闭file句柄
		file.Close()
		// 删除文件
		os.Remove(fileName)
	}
}

// 将producer.dataBuffer中的数据发送给队列或放到producer.confirmFailBuffer中
func (producer *Producer) flushDataBuffer() {
	dataJson, _ := json.Marshal(producer.dataBuffer)
	// todo 可以不用对json.Marshal的err处理, 能保证 producer.dataStrBuffer是slice
	chS := producer.chooseChannel()
	if chS == nil {
		// 没有合适的channel, 放到本地失败缓存中
		producer.failBuffer = append(producer.failBuffer, dataJson)
	} else {
		// 有合适的channel, 发给channel
		producer.sendToQueue(dataJson, chS)
	}
	// 清空buffer
	producer.dataBuffer = make([]interface{}, 0, msgNumDelay)
}

// 将producer.failBuffer中的数据重发或者写文件
func (producer *Producer) flushFailBuffer() {
	for {
		if len(producer.failBuffer) <= 0 {
			// 没有数据
			break
		}
		chS := producer.chooseChannel()
		if chS == nil {
			// 没有合适的chS, 直接全部写文件
			if len(producer.failBuffer) >= dataJsonBufferLimit {
				// 超出缓存上限, 全部写文件, 结束循环
				// 没超出的话, 就结束循环, 等待下一次flushDataJsonBuffer
				producer.writeFile()
			}
			break
		} else {
			// 有合适的channel, 发给channel
			producer.sendToQueue(producer.failBuffer[0], chS)
			producer.failBuffer = producer.failBuffer[1:]
			// 继续for循环
		}
	}
}

// 选择
func (producer *Producer) chooseChannel() *ChSession {
	if !GProducer.connSession.isReady {
		return nil
	}
	for i := 0; i < producer.channelNum; i++ {
		chS := GProducer.connSession.channelMap[producer.loop]
		// loop+1
		producer.loop = (producer.loop + 1) % producer.channelNum
		if !chS.isReady || chS.isThrottling {
			// 该chS不合适, 继续下一个
			continue
		}
		// 返回chS
		return chS
	}
	// 没找到合适的, 返回nil
	return nil
}

// 判断是否有合适的channel
func (producer *Producer) hasChannel() bool {
	if !GProducer.connSession.isReady {
		return false
	}
	for i := 0; i < producer.channelNum; i++ {
		chS := GProducer.connSession.channelMap[i]
		if chS == nil || !chS.isReady || chS.isThrottling {
			// 该chS不合适, 继续下一个
			continue
		}
		return true
	}
	// 没找到合适的
	return false
}

// 发送消息给队列
func (producer *Producer) sendToQueue(dataJson []byte, chS *ChSession) {
	producer.dataCount++
	msg := &msgSt{
		id:  producer.dataCount,
		msg: dataJson,
	}
	log.Debug("send msg， count is ", producer.dataCount)
	producer.toBeConfirmMap[msg.id] = dataJson
	chS.msgChan <- msg
}

// 写文件
func (producer *Producer) writeFile() {
	producer.maxFileId++
	fileName := fileDir + "/" + fileNamePrefix + strconv.FormatUint(producer.maxFileId, 10)
	file, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		producer.maxFileId--
		return
	}
	//及时关闭file句柄
	defer file.Close()
	writer := bufio.NewWriter(file)
	for _, v := range producer.failBuffer {
		writer.Write(v)
		writer.Write([]byte{'\n'})
	}
	//Flush将缓存的文件真正写入到文件中
	writer.Flush()
	// 增加文件id到slice中
	producer.fileIdSlice = append(producer.fileIdSlice, producer.maxFileId)
	// 清空dataJsonBuffer
	producer.failBuffer = make([][]byte, 0, dataJsonBufferLimit)
}
