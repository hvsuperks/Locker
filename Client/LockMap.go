package main

import (
	"fmt"
	"path/filepath"
	"runtime"
	"sync"
)

var isDebug = true

func LockMap(mu *sync.RWMutex) {
	if isDebug {
		_, file, line, _ := runtime.Caller(1)
		LogInfo(&Logger.manager, fmt.Sprintf(
			"LockMap %s:%d",
			filepath.Base(file),
			line,
		))
	}
	mu.Lock()
}

func UnlockMap(mu *sync.RWMutex) {
	if isDebug {
		_, file, line, _ := runtime.Caller(1)
		LogInfo(&Logger.manager, fmt.Sprintf(
			"UnlockMap %s:%d",
			filepath.Base(file),
			line,
		))
	}
	mu.Unlock()
}
