package Get

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

var (
	oneBotMu      sync.Mutex                  //互斥锁
	oneBotRunning bool                        //记录OneBot是否监听
	oneBotQueue   = make(chan QQMessage, 256) //缓存消息
	oneBotErrors  = make(chan error, 1)       //缓存连接错误
)

type oneBotEvent struct {
	postType    string          `json:"post_type"`
	MessageType string          `json:"message_type"`
	MessageID   int64           `json:"message_id"`
	UserID      int64           `json:"user_id"`
	GroupID     int64           `json:"group_id"`
	Message     json.RawMessage `json:"message"` //消息内容，可能是字符串或消息段数组
	//本质是一个字节切片
	RawMessage string `json:"raw_message"` //原始文本
	Time       int64  `json:"time"`        //Unix时间戳
	Sender     struct {
		Nickname string `json:"nickname"`
		Card     string `json:"card"`
	} `json:"sender"`
}

func startOneBot(ctx context.Context) {
	oneBotMu.Lock() //启动监听
	if oneBotRunning {
		oneBotMu.Unlock()
		return
	}
	oneBotRunning = true //设置锁
	oneBotMu.Unlock()    //释放
	go func() {
		defer func() {
			oneBotMu.Lock()
			oneBotRunning = false //设置状态
			oneBotMu.Unlock()
		}()
	}()
	for {
		err := connectOneBot(ctx) //连接oneBot并堵塞运行
		if ctx.Err() != nil {
			return //上下文取消时退出
		}
		if err != nil {
			select {
			case oneBotErrors <- err:
			default:
			}
		}
		time.Sleep(3 * time.Second) //3秒刷新
	}
}
func NextMessages(ctx context.Context, interval time.Duration) ([]QQMessage, error) {
	startOneBot(ctx)

}
