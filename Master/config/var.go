package config

import (
	"sync/atomic"
)

var (
	MasterLocker     = atomic.Value{}
	MasterAOI        = atomic.Value{}
	MasterUILocker   = atomic.Value{}
	MasterUIAoi      = atomic.Value{}
	MasterService    = atomic.Value{}
	MasterVbatoolVer = atomic.Value{}
	IP               atomic.Value
	IsDog            atomic.Bool
	//Master.exe
	DstAppName = `Master.exe`

	//D:\Master\PGM
	HttpMasterDir = `D:\Master\PGM`

	//D:\Master\Backup
	BackupPGM = `D:\Master\Backup`

	//D:\Master
	HttpDir = `D:\Master`

	//D:\Master\Upload
	HttpUpload = `D:\Master\Upload`

	//`D:\Master\web`
	WebPathDir = `D:\Master\web`

	//D:\Backup_MES
	BackupRoot = `D:\Backup_MES`

	//D:\Backup_MES_2
	BackupRoot2 = `D:\Backup_MES_2`

	//D:\Merge_Data_lot
	OutputRoot = `D:\Merge_Data_lot`

	//D:\MES
	MesPath = `D:\MES`

	//D:\upload
	UploadRoot = `D:\upload`

	//lot_no
	LotColumn     = `lot_no`
	User          = ``
	Pass          = ``
	NatLoginState = atomic.Bool{}

	//C:\Locked
	MasterDir = `C:\Locked`

	//C:\xOIS
	XoisPath = `C:\xOIS`

	//SOFTWARE\Produc\lock
	RegPath = `SOFTWARE\Produc\lock`

	//D:\Master\log\
	LogPath_master = `D:\Master\log\`

	//D:\log\
	LogPath = `D:\log\`

	//C:\ProgramData\Locker
	AppRootDir = `C:\ProgramData\Locker`

	//C:\ProgramData\Locker\watcherdog.exe
	AppDogPath = `C:\ProgramData\Locker\watcherdog.exe`

	//50001
	MasterPort = `50001`

	//D:\Master\RawData
	BackupRawData = `D:\Master\RawData`

	//D:\CSV_tmp\
	CsvTMPPath = `D:\CSV_tmp\`
	IsDebug    = false
	StartTime  = ``
	TypeMap    = map[string]string{
		`00`: `Aging`,
		`10`: `AF`,
		`20`: `OIS`,
		`30`: `Tilt`,
		`31`: `Tilt`,
		`32`: `Tilt-Ro`,
		`33`: `Tilt-Ro`,
		`40`: `Prism`,
		`50`: `Fra-AF`,
		`60`: `Fra-OIS`,
		`70`: `Flag`,
		`90`: `AOI`,
	}
	Ver = `9.5.3`
)
