import ctypes, threading,queue
import sys,logging
from PySide6.QtWidgets import (
    QApplication,
)
from PySide6.QtCore import Signal,QObject
from Ui_main import MainWindow
from upload import UploadFile
from UploadMes import MesUpload
from pipe import PIPE
from GetCRC import CRCChecker
from ver import ver

VER = ver
upload_root = r"D:\SendFile" #r"D:\SendFile"

class SignalBus(QObject):
    update_signal = Signal(str, str, bool)
    
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(message)s",
    filename=r"D:\log\app.log",
    encoding="utf-8"
)

class Glob:
    def __init__(self):
        self.ID = ""
        self.IDLock = threading.Lock()
        
        self.URLUpload = ""
        self.URLUploadLock = threading.Lock()
        
        self.URLHome = ""
        self.URLHomeLock = threading.Lock()
        
        self.send_queue: queue.Queue[dict] = queue.Queue()
        
        
        self.signal_bus = SignalBus()
        
        self.CRC = ""
        self.CRCLock = threading.Lock()
        
        self.icoPath = r'C:\ProgramData\Locker\_internal\logo.ico'
        self.log = logging.getLogger("App")
               
if __name__ == "__main__":
    mutex = ctypes.windll.kernel32.CreateMutexW( None,  False,  "Global\\Uilock" )
    if ctypes.windll.kernel32.GetLastError() == 183:
        print("Đã có instance đang chạy")
        sys.exit(0)
    state = Glob()
    app = QApplication(sys.argv)
    window = MainWindow(VER=VER,state=state,uploadRoot=upload_root)
    PIPE(VER,state)
    checker = CRCChecker(
        state,
        target_name="CRC",
        interval=0.5
    )
    threading.Thread(
        target=checker.start,
        daemon=True
    ).start()
    window.show()
    UploadFile(State=state,uploadRoot=upload_root)
    threading.Thread(
        target=MesUpload,
        daemon=True
    ).start()
    sys.exit(app.exec())
    
    
