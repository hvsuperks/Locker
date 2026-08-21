package script

import (
	"Locker/config"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"time"
)

func Restart_XOIS(ctx context.Context, Restart_XOIS_chan chan struct{}) {
	defer func() {
		if err := recover(); err != nil {
			Fmt(fmt.Sprintf("panic: %v\n%s", err, debug.Stack()))
			msgbox("Restart_XOIS Fail")
			os.Exit(0)
		}
	}()

	// Khởi tạo một Timer đã dừng ngay từ đầu (hoặc thời gian rất dài)
	// Ở đây ta tạo một Timer đã hết hạn nhưng chủ động Stop nó ngay.
	timer := time.NewTimer(time.Hour)
	defer timer.Stop()
	if !timer.Stop() {
		<-timer.C
	}

	for {
		select {
		case <-ctx.Done():
			return

		case <-Restart_XOIS_chan:
			// 1. Dừng timer cũ nếu nó đang chạy
			if !timer.Stop() {
				select {
				case <-timer.C: // Xả channel nếu timer đã kịp nổ
				default:
				}
				// 2. Tắt ứng dụng
				exec.Command("taskkill", "/IM", "CubeManager.exe", "/T", "/F").Run()
			}

			// 3. Đặt lịch 2 giây sau thì bật lại
			timer.Reset(5 * time.Second)

		case <-timer.C:
			// 4. Đến giờ "bình minh", bật ứng dụng lên
			exec.Command("explorer.exe", filepath.Join(config.XoisPath, "CubeManager.exe")).Run()
		}
	}
}
