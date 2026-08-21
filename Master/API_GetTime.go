package main

import (
	"fmt"
	"net/http"
	"time"
)

func GetTime(w http.ResponseWriter, r *http.Request) {
	now := time.Now().Unix()
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("%d", now)))
}
