package Get

import (
	"context"       //取消监听，控制程序退出
	"encoding/json" //分析oneBot发来的JSON格式事件
	"net/http"      //给webSocket握手请求加鉴权头
	"os"            //读取环境变量配置
	"sync"          //互斥锁
	"time"          //时间，例如：查询时间循环
)

//oneBot是一个标准化的聊天软件接口协议

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

// 连接oneBot webSocket 服务
// webSocket是一种全双工通信协议，服务端主动推送消息
func connectOneBot(ctx context.Context) {
	endpoint := os.Getenv("QQ_DANMAKU_OUEBOT_WS") //读取webSocket地主之的环境变量
	if endpoint == "" {
		endpoint = "ws://127.0.0.1:3001/" //默认地址
	}
	header := http.Header{}                       //创建请求头
	token := os.Getenv("QQ_DANMAKU_OUEBOT_TOKEN") //访问令牌
	if token != "" {
		header.Set("Authorization", "Bearer "+token)
		//设置令牌
	}
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
