package key

const (
	KeyPGM       = 1
	KeyCRC       = 2
	KeyLockerPGM = 3
	KeyLockerCRC = 4
	KeyMerge     = 5
	KeyConnect   = 6
	KeyReg       = 7
	KeyFile      = 8

	KeyPackCRC    = "a"
	KeyPackPGM    = "b"
	KeyPackMerge  = "c"
	KeyPackSocket = "d"

	ChanLockerPGM = "1"
	ChanLockerCRC = "2"
	ChanPGM       = "3"
	ChanCRC       = "4"
	ChanMerge     = "5"
	ChanConnect   = "6"

	PgmLockerBind = "2"
	CrcLockerBind = "3"
	PgmBind       = "4"
	CrcBind       = "5"
	MergeBind     = "6"
	ConnectBind   = "8"
)
