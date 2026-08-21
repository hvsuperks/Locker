import subprocess,os
import time,psutil,win32process
import win32gui
import win32con

TITLE = "jahwa_ecm_agent_v2.exe"
EXE = r"C:\ProgramData\Jahwa Electronics\ECM\ECM Client Agent\10.1.1.36\Jahwa ECM Agent V2\Jahwa_ECM_Agent_V2.exe"

def MesUpload():
    def get_pids():
        pids = []

        for p in psutil.process_iter(["pid", "name"]):
            try:
                if p.info["name"] and p.info["name"].lower() == TITLE.lower():
                    pids.append(p.info["pid"])
            except:
                pass

        return pids


    def minimize_windows(pid):
        def callback(hwnd, _):
            if not win32gui.IsWindowVisible(hwnd):
                return

            _, win_pid = win32process.GetWindowThreadProcessId(hwnd)

            if win_pid == pid:
                win32gui.ShowWindow(hwnd, win32con.SW_MINIMIZE)

        win32gui.EnumWindows(callback, None)


    while True:
        pids = get_pids()

        # Không có process -> mở
        if len(pids) == 0 and os.path.exists(EXE):
            subprocess.Popen(EXE)
            time.sleep(3)

        # Nhiều process -> kill hết rồi mở lại
        elif len(pids) > 1:
            for pid in pids:
                try:
                    psutil.Process(pid).kill()
                except:
                    pass

            time.sleep(2)
            subprocess.Popen(EXE)
            time.sleep(3)

        # Luôn ép minimize
        pids = get_pids()

        if len(pids) == 1:
            minimize_windows(pids[0])

        time.sleep(1)
        
        
