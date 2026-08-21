package ui

import (
	"Locker/config"
	"Locker/config/key"
	"Locker/script"
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// Cấu trúc quản lý bộ đếm click ẩn app
type ClickTracker struct {
	count     int
	lastClick time.Time
	mode      string
	value     binding.String
	color     chan color.Color
}

func (ct *ClickTracker) OnTap(w fyne.Window, resetPackChan chan string) {
	now := time.Now()
	// Quá 2 giây không bấm tiếp -> reset đếm lại từ đầu
	if now.Sub(ct.lastClick) > 2*time.Second {
		ct.count = 0
	}
	ct.count++
	ct.lastClick = now

	if ct.count >= 5 {
		ct.count = 0 // Reset bộ đếm
		switch ct.mode {
		case "show":
			script.RegWrite(config.RegPath, "show", "hide")
			w.Hide()

		case "locker":
			l := script.RegRead(config.RegPath, "lockercrc")
			if l == "Locked" {
				script.RegWrite(config.RegPath, "lockercrc", "Unlock")
				ct.value.Set("Unlock")
				ct.color <- colorMap["Fail"]
			} else {
				script.RegWrite(config.RegPath, "lockercrc", "Locked")
				ct.value.Set("Locked")
				ct.color <- colorMap["OK"]
			}

			resetPackChan <- key.KeyPackCRC
		}
	}
}

// 1. Định nghĩa một Widget tàng hình tùy biến chỉ nhận sự kiện Click
type InvisibleTapZone struct {
	widget.BaseWidget
	OnTap func() // Hàm sẽ chạy khi được click
}

// Hàm khởi tạo cho Widget tàng hình
func NewInvisibleTapZone(onTap func()) *InvisibleTapZone {
	zone := &InvisibleTapZone{OnTap: onTap}
	zone.ExtendBaseWidget(zone) // Bắt buộc phải có trong Fyne Custom Widget
	return zone
}

// Ghi đè phương thức Tapped để hứng sự kiện Click chuột
func (z *InvisibleTapZone) Tapped(_ *fyne.PointEvent) {
	if z.OnTap != nil {
		z.OnTap()
	}
}

// Vẽ giao diện cho widget: Trả về một hình chữ nhật trong suốt hoàn toàn
func (z *InvisibleTapZone) CreateRenderer() fyne.WidgetRenderer {
	rect := canvas.NewRectangle(color.Transparent)
	return widget.NewSimpleRenderer(rect)
}

func Client(windows fyne.Window, resetPackChan chan string) *fyne.Container {
	// 1. Khai báo các biến Binding
	idBind = binding.NewString()
	pgmLockerBind = binding.NewString()
	crcLockerBind = binding.NewString()
	pgmBind = binding.NewString()
	crcBind = binding.NewString()
	mergeBind = binding.NewString()
	verBind = binding.NewString()
	connectBind = binding.NewString()
	// 3. Tạo các item và lấy channel điều khiển màu
	itemID, _ := createItem("ID:", idBind)
	tracker1 := ClickTracker{mode: "show"}
	invisibleButton := NewInvisibleTapZone(func() {
		tracker1.OnTap(windows, resetPackChan)
	})
	itemIDs := container.NewStack(
		itemID,          // Nằm dưới (hiển thị cho người ta xem)
		invisibleButton, // Nằm trên (tàng hình, làm nhiệm vụ hứng click chuột)
	)
	itemLockerPGM, chanLockerPGM = createItem("PGM Locker:", pgmLockerBind)
	itemLockerCRC, chanLockerCRC = createItem("CRC Locker:", crcLockerBind)
	tracker2 := ClickTracker{mode: "locker", value: crcLockerBind, color: chanLockerCRC}
	lock := NewInvisibleTapZone(func() {
		tracker2.OnTap(windows, resetPackChan)
	})
	itemLockerCRCs := container.NewStack(
		itemLockerCRC,
		lock,
	)
	itemPGM, chanPGM = createItem("PGM:", pgmBind)
	itemCRC, chanCRC = createItem("CRC:", crcBind)
	itemMerge, chanMerge = createItem("Merge:", mergeBind)
	itemConnect, chanConnect = createItem("", connectBind)
	itemVer, _ = createItem("Ver:", verBind)
	// 4. Dựng Layout
	con1 := container.NewHBox(
		layout.NewSpacer(),
		itemIDs,
		NewVLine(),
		itemLockerPGM,
		NewVLine(),
		itemLockerCRCs,
		NewVLine(),
		itemPGM,
		NewVLine(),
		itemCRC,
		NewVLine(),
		itemMerge,
		NewVLine(),
		itemConnect,
		NewVLine(),
		itemVer,
		layout.NewSpacer(),
	)
	backgroundColor := colorMap["OK"]
	background := canvas.NewRectangle(backgroundColor)
	background.SetMinSize(fyne.NewSize(1200, 30))
	ui := container.NewStack(background, container.NewCenter(con1))
	return ui

}

func NewVLine() fyne.CanvasObject {
	// Tạo đường kẻ dọc màu trắng, độ trong suốt 50% (Alpha 150)
	line := canvas.NewRectangle(color.NRGBA{R: 0, G: 0, B: 40, A: 255})

	// Ép chiều rộng là 1px, chiều cao tự dãn theo Container (25px)
	line.SetMinSize(fyne.NewSize(4, 25))

	return line
}

func NewHLine() fyne.CanvasObject {
	// Tạo đường kẻ dọc màu trắng, độ trong suốt 50% (Alpha 150)
	line := canvas.NewRectangle(color.NRGBA{R: 0, G: 0, B: 40, A: 255})

	// Ép chiều rộng là 1px, chiều cao tự dãn theo Container (25px)
	line.SetMinSize(fyne.NewSize(1200, 1))

	return line
}

func createItem(label string, data binding.String) (fyne.CanvasObject, chan color.Color) {
	// Nền của Label tiêu đề
	bg := canvas.NewRectangle(colorMap["Fail"]) // Màu mặc định xám
	if label == "ID:" || label == "Ver:" {
		bg = canvas.NewRectangle(colorMap["OK"]) // Màu mặc định xám
	}
	bg.CornerRadius = 3

	title := canvas.NewText(" "+label+" ", color.White)
	title.Alignment = fyne.TextAlignCenter
	title.TextSize = 14 // Chỉnh size vừa vặn với chiều cao 25px
	title.TextStyle = fyne.TextStyle{Bold: true}

	// Stack để bọc title vào trong bg
	value := NewBindingText(data)
	wd := container.NewHBox(title, value)
	colorChan := make(chan color.Color, 50)
	go func(label string, colorChan chan color.Color, bg *canvas.Rectangle) {
		for c := range colorChan {
			newColor := c
			fyne.Do(func() {
				bg.FillColor = newColor
				bg.Refresh()
			})
		}

	}(label, colorChan, bg)

	// Trả về HBox chứa (Nền+Tiêu đề) và (Giá trị)
	return container.NewStack(bg, container.NewCenter(wd)), colorChan
}

func NewBindingText(data binding.String) fyne.CanvasObject {
	// 1. Khởi tạo đối tượng canvas.Text (giống như bạn muốn)
	valText := canvas.NewText("", color.White)
	valText.TextSize = 14 // Size 12 là đẹp nhất cho thanh 25px
	valText.TextStyle = fyne.TextStyle{Bold: true}
	valText.Alignment = fyne.TextAlignCenter // Căn giữa

	// 2. Lắng nghe thay đổi từ binding string
	data.AddListener(binding.NewDataListener(func() {
		str, _ := data.Get()
		valText.Text = " " + str + " "
		valText.Refresh() // Bắt buộc phải Refresh để vẽ lại chữ mới
	}))

	return valText
}
