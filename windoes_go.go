package main

import "golang.org/x/sys/windows"

const (
	overlayWindowsTitle = "QQ桌面弹幕"
	wmHotKey            = 0x0312       //定义windows热键消息编号
	modAlt              = 0x0001       //定义Alt修饰键
	modControl          = 0x0002       //定义Ctrl键
	modNoRepeat         = 0x400        //定义按住热键时不重复触发的标志
	hotKeyID            = 1            //定义当前全局热键唯一的编号
	KeyD                = 0x44         //定义字母D的虚拟键码
	wsExLayered         = 0x00080000   //定义窗口分层扩展样式，保持窗口透明能力
	wsExTransparent     = 0x00000020   //定义窗口鼠标穿透扩展样式
	swpNopSize          = 0x0001       //定义窗口尺寸不变
	swpNoMove           = 0x0002       //定义保持窗口位置不变
	swpNoZOrder         = 0x0004       //定义保持窗口Z顺序不变
	swpNoActivate       = 0x0010       //定义修改窗口样式时不激活窗口
	swpFrameChanged     = 0x0020       //定义通知系统重新计算窗口边框
	gwlExStyle          = ^uintptr(19) //定义GWL_EXSTYLE 的数值，对应-20
)

var (
	overlayUser32        = windows.NewLazySystemDLL("user32.dll")       //加载Windows user 32.dll
	procFindWindowW      = overlayUser32.NewProc("FindWindowW")         //加载FindWindows函数
	procGetWindowLongPtr = overlayUser32.NewProc("GetWindowLongPtrW")   //加载GetWindowLongPtrW函数
	procSetWindowLongPtr = overlayUser32.NewProc("SetWindowLongPtrW")   //加载SetWindowLongPtrW函数
	procSetWindowPos     = overlayUser32.NewProc("SetWindowPos")        //加载SetWindowPos函数
	procRegisterHotKey   = overlayUser32.NewProc("RegisterHotKey")      //加载RegisterHotKey函数
	procUnregisterHotKey = overlayUser32.NewProc("UnregisterHotKey")    //加载UnregisterHotKey函数
	procGetMessage       = overlayUser32.NewProc(" GetMessageW")        //加载 GetMessageW函数
	procSetForegroundWnd = overlayUser32.NewProc("SetForegroundWindow") //加载SetForegroundWindow函数

)

type winPoint struct {
	x, y int32
}
type winMsg struct {
	Hwnd uintptr //保存接受消息的窗口标识符
}
