package Get

import "sync"

var (
	oneBotMm      sync.Mutex                  //互斥锁
	oneBotRunning bool                        //记录OneBot是否监听
	oneBotQueue   = make(chan QQMessage, 256) //缓存消息
	oneBotErrors  = make(chan error, 1)       //缓存连接错误
)
