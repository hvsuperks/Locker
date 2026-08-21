package ui

import (
	"Locker/config"
	"Locker/config/key"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
)

var (
	colorMap = map[string]color.NRGBA{
		"OK":   (color.NRGBA{R: 0, G: 0, B: 40, A: 255}),
		"Fail": (color.NRGBA{R: 150, G: 0, B: 0, A: 255}),
		"SKIP": (color.NRGBA{R: 255, G: 143, B: 243, A: 255}),
	}
	itemID, itemLockerPGM, itemLockerCRC, itemPGM, itemCRC, itemMerge, itemVer, itemConnect fyne.CanvasObject
	chanLockerPGM, chanLockerCRC, chanPGM, chanCRC, chanMerge, chanConnect                  chan color.Color
	idBind, pgmLockerBind, crcLockerBind, pgmBind, crcBind, mergeBind, verBind, connectBind binding.String
)
var Ver string

func Manager(windows fyne.Window, resetPackChan chan string, ver, id, mode string, bind, color chan config.KV) *fyne.Container {
	Ver = ver
	i := Client(windows, resetPackChan)
	idBind.Set(id)
	verBind.Set(ver)
	go colerChange(color)
	go bindChange(bind)
	return i

}

func colerChange(color chan config.KV) {
	for {
		ch := <-color
		switch ch.Key {
		case key.ChanLockerPGM:
			chanLockerPGM <- colorMap[ch.Value]

		case key.ChanLockerCRC:
			chanLockerCRC <- colorMap[ch.Value]

		case key.ChanPGM:
			chanPGM <- colorMap[ch.Value]

		case key.ChanCRC:
			chanCRC <- colorMap[ch.Value]

		case key.ChanMerge:
			chanMerge <- colorMap[ch.Value]

		case key.ChanConnect:
			chanConnect <- colorMap[ch.Value]
		}
	}
}

func bindChange(bidn chan config.KV) {
	for {
		ch := <-bidn
		switch ch.Key {
		case key.PgmLockerBind:
			pgmLockerBind.Set(ch.Value)

		case key.CrcLockerBind:
			crcLockerBind.Set(ch.Value)

		case key.PgmBind:
			pgmBind.Set(ch.Value)

		case key.CrcBind:
			crcBind.Set(ch.Value)

		case key.ConnectBind:
			connectBind.Set(ch.Value)

		case key.MergeBind:
			mergeBind.Set(ch.Value)
		}
	}
}
