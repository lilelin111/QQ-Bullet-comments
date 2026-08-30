package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// PowerShell 脚本：调用 Windows 通知中心的 UserNotificationListener，
// 拉取系统里的 Toast 通知，只输出 QQ/腾讯相关通知。
//
// 输出格式：通知ID<TAB>时间<TAB>标题/正文
// 用通知ID是为了让 Go 侧能做去重。
const queryScript = `
$listenerType = [Windows.UI.Notifications.Management.UserNotificationListener, Windows.UI.Notifications.Management, ContentType = WindowsRuntime]
$listener = [Windows.UI.Notifications.Management.UserNotificationListener]::Current

# 请求读取通知的权限；第一次运行可能需要在系统设置里允许
$status = $listener.RequestAccessAsync().GetAwaiter().GetResult()
if ($status -ne 'Allowed') {
    Write-Error '没有通知访问权限，请开启 Windows 通知历史记录并允许 QQ 通知'
    exit 1
}

# 获取当前通知中心里的全部 Toast 通知
$all = $listener.GetNotificationsAsync([Windows.UI.Notifications.NotificationKinds]::Toast).GetAwaiter().GetResult()

foreach ($n in $all) {
    # 取出发送通知的应用名，比如 "QQ" 或 "腾讯QQ"
    $app = ''
    if ($null -ne $n.AppInfo -and $null -ne $n.AppInfo.DisplayInfo) {
        $app = [string]$n.AppInfo.DisplayInfo.DisplayName
    }

    # 只保留 QQ 相关通知，其它应用的通知直接跳过
    if ($app -notmatch 'QQ|腾讯') { continue }

    # 遍历通知的视觉绑定，把标题、正文等文本元素全部收集起来
    $texts = @()
    if ($null -ne $n.Notification -and $null -ne $n.Notification.Visual) {
        foreach ($binding in $n.Notification.Visual.Bindings) {
            foreach ($t in $binding.GetTextElements()) {
                $texts += [string]$t.Text
            }
        }
    }

    # 用制表符（[char]9）分隔字段，方便 Go 解析；时间转成本地时间
    "{0}{1}{2:yyyy-MM-dd HH:mm:ss}{1}{3}" -f $n.Id, [char]9, $n.CreationTime.LocalDateTime, ($texts -join ' | ')
}
`

// queryQQNotifications 执行 PowerShell 脚本，并返回每一行结果。
// 每行格式：通知ID<TAB>时间<TAB>文本
func queryQQNotifications() ([]string, error) {
	// -NoProfile 避免加载用户配置导致启动变慢
	// -ExecutionPolicy Bypass 避免脚本执行策略拦截
	cmd := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", queryScript)

	// 把 PowerShell 的错误输出直接打到终端，方便排查
	cmd.Stderr = os.Stderr

	// 运行脚本并捕获标准输出
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("查询通知失败: %w", err)
	}

	// 按行拆分输出
	var lines []string
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	return lines, sc.Err()
}

func main() {
	// seen 记录已经打印过的通知ID，避免同一通知被重复输出
	seen := map[string]bool{}

	for {
		lines, err := queryQQNotifications()

		if err == nil {
			// 每行第一个字段是通知ID，用它去重
			for _, line := range lines {
				id, _, ok := strings.Cut(line, "\t")
				if ok && !seen[id] {
					seen[id] = true
					fmt.Println(line)
				}
			}
		} else {
			fmt.Fprintln(os.Stderr, err)
		}

		// 轮询间隔：3 秒拉一次通知中心
		time.Sleep(3 * time.Second)
	}
}
