package main

import (
	"Locker/config"
	"Locker/config/key"
	"Locker/script"
	"context"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"time"
)

func Start(ctx context.Context, cancel context.CancelFunc) {
	defer RecoverLog()
	reconnectLock.Store(false)
	fmt.Println("Start")
	go func() {
		timer := time.NewTicker(5 * time.Second)
		for {
			<-timer.C
			os.Mkdir("D:\\MES", os.ModePerm)
			os.Mkdir("D:\\Backup_MES", os.ModePerm)
		}
	}()

	// PGM LoadFi
	p := FileMap_struct{
		PGM:   script.RegRead(config.RegMasterPath, "PGM"),
		MD5:   script.RegRead(config.RegMasterPath, "MD5"),
		URL:   script.RegRead(config.RegMasterPath, "URL"),
		Color: "Fail",
	}
	STATUS_PUT(&STATUS.MasterMap.PGM, p)

	var c RegMap_struct
	// CRC Load
	if m, err := script.LoadDeepReg(config.RegCRCMasterPath); err == nil {
		md5CRC := ""
		if info, ok := m["info"]; ok {
			md5CRC = info["md5"]
		}
		c = RegMap_struct{
			CRC:    script.RegRead(config.RegMasterPath, "CRC"),
			RegMap: m,
			Color:  "Fail",
			MD5:    md5CRC,
		}

	} else {
		LogInfo(&Logger.debug, "CRC Load", err)
		c = RegMap_struct{
			CRC:    "Fail",
			RegMap: nil,
			Color:  "Fail",
		}
	}

	STATUS_PUT(&STATUS.MasterMap.CRC, c)

	// Locker Load

	pgmLock := script.RegRead(config.RegPath, "lockerpgm")
	if strings.EqualFold(pgmLock, "Locked") {
		STATUS_PUT(&STATUS.Locker_PGM, Locker_state{Value: "Locked", Color: "OK"})
	} else {
		STATUS_PUT(&STATUS.Locker_PGM, Locker_state{Value: "Unlock", Color: "Fail"})
	}
	//script.Setting_Load()
	defer close(resetXOISChan)
	go script.Restart_XOIS(ctx, resetXOISChan)

	// Tạo kênh Reset các pack
	go Packege_Manager(ctx, cancel, resetPackChan, resetXOISChan)
	<-ctx.Done()
}

func Packege_Manager(ctx context.Context, cancel context.CancelFunc, resetPackChan chan string, resetXOISChan chan struct{}) {
	defer RecoverLog()
	// Tạo các biến kiểm tra hoạt động của app
	var (
		cancelCrc, cancelPgm, cancelMerge context.CancelFunc
		ctxCrc, ctxPgm, ctxMerge          context.Context
		LockerCRC_chan                    = make(chan struct{}, 10)
		LockerPGM_chan                    = make(chan struct{}, 10)
	)
	var (
		crcRuning    atomic.Bool
		pgmRuning    atomic.Bool
		mergeRuning  atomic.Bool
		socketRuning atomic.Bool
	)
	// Tạo kênh kiểm tra các app còn sống hay ko
	packHeart := time.NewTicker(3 * time.Second)
	defer packHeart.Stop()

	go func() {
		for range packHeart.C { // Phải có for để chạy lặp lại
			if !crcRuning.Load() {
				resetPackChan <- key.KeyPackCRC
			}
			if !pgmRuning.Load() {
				resetPackChan <- key.KeyPackPGM
			}
			if !mergeRuning.Load() {
				resetPackChan <- key.KeyPackMerge
			}
			if !socketRuning.Load() {
				resetPackChan <- key.KeyPackSocket
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case pack := <-resetPackChan:
			if isdownload.Load() && pack == key.KeyPackPGM {
				continue
			}
			switch pack {
			case key.KeyPackCRC:
				if cancelCrc != nil {
					cancelCrc()
				}
				ctxCrc, cancelCrc = context.WithCancel(ctx)

				a := mClient{
					ctx:    ctxCrc,
					cancel: cancelCrc,
					Log:    Log,
				}
				if crcRuning.CompareAndSwap(false, true) {
					go a.crcManager(&crcRuning, LockerCRC_chan, resetXOISChan)
				}
			case key.KeyPackPGM:
				if cancelPgm != nil {
					cancelPgm()
				}
				ctxPgm, cancelPgm = context.WithCancel(ctx)

				a := mClient{
					ctx:    ctxPgm,
					cancel: cancelPgm,

					Log: Log,
				}
				if pgmRuning.CompareAndSwap(false, true) {
					go a.pgmManager(&pgmRuning, LockerPGM_chan, resetXOISChan)
				}
			case key.KeyPackMerge:
				if cancelMerge != nil {
					cancelMerge()
				}
				ctxMerge, cancelMerge = context.WithCancel(ctx)

				a := mClient{
					ctx:    ctxMerge,
					cancel: cancelMerge,

					Log: Log,
				}
				LogInfo(&Logger.merge, "INFO:", "Merge Start")
				go a.mergeManager(&mergeRuning)

			default:
			}

		}
	}
}
