import time,json,threading
import win32file, queue
from typing import cast
from typing import Any

class PIPE():
    def __init__(self,ver,State):
        self.state = State
        self.thisVer = ver
        self.PipeLocker = None
        self.ID = ""
        threading.Thread(
            target=self.Heartbeat,
            daemon=True
        ).start()
        
        threading.Thread(
            target=self.LockerConnect,
            daemon=True
        ).start()
        
        threading.Thread(
                target=self.sendLocker,
                daemon=True
            ).start()
        
        
    def uichange(self,data):
        if data["key"] == "ID" and self.ID != data["value"]:
            self.ID = data["value"][-9:]
            with self.state.IDLock:
                self.state.ID = self.ID
        self.state.signal_bus.update_signal.emit(
            data["key"],
            data["value"],
            data["state"]
        )

    def Heartbeat(self):
        pipe: Any = None
        isVer = ""
        thisverEncode = self.thisVer.encode()
        while True:
            try:
                pipe = win32file.CreateFile(
                    r'\\.\pipe\Service_ui',
                    win32file.GENERIC_READ | win32file.GENERIC_WRITE,
                    0,
                    None,
                    win32file.OPEN_EXISTING,
                    0,
                    None
                )
                while True:
                    win32file.WriteFile(cast(int, pipe),thisverEncode )
                    
                    rc, data = win32file.ReadFile(cast(int,pipe), 1024)
                    data = cast(bytes, data)
                    ver = data.decode("utf-8")
                    if isVer != ver:
                        d ={
                            "key":"VERSERVICE",
                            "value":f"SC:{ver}",
                            "state":True
                        }
                        self.uichange(d)
                        isVer = ver
                    time.sleep(0.5)

            except Exception as e:
                self.state.log.error(f"AutoUpdate Pipe {e}")
                try:
                    win32file.CloseHandle(pipe)
                except:
                    pass
                d ={
                    "key":"VERSERVICE",
                    "value":"SC:None",
                    "state":False
                }
                isVer = "None"
                self.uichange(d)
                time.sleep(0.5)

    def sendLocker(self):
        while True:
            try:
                data = self.state.send_queue.get(timeout=1)
            except queue.Empty:
                data = {
                    "type": "ver",
                    "key": self.thisVer
                }
                time.sleep(0.5)
            try:
                payload = (
                    json.dumps(data, ensure_ascii=False)
                    + "\n"
                ).encode("utf-8")
                win32file.WriteFile(cast(int,self.pipeLocker), payload)
            except Exception:
                self.pipeLocker = None
                d ={
                    "key":"VERSION",
                    "value":"GO:None",
                    "state":False
                }
                self.uichange(d)
                
    def LockerConnect(self):
        while True:
            try:
                self.pipeLocker = win32file.CreateFile(
                    r'\\.\pipe\Locker',
                    win32file.GENERIC_READ | win32file.GENERIC_WRITE,
                    0,
                    None,
                    win32file.OPEN_EXISTING,
                    0,
                    None
                )

                while True:
                    rc, data = win32file.ReadFile(cast(int,self.pipeLocker), 4096)
                    data = cast(bytes, data)
                    datas = json.loads(data.decode("utf-8"))
                    
                    for k, v in datas.items():
                        match k:
                            case "uichange":
                                for item in v:
                                    self.uichange(item)
                            case "url":
                                for item in v:
                                    if item.get("key") == "home":
                                        with self.state.URLHomeLock:
                                            self.state.URLHome = item.get("value")
                                    elif item.get("key") == "upload":
                                        with self.state.URLUploadLock:
                                            self.state.URLUpload = item.get("value")

            except Exception as e:
                try:
                    win32file.CloseHandle(cast(int,self.pipeLocker))
                except:
                    pass
                self.state.log.error(f"Locker Pipe {e}")
                time.sleep(0.5)