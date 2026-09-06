package main

import (
	"context"
	"sync"
	"temp-project/store"
	"time"
)

type App struct {
	ctx context.Context
	mu  sync.Mutex
}

func (a *App) Register(username, password string) map[string]interface{} {
	user, err := store.CreateUser(username, password)
	if err != nil {
		return map[string]interface{}{"success": false, "message": err.Error()}
	}
	return map[string]interface{}{"success": true, "message": "注册成功!", "user": user}
}
func (a *App) Login(username, password string) map[string]interface{} {
	user, err := store.LoginService(username, password)
	if err != nil {
		return map[string]interface{}{"success": false, "message": err.Error()}
	}
	return map[string]interface{}{"success": true, "message": "登陆成功!", "user": user}
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
