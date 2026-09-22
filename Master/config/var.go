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
		`0`: `Marking`,
		`1`: `AF`,
		`2`: `OIS`,
		`3`: `Tilt`,
		`4`: `Prism`,
		`5`: `Fra_AF`,
		`6`: `Fra_OIS`,
		`7`: `Flag`,
		`9`: `AOI`,
	}
	Ver = `9.4.1`
)
