package Get

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// 通过 Windows 通知中心的 UserNotificationListener 拉取通知，
// 输出格式：通知ID<TAB>时间<TAB>标题/正文，并只保留 QQ 相关通知。
const queryScript = `
$listenerType = [Windows.UI.Notifications.Management.UserNotificationListener, Windows.UI.Notifications.Management, ContentType = WindowsRuntime]
$listener = [Windows.UI.Notifications.Management.UserNotificationListener]::Current
$status = $listener.RequestAccessAsync().GetAwaiter().GetResult()
if ($status -ne 'Allowed') {
    Write-Error '没有通知访问权限，请开启 Windows 通知历史记录并允许 QQ 通知'
    exit 1
}

$all = $listener.GetNotificationsAsync([Windows.UI.Notifications.NotificationKinds]::Toast).GetAwaiter().GetResult()

foreach ($n in $all) {
    $app = ''
    if ($null -ne $n.AppInfo -and $null -ne $n.AppInfo.DisplayInfo) {
        $app = [string]$n.AppInfo.DisplayInfo.DisplayName
    }
    if ($app -notmatch 'QQ|腾讯') { continue }

    $texts = @()
    if ($null -ne $n.Notification -and $null -ne $n.Notification.Visual) {
        foreach ($binding in $n.Notification.Visual.Bindings) {
            foreach ($t in $binding.GetTextElements()) {
                $texts += [string]$t.Text
            }
        }
    }

    "{0}{1}{2:yyyy-MM-dd HH:mm:ss}{1}{3}" -f $n.Id, [char]9, $n.CreationTime.LocalDateTime, ($texts -join ' | ')
}
`

func queryQQNotifications() ([]string, error) {
	cmd := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", queryScript)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("查询通知失败: %w", err)
	}

	var lines []string
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	return lines, sc.Err()
}

func GetMessage() string {
	seen := map[string]bool{}
	for {
		lines, err := queryQQNotifications()
		if err != nil {
			// 如果查询出错，返回错误信息字符串（或者你也可以选择 return "" 忽略错误）
			return fmt.Sprintf("查询通知失败: %v", err)
		}

		// 如果查询成功，遍历所有行
		for _, line := range lines {
			id, _, ok := strings.Cut(line, "\t")
			if ok && !seen[id] {
				seen[id] = true
				// 找到新消息，立刻返回这个字符串，结束本次函数调用
				return line
			}
		}

		// 如果这一轮没有新消息，也没有报错，就等 3 秒后再查
		time.Sleep(3 * time.Second)
	}
}
