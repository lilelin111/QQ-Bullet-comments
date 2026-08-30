package main

import (
	"context"
	"my-project/store"
	"sync"
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
func (a *App) CreateMessage(title string, u *store.User) map[string]interface{} {
	str, err := store.CreateMessages(title, u)
	if err != nil {
		return map[string]interface{}{"success": false, "message": err.Error()}
	}
	return map[string]interface{}{"success": true, "message": str}
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
