import time
import threading,winreg,psutil
import uiautomation as auto

REG_PATH = r"SOFTWARE\Produc\lock"
REG_VALUE = "WindowTitle"
TARGET_EXE = "CubeManager.exe"

def reg_write(title):
    key = winreg.CreateKey(winreg.HKEY_CURRENT_USER, REG_PATH)

    winreg.SetValueEx(
        key,
        REG_VALUE,
        0,
        winreg.REG_SZ,
        title
    )

    winreg.CloseKey(key)


def reg_read():
    try:
        key = winreg.OpenKey(
            winreg.HKEY_CURRENT_USER,
            REG_PATH
        )

        value, _ = winreg.QueryValueEx(key, REG_VALUE)

        winreg.CloseKey(key)

        return value

    except:
        return None
    
    
def is_target_window(win):
    try:
        pid = win.ProcessId

        p = psutil.Process(pid)

        return p.name().lower() == TARGET_EXE.lower()

    except:
        return False

def window_has_crc(win):
    for control, depth in auto.WalkControl(win):
        try:
            if "CRC" in str(control.Name).upper():
                return True
        except:
            pass
    return False
        
def scan_title():
    root = auto.GetRootControl()

    for win in root.GetChildren():

        try:
            if not is_target_window(win):
                continue

            if window_has_crc(win):
                print("Find CRC:", win.Name)

                reg_write(win.Name)

                return win.Name

        except:
            pass

    return None

def load_saved_window():
    title = reg_read()

    if not title:
        return None

    win = auto.WindowControl(Name=title)

    if win.Exists(1):
        return win

    return None

def is_running(exe_name):
    exe_name = exe_name.lower()

    for proc in psutil.process_iter(['name']):
        try:
            if proc.info['name'] and proc.info['name'].lower() == exe_name:
                return True
        except:
            pass

    return False

def get_target_window():
    if not is_running(TARGET_EXE):
        return None
    # dùng cache trước
    win = load_saved_window()

    if win:
        print("Load from reg:", win.Name)
        return win

    # fail thì scan lại
    title = scan_title()

    if not title:
        return None

    return auto.WindowControl(Name=title)

class UIWatcher:
    def __init__(self,State, target_name, interval=5.0):
        self.app_name = ""
        self.target_name = target_name
        self.interval = interval

        self.win = None
        self.pid = None
        self.state = State
        self.target_info = None
        self.last_value = None

        self.running = False
        
    # =====================================================
    # CONNECT
    # =====================================================

    def connect(self):
        try:
            self.win = get_target_window()
            if not self.win:
                return False
            if not self.win.Exists(3):
                return False

            self.pid = self.win.ProcessId

            print(f"[OK] Connected PID={self.pid}")

            return True

        except Exception as e:
            print("[CONNECT]", e)
            return False

    # =====================================================
    # APP CHECK
    # =====================================================

    def is_app_alive(self):
        try:
            win = auto.WindowControl(Name=self.app_name)

            if not win.Exists(0):
                return False

            return win.ProcessId == self.pid

        except:
            return False

    # =====================================================
    # FIND ANCHOR
    # =====================================================

    def find_anchor(self, ctrl):
        depth = 0

        while ctrl:

            try:
                if ctrl.AutomationId:
                    return ctrl, depth

                if ctrl.NativeWindowHandle:
                    return ctrl, depth

                ctrl = ctrl.GetParentControl()
                depth += 1

            except:
                break

        return None, depth

    # =====================================================
    # FULL SCAN
    # =====================================================

    def discover(self):

        print("[SCAN] Searching target...")
        if self.win is None:
                return False

        for c, _ in auto.WalkControl(self.win):

            try:

                name = c.Name

                if not name:
                    continue

                if self.target_name not in name:
                    continue

                anchor, parent_depth = self.find_anchor(c)

                if not anchor:
                    continue

                self.target_info = {
                    "name": c.Name,
                    "autoid": c.AutomationId,
                    "control_type": c.ControlTypeName,
                    "anchor_name": anchor.Name,
                    "anchor_autoid": anchor.AutomationId,
                    "anchor_handle": anchor.NativeWindowHandle,
                    "parent_depth": parent_depth,
                }

                print("[FOUND]")
                print(self.target_info)

                return True

            except:
                pass

        return False

    # =====================================================
    # FIND CONTROL AGAIN
    # =====================================================

    def locate_control(self):

        info = self.target_info

        if not info:
            return None

        try:
            if self.win is None:
                return False
            for c, _ in auto.WalkControl(self.win):

                try:

                    if (
                        info["autoid"]
                        and c.AutomationId == info["autoid"]
                    ):
                        return c

                    if (
                        not info["autoid"]
                        and c.Name == info["name"]
                    ):
                        return c

                except:
                    pass

        except:
            pass

        return None

    # =====================================================
    # READ VALUE
    # =====================================================

    def read_value(self):

        ctrl = self.locate_control()

        if not ctrl:
            return None

        try:
            return ctrl.Name.strip()

        except:
            return None

    # =====================================================
    # REBUILD
    # =====================================================

    def rebuild(self):

        print("[REBUILD]")

        if not self.connect():
            return False

        return self.discover()

    # =====================================================
    # CHANGE EVENT
    # =====================================================

    def on_change(self, old, new):
        print(f"[CHANGE] {old} -> {new}")

    # =====================================================
    # LOOP
    # =====================================================

    def monitor(self):
        while True:
            crc = "None"
            try:
                while not self.is_app_alive():
                    if self.rebuild():
                        break
                    try:
                        self.state.send_queue.put({
                            "type": "crc",
                            "key": "AppClose"
                        })
                    except:
                        pass
                    time.sleep(2)
                value = self.read_value()
                if value is None:
                    if not self.rebuild():
                        try:
                            self.state.send_queue.put({
                                "type": "crc",
                                "key": "AppClose"
                            })
                        except:
                            pass
                        time.sleep(1)
                        continue
                else:
                    try:
                        crc = value.replace("CRC:", "").strip()
                        self.state.signal_bus.update_signal.emit("MSG",f"Current {value} ",True)
                        self.state.send_queue.put({
                            "type": "crc",
                            "key": crc
                        })
                    except:
                        pass
                    if value != self.last_value:
                        self.on_change(
                            self.last_value,
                            value
                        )
                        self.last_value = value
            except Exception as e:
                print("[ERROR]", e)
            time.sleep(self.interval)

    # =====================================================
    # START
    # =====================================================

    def start(self):
        while True:
            time.sleep(1)
            if is_running(TARGET_EXE):
                if not self.connect():
                    continue

                if not self.discover():
                    continue
                self.running = True
                break
        threading.Thread(
            target=self.monitor,
            daemon=True
        ).start()

    # =====================================================
    # STOP
    # =====================================================

    def stop(self):
        self.running = False

class CRCChecker(UIWatcher):
    def on_change(self, old, new):
        with self.state.CRCLock:
            self.state.CRC = new
        self.state.signal_bus.update_signal.emit("MSG",f"CRC Changed: {old} -> {new}",True)


