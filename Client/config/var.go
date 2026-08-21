package config

import "sync/atomic"

var (
	MasterAOI        = ""
	IP               atomic.Value
	IsDog            atomic.Bool
	DstAppName       = "Locker.exe"
	BackupRoot       = "D:\\Backup_MES"
	BackupRoot2      = "D:\\Backup_MES_2"
	OutputRoot       = "D:\\Merge_Data_lot"
	MesPath          = "D:\\MES"
	UploadRoot       = "D:\\upload"
	LotColumn        = "lot_no"
	User             = ""
	Pass             = ""
	MasterDir        = "C:\\Locked"
	XoisPath         = "C:\\xOIS"
	AutoUpdatePath   = "C:\\ProgramData\\Locker\\AutoUpdate.exe"
	RegPath          = "SOFTWARE\\Produc\\lock"
	RegMasterPath    = "SOFTWARE\\Produc\\lock\\master"
	RegCRCMasterPath = "SOFTWARE\\Produc\\lock\\master\\CRCMap"
	LogPath          = "D:\\log\\"
	AppRootDir       = "C:\\ProgramData\\Locker"
	UiPath           = "C:\\ProgramData\\Locker\\Ui.exe"
	MasterPort       = "50001"
	BackupMes        = "D:\\Master\\RawData"
	IsDebug          = false
	TypeMap          = map[string]string{
		"0": "Marking",
		"1": "AF",
		"2": "OIS",
		"3": "Tilt",
		"4": "Prism",
		"5": "Fra_AF",
		"6": "Fra_OIS",
		"7": "Flag",
		"9": "AOI",
	}
	RemoveLog bool
)
