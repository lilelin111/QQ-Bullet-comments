package Get

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
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

    $title = ''
    $body = ''
    if ($texts.Count -ge 1) {
        $title = $texts[0]
    }
    if ($texts.Count -ge 2) {
        $body = ($texts[1..($texts.Count-1)] -join ' ')
    } elseif ($texts.Count -eq 1) {
        $body = $texts[0]
    }

    "{0}{1}{2:yyyy-MM-dd HH:mm:ss}{1}{3}{1}{4}" -f $n.Id, [char]9, $n.CreationTime.LocalDateTime, $title, $body
}`

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

var (
	seenMu sync.Mutex              // seenMu：保护 seen 的并发读写
	seen   = make(map[string]bool) // seen：记录“已经处理过”的通知 ID
)

type QQMessage struct {
	NotificationID string `json:"notification_id"` //通知ID
	Time           string `json:"time"`            //通知时间
	Title          string `json:"title"`           //群名
	Body           string `json:"body"`            //通知内容
}

func parseLine(line string) (QQMessage, bool) {
	parts := strings.SplitN(line, "\t", 4)
	if len(parts) != 3 {
		return QQMessage{}, false
	}
	return QQMessage{
		NotificationID: parts[0],
		Time:           parts[1],
		Title:          parts[2],
		Body:           parts[3],
	}, true
}

func PrimeSeen() {
	lines, err := queryQQNotifications()
	if err != nil {
		return
	}
	seenMu.Lock()
	defer seenMu.Unlock()
	for _, line := range lines {
		if m, ok := parseLine(line); ok {
			seen[m.NotificationID] = true
		}
	}
}

func NextMessage(ctx context.Context, interval time.Duration) (QQMessage, error) { // NextMessage：持续查询，只返回真正的新通知
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		lines, err := queryQQNotifications()
		if err != nil {
			return QQMessage{}, fmt.Errorf("查询通知失败: %w", err) // 返回错误，不再把错误当弹幕内容
		}
		seenMu.Lock()
		for _, line := range lines {
			m, ok := parseLine(line)
			if !ok {
				continue
			}
			if !seen[m.NotificationID] {
				seen[m.NotificationID] = true
				seenMu.Unlock()
				return m, nil
			}
		}
		seenMu.Unlock()
		select {
		case <-ctx.Done():
			return QQMessage{}, ctx.Err()
		case <-ticker.C:
		}
	}
}
