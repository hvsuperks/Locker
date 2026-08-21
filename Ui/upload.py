import time,  os, requests,threading
from datetime import datetime
lasttime = time.time()

class UploadFile():
    def __init__(self,State,uploadRoot):
        self.state = State
        self.uploadRoot = uploadRoot
        self.ID = ""
        threading.Thread(
            target=self.start,
            daemon=True
        ).start()
        
    def start(self):
        URLUpload = ""
        while True:
            time.sleep(5)
            with self.state.IDLock:
                if self.state.ID == "":
                    continue
                self.ID = self.state.ID
                    
            with self.state.URLUploadLock:
                URLUpload = self.state.URLUpload
            if URLUpload != "":
                for root, _, files in os.walk(self.uploadRoot):
                    for name in files:
                        if name.lower().endswith(".tmp"):
                            continue
                        filepath = os.path.join(root,name)
                        rel = os.path.relpath(root, self.uploadRoot)
                        if root == self.uploadRoot:
                            rel = ""
                        if not self.upload_file(URLUpload,filepath,rel):
                            break
                        try:
                            os.remove(filepath)
                        except Exception as e:
                            self.state.log.error("Delete fail %s : %s", filepath, e)
                
    def upload_file(self,url, file_path, target_folder) -> bool:
            with open(file_path, "rb") as f:
                self.state.log.info(f"Upload: {file_path} {target_folder}")
                data = {
                    "folder": target_folder
                }
                files = {
                    "myFile": (os.path.basename(file_path), f)
                }
                r = requests.post(
                    url,
                    files=files,
                    data=data,
                    timeout=300
                )
            if r.status_code != 200:
                self.state.log.error(f"{url}/n{file_path}")
                i = f"Upload failed: {r.status_code} {r.reason} {r.text}"
                self.state.log.error(i)
                self.state.signal_bus.update_signal.emit("MSG",i,False)
                return False
            self.state.signal_bus.update_signal.emit("MSG","Upload PASS",True)
            return True