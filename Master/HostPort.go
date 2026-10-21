package main

import (
	"fmt"
	"net/http"
)

func HostPort() {

	// 1. Giả lập Web Probe (trả về đúng chuỗi "Microsoft Connect Test")
	http.HandleFunc("/connecttest.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Microsoft Connect Test"))
		LogInfo(&Logger.Debug, "[NCSI Hit] Nhận được request kiểm tra Web từ Windows! connecttest.txt")
	})

	// (Tùy chọn) Bắt các đường dẫn khác nếu phần mềm khác gọi ngầm
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	http.HandleFunc("/ncsi.txt", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Microsoft NCSI")
		LogInfo(&Logger.Debug, "[NCSI Hit] Nhận được request kiểm tra Web từ Windows! ncsi.txt")
	})

	fmt.Println("Go NCSI Server đang chạy tại cổng 80...")
	// Windows NCSI mặc định gọi qua cổng HTTP (80)
	err := http.ListenAndServe(":80", nil)
	if err != nil {
		LogInfo(&Logger.Debug, "Lỗi khởi động server:", err)
	}
}
