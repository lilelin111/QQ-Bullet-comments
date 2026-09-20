//go:build windows

package main

import (
	"context"
	"fmt"
	"runtime"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	overlayWindowsTitle = "QQ桌面弹幕"
	wmHotKey            = 0x0312       //定义windows热键消息编号
	modAlt              = 0x0001       //定义Alt修饰键
	modControl          = 0x0002       //定义Ctrl键
	modNoRepeat         = 0x4000       //定义按住热键时不重复触发的标志
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
	procFindWindow       = overlayUser32.NewProc("FindWindowW")         //加载FindWindows函数
	procGetWindowLongPtr = overlayUser32.NewProc("GetWindowLongPtrW")   //加载GetWindowLongPtrW函数
	procSetWindowLongPtr = overlayUser32.NewProc("SetWindowLongPtrW")   //加载SetWindowLongPtrW函数
	procSetWindowPos     = overlayUser32.NewProc("SetWindowPos")        //加载SetWindowPos函数
	procRegisterHotKey   = overlayUser32.NewProc("RegisterHotKey")      //加载RegisterHotKey函数
	procUnregisterHotKey = overlayUser32.NewProc("UnregisterHotKey")    //加载UnregisterHotKey函数
	procGetMessage       = overlayUser32.NewProc("GetMessageW")         //加载 GetMessageW函数
	procSetForegroundWnd = overlayUser32.NewProc("SetForegroundWindow") //加载SetForegroundWindow函数

)

type winPoint struct {
	x, y int32
}
type winMsg struct {
	Hwnd     uintptr  //保存接受消息的窗口标识符
	Message  uint32   //保存消息编号
	_        uint32   //填充 4 字节，保证64位结构体的对齐
	WParam   uintptr  //保存消息附加参数
	LParam   uintptr  //保存消息附加参数
	Time     uint32   //保存时间
	Pt       winPoint //保存鼠标坐标
	LPrivate uint32   //保存私有参数
}

func startOverlayIntegration(ctx context.Context) {
	go func() {
		hwnd := waitForOverlayWindows(ctx) //获取标识符
		if hwnd == 0 {
			fmt.Println("未找到窗口，鼠标穿透未启用！")
			return
		}
		runtime.LockOSThread()         //锁定为独立操作系统
		defer runtime.UnlockOSThread() //释放
		//注册 Ctrl，Alt，D，全局快捷键
		ok, _, callErr := procRegisterHotKey.Call(
			0, //进当前线程
			hotKeyID,
			modControl|modAlt|modNoRepeat, //禁止重复
			KeyD,
		)
		if ok == 0 {
			fmt.Println("全局快捷键注册失败！", winCallError(callErr))
			return
		}
		defer procUnregisterHotKey.Call(0, hotKeyID) //函数退出时注销快捷键
		err := setOverlayClickThrough(hwnd, true)    //启用鼠标穿透
		if err != nil {
			fmt.Println("启动鼠标穿透失败！")
			return
		}
		fmt.Println("鼠标穿透已经启用，按Ctrl+Alt+D切换模式")
		enabled := true //记录是否实在鼠标穿透状态
		var msg winMsg
		for {
			ret, _, callErr := procGetMessage.Call(
				// 传入消息结构体地址。
				uintptr(unsafe.Pointer(&msg)),
				// 接收所有窗口句柄的消息。
				0,
				// 不限制最小消息编号。
				0,
				// 不限制最大消息编号。
				0,
			)
			if int32(ret) == -1 {
				//输出消息循环错误
				fmt.Println("读取快捷消息失败!", winCallError(callErr))
				return
			}
			//结束消息循环
			if ret == 0 {
				return
			}
			if msg.Message != wmHotKey || msg.WParam != hotKeyID {
				//读下一条消息
				continue
			}
			enabled = !enabled
			err := setOverlayClickThrough(hwnd, enabled) //启用鼠标穿透
			if err != nil {
				fmt.Println("启动鼠标穿透失败！")
				continue
			}
			if enabled {
				fmt.Println("鼠标穿透已经打开！")
			} else {
				//可以操作面板
				fmt.Println("鼠标穿透已经关闭，可以操作面板！")
				procSetForegroundWnd.Call(hwnd) //弹幕窗口切换为前台
			}
		}
	}()
}

// uintptr 是 Go 语言中一个专门用于底层系统编程的特殊整数类型，
// 它的核心作用是安全地存储指针的内存地址值。
func waitForOverlayWindows(ctx context.Context) uintptr {
	title, _ := windows.UTF16PtrFromString(overlayWindowsTitle)
	//转化成 Windows UTF-8字符串
	ticker := time.NewTicker(100 * time.Millisecond)
	//100毫秒的计时器
	defer ticker.Stop()
	//函数结束释放计时器
	for {
		hwnd, _, _ := procFindWindow.Call( //找窗口标识符
			0,                              //不限制窗口类型
			uintptr(unsafe.Pointer(title))) //传入指针
		if hwnd != 0 {
			return hwnd //找到了
		}
		select { //检查上下文是否取消
		case <-ctx.Done(): //放弃等待直接返回
			return 0 //未找到窗口
		case <-ticker.C: //下一次触发
		}
	}
}
func setOverlayClickThrough(hwnd uintptr, enabled bool) error {
	style, _, callErr := procGetWindowLongPtr.Call(hwnd, gwlExStyle)
	//读取扩展模式
	if style == 0 && !winCallOk(callErr) {
		return winCallError(callErr)
	}
	newstyle := style
	//判断是否需要启动鼠标穿透
	if enabled {
		//保留分层样式并增加鼠标穿透样式
		newstyle |= wsExLayered | wsExTransparent
	} else {
		//移除鼠标穿透样式，保留分层
		newstyle &^= wsExTransparent
	}
	//窗口是否发生变化
	if newstyle != style {

		previous, _, callErr := procSetWindowLongPtr.Call(
			hwnd,       //标识符
			gwlExStyle, //修改扩展样式
			newstyle,
		) //传入新的
		//窗口设置是否成功
		if previous == 0 && !winCallOk(callErr) {
			return winCallError(callErr)
		}
	}
	ok, _, callErr := procSetWindowPos.Call(
		hwnd, //标识符
		0,    //窗口z
		0,    //x
		0,    //y
		0,    //原宽度
		0,    //原高度
		// 使用不移动、不缩放、不激活和刷新边框标志。
		swpNoMove|swpNopSize|swpNoZOrder|swpNoActivate|swpFrameChanged,
	)
	//刷新窗口是否成功
	if ok == 0 {
		return winCallError(callErr)
	}
	return nil
}
func winCallOk(err error) bool {
	//判断错误是否为空
	if err == nil { //空错误调用成功
		return true
	}
	//转换错误码
	errno, ok := err.(syscall.Errno)
	if ok && errno == 0 {
		return true
	}
	return false
}
func winCallError(err error) error {
	//错误是否表示调用成功
	if winCallOk(err) {
		return nil
	}
	return err
}
