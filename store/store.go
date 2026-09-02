package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"my-project/Get"
	"os"
	"path/filepath"
	"unicode"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
	"golang.org/x/crypto/bcrypt"
)

var usersFilePath = "store/users.json"
var messageFilePath = "store/message.json"
var Users []User
var Messages []Message

type User struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Password string `json:"password"`
}
type Message struct {
	ID      int64  `json:"id"`
	UserId  int64  `json:"userid"`
	Title   string `json:"title"`
	Message string `json:"message"`
}

func init() {
	LoadUser()
	LoadMessage()
}
func LoadUser() {
	data, err := os.ReadFile(usersFilePath)
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Println("用户数据解析失败！")
		}
	}
	err = json.Unmarshal(data, &User{})
	if err != nil {
		fmt.Println("解析用户数据失败！")

	}
}
func SaveUser() error {
	data, err := json.MarshalIndent(User{}, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(usersFilePath, data, 0644)
}
func LoadMessage() error {
	data, err := os.ReadFile(messageFilePath)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	}
	err = json.Unmarshal(data, &Message{})
	if err != nil {
		return err
	}
	return nil
}
func SaveMessage() error {
	data, err := json.MarshalIndent(Message{}, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(messageFilePath, data, 0644)

}
func CreateUser(name string, password string) (*User, error) {
	if name == "" || password == "" {
		return nil, errors.New("用户名或密码不能为空！")
	}
	count := len(password)
	if count < 8 || count > 16 {
		return nil, errors.New("密码长度必须在8~16之间！")
	}
	count1 := 0
	count2 := 0
	for _, i := range password {
		if unicode.IsLower(i) {
			count1++
		}
		if unicode.IsUpper(i) {
			count2++
		}
	}
	if count1 == 0 || count2 == 0 {
		return nil, errors.New("密码里必须包含大小写字母！")
	}
	if (count - count1 - count2) == 0 {
		return nil, errors.New("密码里必须包含除了字母以外的其他符号，数字等等！")
	}
	newPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Println("密码加密失败")
		return nil, err
	}
	newID := int64(1)
	if len(Users) > 0 {
		newID = Users[len(Users)-1].ID + 1
	}
	user := &User{
		ID:       newID,
		Name:     name,
		Password: string(newPassword),
	}
	Users = append(Users, *user)
	err = SaveUser()
	if err != nil {
		return nil, errors.New("保存用户数据失败！")
	}
	return user, nil
}
func CheckUser(username string, password string, u *User) (*User, error) {
	if username == "" || password == "" {
		return nil, errors.New("请输入用户名或密码！")
	}
	err := bcrypt.CompareHashAndPassword([]byte(username), []byte(password))
	if err != nil {
		return nil, errors.New("密码解析错误！")
	}
	if username != u.Name {
		return nil, errors.New("用户名错误或该用户不存在！")
	}
	return u, nil
}
func FindUserName(name string) (*User, error) {
	for _, u := range Users {
		if u.Name == name {
			return &u, nil
		}
	}
	return nil, errors.New("用户不存在！")
}
func LoginService(username, password string) (*User, error) {
	u, err := FindUserName(username)
	if err != nil {
		return nil, err
	}
	u2, err2 := CheckUser(u.Name, password, u)
	if err2 != nil {
		return nil, err2
	}
	return u2, nil
}
func CreateMessages(title string, u *User) (string, error) {
	message := Get.GetMessage()
	newID := int64(1)
	if len(Users) > 0 {
		newID = Users[len(Users)-1].ID + 1
	}
	Message1 := &Message{
		ID:      newID,
		UserId:  u.ID,
		Title:   title,
		Message: message,
	}
	Messages = append(Messages, *Message1)
	err := SaveMessage()
	if err != nil {
		return fmt.Sprintf("保存数据失败！"), nil
	}
	return fmt.Sprintf("保存数据成功！"), nil
}
func ShowGetMessage(UserID int64, Id int) (string, error) {
	for _, i := range Messages {
		if i.UserId == UserID {
			if Id >= 0 && Id < len(Messages) {
				msg := fmt.Sprintf("%.5s", Messages[Id].Message)
				return msg, nil
			}
		}
	}
	return "", errors.New("未找到匹配的消息")
}
func ShowGetTitle(UserID int64, Id int) (string, error) {
	for _, i := range Messages {
		if i.UserId == UserID {
			if Id >= 0 && Id < len(Messages) {
				msg := fmt.Sprintf("%.3s", Messages[Id].Title)
				return msg, nil
			}
		}
	}
	return "", errors.New("未找到匹配的消息")
}
func CreateDesktopShortcut(s string) error {
	err := ole.CoInitialize(0)
	if err != nil {
		return fmt.Errorf("COM未初始化成功；%v", err)
	}
	defer ole.CoUninitialize()
	desktopPath, err := GetDesktopPath()
	if err != nil {
		return err
	}
	lnkName := "QQ弹幕.lnk"
	lnkPath := filepath.Join(desktopPath, lnkName)
	exeDir := filepath.Dir(s)
	iconFullPath := filepath.Join(exeDir, "app.ico")
	if _, err := os.Stat(lnkPath); os.IsNotExist(err) {
		iconFullPath = s + ",0"
	}
	WshShell, err := oleutil.CreateObject("Wcript.Shell")
	if err != nil {
		return err
	}
	defer WshShell.Release()
	dispatch, err := WshShell.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return err
	}
	defer dispatch.Release()
	rawObj, err := oleutil.CallMethod(dispatch, "CreateDesktopShortcut", lnkPath)
	if err != nil {
		return err
	}
	link := rawObj.ToIDispatch()
	defer link.Release()
	props := []struct {
		name  string
		value interface{}
	}{
		{"TargetPath", s},
		{"WorkingDirectory", exeDir},
		{"IconLocation", iconFullPath},
		{"Description", "QQ弹幕"},
	}
	for _, prop := range props {
		if _, err := link.PutProperty(prop.name, prop.value); err != nil {
			return fmt.Errorf("设置 %s 失败: %v", prop.name, err)
		}
	}
	_, err = oleutil.CallMethod(link, "Save")
	if err != nil {
		return fmt.Errorf("save shortcut failed : %v", err)
	}
	log.Panicln("桌面快捷方式已创建图标：", iconFullPath)
	return nil
}
func GetDesktopPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, "Desktop"), nil
}
