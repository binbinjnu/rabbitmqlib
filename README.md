# rabbitmq lib
基于AMQP实现的rabbitmq的生产者和消费者

## 生产者 package producer
```text
实现功能：
    - 单例模式
    - 单connection，多channel/queue
    - 支持rabbitmq的connection和channel重连
    - 支持定时检测队列消息数，如果超出预设值，则将该队列设置成限量状态，不再向其发送消息，直至降低至预设值/2
    - 消息支持缓存发送/定时发送
    - 失败重发，无法重发时写本地文件
    - 定时检查本地文件重发
    - 关闭时保证未及时同步的消息写本地文件
    
调用函数：
    producer.NewProducer    新建一个生产者
        - prefixName: 队列名前缀
        - addr：URI地址，例（amqp://admin:123456@192.168.146.128:5672）
        - channelNum：开启channel/queue的数量，最小值为1
        - queueVolume：队列容量限制预设值，最小为20
    producer.CloseProducer  关闭生产者
    producer.SendMsg    发送消息给mq
        - data：interface{}格式，一般为可json序列化的结构体（首字母大写）或 map[string]interface{}
```
## 消费者 package consumer
```text
实现功能：
    - 单例模式
    - 单connection，多channel/queue
    - 支持rabbitmq的connection和channel重连
    - 支持消息处理回调，会对回调进行recover处理
    - 设置最大未确认ack数，达到后将不再消费，需处理并重启
```
 