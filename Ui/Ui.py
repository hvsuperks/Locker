import os
import subprocess
import time,time
import shutil
from datetime import datetime
from PySide6.QtCore import Qt, Signal
from PySide6.QtWidgets import (
    QApplication,
    QWidget,
    QHBoxLayout,
    QVBoxLayout,
    QLabel,
    QMenu,
    QSystemTrayIcon
)
from PySide6.QtGui import QAction, QIcon

STYLE_PASS = """
background-color: rgb(0, 0, 40);
color:white;
font-weight:bold;
padding-left:5px;
padding-right:5px;
"""

STYLE_FAIL = """
background-color: rgb(150, 0, 0);
color:white;
font-weight:bold;
padding-left:5px;
padding-right:5px;
"""

class ClickLabel(QLabel):
    clicked = Signal()
    def __init__(self, text=""):
        super().__init__(text)
        self.count = 0
        self.last_click = 0

    def mousePressEvent(self, event):
        now = time.time()

        if now - self.last_click > 1:
            self.count = 0

        self.count += 1
        self.last_click = now

        if self.count >= 5:
            self.count = 0
            self.clicked.emit()

        super().mousePressEvent(event)
  
class MainWindow(QWidget):
    def __init__(self,VER,state,uploadRoot):
        super().__init__()
        self.uploadRoot = uploadRoot
        self.state = state
        self.thisVer = VER
        self.state.signal_bus.update_signal.connect(self.set_status)
        self.labels = {}
        self.init_ui()
        self.setFixedWidth(1200)
        self.setPost()
        
        
        
    def init_ui(self):

        self.setWindowFlags(
            Qt.WindowType.FramelessWindowHint
            | Qt.WindowType.WindowStaysOnTopHint
            | Qt.WindowType.Tool
        )

        self.setFixedHeight(25)

        self.setStyleSheet("""
            QWidget{
                background-color: rgb(0,0,40);
                font-size:12px;
            }
        """)


        # Tray
        self.tray = QSystemTrayIcon(
            QIcon(self.state.icoPath),
            self
        )
        self.setWindowIcon(QIcon(self.state.icoPath))
        menu = QMenu()
        act_show = QAction("Show", self)
        act_hide = QAction("Hide", self)
        act_exit = QAction("Exit", self)

        act_show.triggered.connect(self.show)
        act_hide.triggered.connect(self.hide)
        act_exit.triggered.connect(QApplication.quit)

        menu.addAction(act_show)
        menu.addAction(act_hide)
        menu.addSeparator()

        self.tray.setContextMenu(menu)
        self.tray.activated.connect(self.tray_click)
        self.tray.show()

        root = QVBoxLayout(self)
        root.setContentsMargins(0, 0, 0, 0)

        header = QWidget()

        bar = QHBoxLayout(header)
        bar.setContentsMargins(2, 2, 2, 2)
        bar.setSpacing(2)
        
        self.lbl_id = ClickLabel()
        self.lbl_crcLocker = ClickLabel()
        self.lbl_pgmLocker = ClickLabel()
        self.lbl_pgm = ClickLabel()
        self.lbl_crc = ClickLabel()
        self.lbl_merge = ClickLabel()
        self.lbl_ver = ClickLabel()
        self.lbl_verService = ClickLabel()
        self.lbl_verUi = ClickLabel()
        self.lbl_connect = ClickLabel()
        self.lbl_msg = ClickLabel()
        self.lbl_msg.setFixedWidth(200)
        self.lbl_ver.clicked.connect(self.on_ver_click)
        self.lbl_verUi.clicked.connect(self.on_ver_click)
        self.lbl_verService.clicked.connect(self.on_ver_click)
        self.lbl_crcLocker.clicked.connect(self.on_crclocker_click)
        self.lbl_pgmLocker.clicked.connect(self.on_pgmlocker_click)
        self.lbl_pgm.clicked.connect(self.on_pgm_click)
        self.lbl_crc.clicked.connect(self.on_crc_click)
        self.lbl_merge.clicked.connect(self.on_merge_click)
        self.lbl_msg.clicked.connect(self.on_msg_click)
        self.lbl_connect.clicked.connect(self.on_connect_click)
        self.lbl_id.clicked.connect(self.on_id_click)
        self.count = 0
        self.labels = {
            "ID": self.lbl_id,
            "PGM": self.lbl_pgm,
            "CRC": self.lbl_crc,
            "CRC_LOCKER": self.lbl_crcLocker,
            "PGM_LOCKER": self.lbl_pgmLocker,
            "CONNECT": self.lbl_connect,
            "VERSION": self.lbl_ver,
            "MERGE": self.lbl_merge,
            "MSG" : self.lbl_msg ,
            "VERSERVICE": self.lbl_verService
        }
        bar.addStretch()
        bar.addWidget(self.lbl_id)
        bar.addWidget(self.lbl_pgmLocker)
        bar.addWidget(self.lbl_crcLocker)
        bar.addWidget(self.lbl_pgm)
        bar.addWidget(self.lbl_crc)
        bar.addWidget(self.lbl_merge)
        bar.addWidget(self.lbl_connect)
        bar.addStretch()
        bar.addWidget(self.lbl_verService)
        bar.addWidget(self.lbl_ver)
        bar.addWidget(self.lbl_verUi)
        bar.addWidget(self.lbl_msg)
        root.addWidget(header)
        self.setAcceptDrops(True)
        self.lbl_verUi.setText(f"Ui:{self.thisVer}")
        self.lbl_verUi.setStyleSheet(STYLE_PASS)
    
    def dragEnterEvent(self, event):
        if event.mimeData().hasUrls():
            self.state.log.error(f"Debug: Main dragEnterEvent {event}")
            event.acceptProposedAction()

    def dropEvent(self, event):
        for url in event.mimeData().urls():
            src = ""
            try:
                src = url.toLocalFile()
                self.state.log.error(f"src=[ {src} ]")
            except Exception as e:
                self.state.log.error(f"toLocalFile fail: {e}")
            self.state.signal_bus.update_signal.emit("MSG", f"upload: {src}",True)
            now = datetime.now()
            s = now.strftime("%Y%m%d_%H%M%S")
            ID = "unknown"
            with self.state.IDLock:
                ID = self.state.ID
            name = f"{ID}_{s}_{os.path.basename(src)}"
            self.state.log.error(f"name=[ {name} ]")
            dst = os.path.join(self.uploadRoot, name)
            self.state.log.error(f"dst=[ {dst} ]")
            tmp = os.path.join(r"D:\tmp",name)
            self.state.log.error(f"tnp=[ {tmp} ]")
            os.makedirs(r"D:\tmp",exist_ok=True)
            try:
                if os.path.isdir(src):
                    shutil.copytree(
                        src,
                        tmp,
                        dirs_exist_ok=True
                    )
                else:
                    if src != tmp:
                        shutil.copy2(src, tmp)
                os.makedirs(self.uploadRoot, exist_ok=True)
                os.rename(tmp,dst)
            except Exception as e:
                i = f"Copy Upload Faill {e}"
                self.state.log.error(i)
                self.state.signal_bus.update_signal.emit("MSG",i,False)
                
    def set_status(self, lblName, text, ok):
        lbl = self.labels.get(lblName)
        if lbl is None:
            print(f"lbl {lblName} không tồn tại")
            print(text,ok)
            return
        lbl.setText(str(text))
        lbl.setStyleSheet(STYLE_PASS if ok else STYLE_FAIL)
        print(f"{lblName} {text} {ok}")

    def setPost(self):

        screen = QApplication.primaryScreen().availableGeometry()

        x = (screen.width() - 1200) // 2
        y = 0

        self.move(x, y)

    def closeEvent(self, event):
        event.ignore()
        self.hide()

    def tray_click(self, reason):

        if reason == QSystemTrayIcon.ActivationReason.DoubleClick:

            if self.isVisible():
                self.hide()
            else:
                self.show()
                self.raise_()
                self.activateWindow()

    def on_ver_click(self):
        URLHome = ""
        with self.state.URLHomeLock:
            URLHome = self.state.URLHome
        if URLHome == "":
            return
        subprocess.Popen(
            f"cmd /c start {URLHome}",
            shell=True,
            creationflags=subprocess.CREATE_NO_WINDOW
        )
    
    def on_crclocker_click(self):
        self.state.send_queue.put({
            "type": "click",
            "key": "crc_locker",
        })

    def on_pgmlocker_click(self):
        self.state.send_queue.put({
            "type": "click",
            "key": "pgm_locker"
        })
    
    def on_crc_click(self):
        self.state.send_queue.put({
            "type": "click",
            "key": "crc"
        })
    
    def on_pgm_click(self):
        self.state.send_queue.put({
            "type": "click",
            "key": "pgm"
        })
    
    def on_merge_click(self):
        self.state.send_queue.put({
            "type": "click",
            "key": "merge"
        })
    
    def on_id_click(self):
        self.state.send_queue.put({
            "type": "click",
            "key": "id"
        })
        
    def on_connect_click(self):
        self.state.send_queue.put({
            "type": "click",
            "key": "connect"
        })
    
    def on_msg_click(self):
        self.state.send_queue.put({
            "type": "click",
            "key": "msg"
        })

