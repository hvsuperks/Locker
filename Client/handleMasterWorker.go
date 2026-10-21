package main

import (
	"Locker/config"
	"Locker/config/key"
	"Locker/script"
	"fmt"
	"strings"
	"time"
)

func handle_vnc() {
	Cmd(run, "taskkill", "/F", "/IM", "winvnc.exe")
	time.Sleep(time.Second)
	Cmd(start, "C:\\Program Files\\uvnc bvba\\UltraVNC\\winvnc.exe", "-install")
}

func handlelocker_state(mode string, msg Command) error {
	err := script.RegWrite(config.RegPath, mode, msg.Action)
	if err != nil {
		return fmt.Errorf("RegWrite Fail : %w", err)
	}
	switch mode {
	case "lockercrc":
		if strings.EqualFold(msg.Action, "Unlock") {
			STATUS_PUT(&STATUS.Locker_CRC, Locker_state{Value: "Unlock", Color: "Fail"})
		} else {
			STATUS_PUT(&STATUS.Locker_CRC, Locker_state{Value: "Locked", Color: "OK"})
			resetPackChan <- key.KeyPackCRC
		}

	case "lockerpgm":
		if strings.EqualFold(msg.Action, "Unlock") {
			STATUS_PUT(&STATUS.Locker_PGM, Locker_state{Value: "Unlock", Color: "Fail"})
		} else {
			resetPackChan <- key.KeyPackPGM
			STATUS_PUT(&STATUS.Locker_PGM, Locker_state{Value: "Locked", Color: "OK"})
		}
	}
	return nil
}

func handlePGMchange(pgm FileMap_struct) error {
	if pgm.PGM == "" || pgm.MD5 == "" || pgm.URL == "" {
		LogInfo(&Logger.connect, "ERROR:", "PGM msg Data nil : ")
		return fmt.Errorf("PGM Info Error : PGM %s, MD5 %s, URL %s", pgm.PGM, pgm.MD5, pgm.URL)
	}

	err := script.RegWrite(config.RegMasterPath, "PGM", pgm.PGM)
	if err != nil {
		return fmt.Errorf("Reg Write Error : %s %w", pgm.PGM, err)
	}
	err = script.RegWrite(config.RegMasterPath, "MD5", pgm.MD5)

	if err != nil {
		return fmt.Errorf("Reg Write Error : %s %w", pgm.MD5, err)
	}

	err = script.RegWrite(config.RegMasterPath, "URL", pgm.URL)
	if err != nil {
		return fmt.Errorf("Reg Write Error : %s %w", pgm.URL, err)
	}
	pgm.Color = "Fail"
	STATUS_PUT(&STATUS.MasterMap.PGM, pgm)
	resetPackChan <- key.KeyPackPGM
	return nil
}

func handleCRCchange(crc RegMap_struct) error {
	if crc.CRC == "" || crc.RegMap == nil {
		return fmt.Errorf("RegMap_struct Fail : CRC %s, RegMap %p", crc.CRC, crc.RegMap)
	}
	if err := script.RegDelete(config.RegCRCMasterPath); err != nil {
		LogInfo(&Logger.debug, "RegDelete", config.RegCRCMasterPath, err)
	}
	err := script.RegWrite(config.RegMasterPath, "CRC", crc.CRC)
	if err != nil {
		LogInfo(&Logger.debug, "RegWrite", config.RegCRCMasterPath, err)
		return fmt.Errorf("Reg Write Error : %s %w", crc.CRC, err)
	}
	err = script.SaveMapToReg(config.RegCRCMasterPath, crc.RegMap)
	if err != nil {
		LogInfo(&Logger.debug, "SaveMapToReg", crc.RegMap, err)
		return fmt.Errorf("SaveMapToReg Error : %s %w", crc.CRC, err)
	}
	crc.Color = "OK"
	crc.MD5 = ""
	if info, ok := crc.RegMap["info"]; ok {
		crc.MD5 = info["md5"]
	}
	STATUS_PUT(&STATUS.MasterMap.CRC, crc)
	resetPackChan <- key.KeyPackCRC
	return nil
}
