//go:build !windows

// 仅允许该文件在非 Windows 平台参与编译。
package main

// 声明当前文件属于 main 包。
import "context"

// 导入上下文包，用于保持与 Windows 实现相同的方法签名。
func startOverlayIntegration(ctx context.Context) {
	// 非 Windows 平台不启用系统级鼠标穿透。
}

// 非 Windows 平台空实现函数结束。
