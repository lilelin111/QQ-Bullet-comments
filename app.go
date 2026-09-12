package main

import (
	"context"
	"sync"
	"temp-project/Get"
	"temp-project/store"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx           context.Context
	mu            sync.Mutex
	monitorCancel context.CancelFunc
	monitorID     int
}

func (a *App) Register(username, password string) map[string]interface{} {
	user, err := store.CreateUser(username, password)
	if err != nil {
		return map[string]interface{}{"success": false, "message": err.Error()}
	}
	// 注册成功后启动 QQ 通知监听，确保监听只发生在用户登录之后。
	a.startQQMonitor()
	return map[string]interface{}{"success": true, "message": "注册成功!", "user": user}
}
func (a *App) Login(username, password string) map[string]interface{} {
	user, err := store.LoginService(username, password)
	if err != nil {
		return map[string]interface{}{"success": false, "message": err.Error()}
	}
	// 登录成功后启动 QQ 通知监听，未登录时不会查询系统通知。
	a.startQQMonitor()
	return map[string]interface{}{"success": true, "message": "登陆成功!", "user": user}
}
func (a *App) Logout() map[string]interface{} {
	// 退出登录时停止后台监听，避免继续读取和推送 QQ 消息。
	a.stopQQMonitor()
	return map[string]interface{}{"success": true, "message": "已退出登录"}
}
func (a *App) CreateMessage(u *store.User) map[string]interface{} {
	if u == nil || u.ID <= 0 {
		return map[string]interface{}{"success": false, "message": "用户无效，请先登录"}
	}
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()

	msg, err := store.CreateMessages(ctx, u)
	if err != nil {
		return map[string]interface{}{"success": false, "message": err.Error()}
	}
	return map[string]interface{}{
		"success":    true,
		"message":    "保存数据成功！",
		"message_id": msg.ID,
		"group_name": msg.Title,
	}
}
func (a *App) ShowGetMessage(UserID int64, Id int) map[string]interface{} {
	message, err := store.ShowGetMessage(UserID, Id)
	if err != nil {
		return map[string]interface{}{"success": false, "message": err.Error()}
	}
	return map[string]interface{}{"success": true, "message": message}
}
func (a *App) ShowGetTitle(UserID int64, Id int) map[string]interface{} {
	message, err := store.ShowGetTitle(UserID, Id)
	if err != nil {
		return map[string]interface{}{"success": false, "message": err.Error()}
	}
	return map[string]interface{}{"success": true, "message": message}
}
func (a *App) startQQMonitor(ctx context.Context) {
	go func() {
		Get.PrimeSeen()

		for {
			qq, err := Get.NextMessage(ctx, 3*time.Second)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				// 把监听错误推送给前端，避免后台一直重试但界面没有任何提示。
				runtime.EventsEmit(ctx, "qq:monitor-error", err.Error())
				time.Sleep(3 * time.Second)
				continue
			}
			runtime.EventsEmit(ctx, "qq:new-message", qq)
		}
	}()
}
