package store

import (
	"encoding/json"
	"fmt"
	"os"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

var UserFilePath = "store/users.json"
var RecordFilePath = "store/records.json"
var user []User
var user1 []Users
var records []Record

type Users struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Password string `json:"password"`
}
type Record struct {
	ID       int64   `json:"id"`
	Sort     string  `json:"sort"`
	Category string  `json:"category"`
	Amount   float64 `json:"amount"`
	Note     string  `json:"note"`
}
type User struct {
	Users  Users  `json:"users"`
	Record Record `json:"record"`
}

func init() {
	LoadUser()
	LoadRecords()
}

func LoadUser() {
	data, err := os.ReadFile(UserFilePath)
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Println("加载用户数据失败！", err)
		}
		return
	}
	err = json.Unmarshal(data, &user)
	if err != nil {
		fmt.Println("解析用户数据失败！", err)
	}
}
func SaveUser() error {
	data, err := json.MarshalIndent(user, "", "")
	if err != nil {
		fmt.Println("保存用户数据失败！", err)
	}
	return os.WriteFile(UserFilePath, data, 0644)

}
func LoadRecords() {
	data, err := os.ReadFile(RecordFilePath)
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Println("加载用户记录失败！", err)
		}
		return
	}
	err = json.Unmarshal(data, &user)
	if err != nil {
		fmt.Println("解析用户数据失败！", err)
	}
}
func SaveRecords() error {
	data, err := json.MarshalIndent(user, "", "")
	if err != nil {
		fmt.Println("保存用户记录失败！", err)
	}
	return os.WriteFile(RecordFilePath, data, 0644)
}
func FindName(Name string) User {
	for _, u := range user {
		if u.Users.Name == Name {
			return u
		}
	}
	return User{}
}

func CreatUsers(Name string, Password string) (*Users, error) {
	if Name == "" || Password == "" {
		fmt.Println("用户名和密码不能为空！")
		return nil, nil
	}
	foundUser := FindName(Name)
	if foundUser.Users.Name != "" {
		fmt.Println("用户名已经存在！")
		return nil, nil
	}
	if len(Password) < 8 {
		fmt.Println("密码必须长度必须大于8")
		return nil, nil
	}
	count := 0
	count1 := 0
	for _, i := range Password {
		if unicode.IsLetter(i) {
			count++
		} else if unicode.IsDigit(i) {
			count1++
		}
	}
	if count == 0 {
		fmt.Println("密码必须包含字母！")
		return nil, nil
	}
	if count1 == 0 {
		fmt.Println("密码必须包含数字！")
		return nil, nil
	}
	if (len(Password) - count - count1) == 0 {
		fmt.Println("密码必须包含特殊字符！")
		return nil, nil
	}
	newPassword, err := bcrypt.GenerateFromPassword([]byte(Password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Println("密码加密失败！")
		return nil, nil
	}
	newID := int64(1)
	user2 := &Users{
		ID:       newID,
		Name:     Name,
		Password: string(newPassword),
	}
	user1 = append(user1, *user2)
	user3 := &User{
		Users:  *user2,
		Record: Record{},
	}
	user = append(user, *user3)
	err = SaveUser()
	if err != nil {
		return nil, err
	}
	return user2, nil
}
func CheckUsers(Name string, Password string, u User) (*Users, error) {
	err := bcrypt.CompareHashAndPassword([]byte(u.Users.Password), []byte(Password))
	if err != nil {
		return nil, fmt.Errorf("密码错误！")
	}
	if Name != u.Users.Name {
		return nil, fmt.Errorf("用户名错误！")
	}
	return &u.Users, nil
}
func LoadServes(Name string, Password string, u User) (*Users, error) {
	User1, err := CheckUsers(Name, Password, u)
	if err != nil {
		return nil, err
	}
	return User1, nil
}

var IncomeCategories = []string{
	"工资", "奖金", "投资收益", "兼职", "其他收入",
}

var ExpenseCategories = []string{
	"餐饮", "交通", "购物", "娱乐", "医疗", "教育", "住房", "其他支出",
}

func BoolSort(a string) bool {
	if a == "Income" || a == "Outcome" {
		return true
	}
	return false
}
func Contains(list []string, category string) bool {
	for _, i := range list {
		if i == category {
			return true
		}
	}
	return false
}
func CreatRecords(sort string, category string, amount float64, note string) (Record, error) {
	if !BoolSort(sort) {
		return Record{}, fmt.Errorf("非正常选择类型！")
	}
	if amount < 0 {
		return Record{}, fmt.Errorf("输入金额必须大于0！")
	}
	if sort == "Income" {
		if !Contains(IncomeCategories, category) {
			return Record{}, fmt.Errorf("无效收支类型！")
		}
	} else if sort == "Outcome" {
		if !Contains(ExpenseCategories, category) {
			return Record{}, fmt.Errorf("无效收支类型！")
		}
	}
	newID := int64(1)
	if len(records) > 0 {
		newID = records[len(records)-1].ID
	}
	record := Record{
		ID:       newID,
		Sort:     sort,
		Category: category,
		Amount:   amount,
		Note:     note,
	}
	records = append(records, record)
	err := SaveRecords()
	if err != nil {
		return Record{}, err
	}
	return record, nil
}
func ShowRecord(Id int64) error {
	for i, _ := range records {
		if records[i].ID == Id {
			fmt.Println(records[i])
			continue
		}
	}
	return fmt.Errorf("查找记录不存在！")
}
func ShowRecord1(Category string) error {
	for i, _ := range records {
		if records[i].Category == Category {
			fmt.Println(records[i])
			continue
		}
	}
	return fmt.Errorf("查找记录不存在！")
}
func ShowRecord2(Note string) error {
	for i, _ := range records {
		if records[i].Note == Note {
			fmt.Println(records[i])
			continue
		}
	}
	return fmt.Errorf("查找记录不存在！")
}
func ShowRecord3(amount float64) error {
	for i, _ := range records {
		if records[i].Amount == amount {
			fmt.Println(records[i])
			continue
		}
	}
	return fmt.Errorf("查找记录不存在！")
}
func ShowRecord4(Sort string) error {
	for i, _ := range records {
		if records[i].Sort == Sort {
			fmt.Println(records[i])
			continue
		}
	}
	return fmt.Errorf("查找记录不存在！")
}
func ShowRecord5(users User) (Record, error) {
	for i, _ := range user {
		if users == user[i] {
			return users.Record, nil
		}
	}
	return Record{}, fmt.Errorf("没有对应用户！")
}
func ShowRecord6() (float64, float64, float64, error) {
	var Total float64
	income := 0.0
	outcome := 0.0
	for _, i := range user {
		if i.Record.Sort == "Outcome" {
			outcome += i.Record.Amount
			Total -= i.Record.Amount
		}
		if i.Record.Sort == "Income" {
			income += i.Record.Amount
			Total += i.Record.Amount
		}
	}
	return Total, income, outcome, nil
}
func DelectRecord(Id int64) ([]Record, error) {
	for i, _ := range user {
		if user[i].Record.ID == Id {
			Record = append(user[i].Record[:i])
		}
	}
}
