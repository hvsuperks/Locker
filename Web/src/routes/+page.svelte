<script>
import { onMount } from 'svelte';

  // --- DỮ LIỆU TRẠNG THÁI (Svelte 5 Runes) ---
  let groups = $state([]);
  let Status = $state({})
  let cd = ["Marking", "AF", "OIS", "Tilt","Prism", "Fra_AF", "Fra_OIS", "Flag", "AOI"];
  let actionControl = $state("command");
  let modelMap = $state({
    10840: "MD10840",
    16849: "MD16849",
    15342: "MD15342",
    12338: "MD12338",
    18049:"MD18049",
    11542:"MD11542",
    15984:"MD15984",
    17997:"MD17997",
    17748:"P17748"
  })
  let mays = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10];
  let ListFile = $state(null)
  let modelconfig = $state(null);
  let md5map = $state(null);
  let regmap = $state(null);
  let connect = $state(false);
  
  let menuVisible = $state(false);
  let menuselect = $state("");
  let modelselect = $state("");
  let congdoanselect = $state("");
  let menuPos = $state({ x: 0, y: 0 });
  let activeMachine = $state(null);
  let reconnectInterval = 1000; // Thử lại sau 3 giây
  let lastPing = Date.now();
  let natLoginState = $state(false);
  let countData = $derived.by(() => {
      let total = 0;

      // 1. Duyệt qua mảng model (ví dụ: '108402', '108401'...)
      groups.forEach(m => {
        
        // 2. Duyệt qua các công đoạn (ví dụ: 'AF', 'OIS', 'TILT'...)
        for (const tenCongDoan in m.congdoan) {
          const danhSachMay = m.congdoan[tenCongDoan];

          // 3. Duyệt qua từng máy trong công đoạn đó (ví dụ: máy '1', máy '2'...)
          for (const idMay in danhSachMay) {
            const may = danhSachMay[idMay];

            // 4. Kiểm tra điều kiện như bạn muốn
            if (may.connect?.color === "OK" && may.merge?.value) {
              // Cộng dồn giá trị (ép kiểu Number để tránh bị nối chuỗi)
              // Chuyển đổi sang số, nếu lỗi (NaN) thì lấy giá trị là 0
                    const val = parseInt(may.merge.value);
                    
                    if (!isNaN(val)) {
                        total += val;
                    }
            }
          }
        }
      });

      return total;
    });

  let reportControl =$state(null)
  let rp = $derived.by(() => {
    return reportControl
  });
  // Svelte 5 syntax
  let myMap = $state({
    "MienBac": {
      "HaNoi": { "BaDinh": "10000" }
    }
  });

  // 1. THÊM phần tử mới (cần key mới)
  function addItem(vung, tinh, quan, code) {
    if (!myMap[vung]) myMap[vung] = {};
    if (!myMap[vung][tinh]) myMap[vung][tinh] = {};
    myMap[vung][tinh][quan] = code;
  }

  // 2. XÓA phần tử
  function deleteItem(vung, tinh, quan) {
    delete myMap[vung][tinh][quan];
    // Nếu tỉnh trống, xóa luôn tỉnh
    if (Object.keys(myMap[vung][tinh]).length === 0) delete myMap[vung][tinh];
  }
  async function loadStatus() {
    try {
        const res = await fetch("/api/GetStatus");

        if (!res.ok) {
            throw new Error("API Error");
        }

        const data = await res.json();

        if (data.type === "status") {
            console.log(data)
            lastPing = Date.now();
            const payload = data.data;
            groups = payload.model;
            modelconfig = payload.masterconfig.modelconfig;
            md5map = payload.masterconfig.md5map;
            regmap = payload.masterconfig.regmap;
            connect = payload.connect;
            natLoginState = payload.natlogin;
            ListFile = payload.filelist;
        }

    } catch (e) {
        console.error(e);
    }
  }
  loadStatus();
  setInterval(() => {
      loadStatus();
      const diff = Date.now() - lastPing;
      if (diff > 5000) {
          console.log("Mất kết nối server");
          connect = false;
      }
  }, 1000);

  function copyToClipboard(text) {
    // Cách 1: Sử dụng API hiện đại nếu môi trường hỗ trợ (HTTPS / Localhost)
    if (navigator.clipboard && window.isSecureContext) {
      navigator.clipboard.writeText(text)
        .then(() => console.log("Copy thành công!"))
        .catch(err => console.error("Lỗi khi copy: ", err));
    } else {
      // Cách 2: Phương pháp dự phòng (Fallback) cho HTTP / Trình duyệt cũ
      const textArea = document.createElement("textarea");
      textArea.value = text;
      
      // Tránh làm cuộn trang web khi thêm phần tử
      textArea.style.position = "fixed";
      textArea.style.top = "0";
      textArea.style.left = "0";
      textArea.style.opacity = "0";
      
      document.body.appendChild(textArea);
      textArea.focus();
      textArea.select();
      
      try {
        const successful = document.execCommand('copy');
        if (successful) {
          console.log("Copy thành công bằng phương pháp dự phòng!");
        } else {
          console.error("Không thể copy bằng phương pháp dự phòng.");
        }
      } catch (err) {
        console.error("Lỗi hệ thống khi cố gắng copy: ", err);
      }
      
      document.body.removeChild(textArea);
    }
  }
  
  // --- HÀM XỬ LÝ GIAO DIỆN ---
  function openMenu(e, model, cd_name,id, mode) {
    activeMachine = { model, cd: cd_name,id,  mode };
    console.log(model, cd_name, mode)
    e.stopPropagation();
    e.preventDefault()
    menuVisible = true;
    menuselect = mode;
    modelselect = model;
    congdoanselect = cd_name;
    menuPos = { x: e.clientX, y: e.clientY };
    
  }

  async function handleAction(type,action) {
    if (!confirm("Xác nhận thay đổi "+ type + ":" + action)) {
            return;
        }
    if (activeMachine) {
            // Chạy lệnh gửi sang Backend Go
            SendCommandToGo(
                activeMachine.model, 
                activeMachine.cd, 
                activeMachine.id, 
                activeMachine.mode, 
                type,
                action
            );
            
            // Nếu gửi thành công thì mới tắt menu
            menuVisible = false; 
    }
  }

  async function SendCommandToGo(model,cd,id,mode,type,action) {
      if (type == "lockerpgm" || type == "lockercrc") {
        lockerChange(id,type,action)
      }else if (type == "pgm" || type == "crc") {
        ModelConfigChange(id,type,action)
      }
	}

  async function commandSend(e){
        const form = new FormData();
        form.append("id", document.getElementById("id").value);
        form.append("action", document.getElementById("code").value);
        const res = await fetch("/api/command", {
        method: "POST",
        body: form
        });
        const text = await res.text();
        console.log(text);
  }

  async function lockerChange(id, mode, state) {
      try {
          const resp = await fetch("/api/LockerChange", {
              method: "POST",
              body: new URLSearchParams({
                  id,
                  mode,
                  state
              })
          });
          const msg = await resp.text();
          if (resp.ok) {
              alert("Thành công: " + msg);
          } else {
              alert("Lỗi: " + msg);
          }
      } catch (err) {
          alert("Kết nối thất bại: " + err);
      }
  }

  async function ModelConfigChange(id, mode, value) {
      try {
          const resp = await fetch("/api/ModelConfigChange", {
              method: "POST",
              body: new URLSearchParams({
                  id,
                  mode,
                  value
              })
          });
          const msg = await resp.text();
          if (resp.ok) {
              alert("Thành công: " + msg);
          } else {
              alert("Lỗi: " + msg);
          }
      } catch (err) {
          alert("Kết nối thất bại: " + err);
      }
  }

  async function ResetVNC(e,id) {
      try {
          const resp = await fetch("/api/resetVNC", {
              method: "POST",
              body: new URLSearchParams({
                  id,
              })
          });
          const msg = await resp.text();
          if (resp.ok) {
              alert("Thành công: " + msg);
          } else {
              alert("Lỗi: " + msg);
          }
      } catch (err) {
          alert("Kết nối thất bại: " + err);
      }
  }

  async function ResetMES(e,id) {
      try {
          const resp = await fetch("/api/resetMES", {
              method: "POST",
              body: new URLSearchParams({
                  id,
              })
          });
          const msg = await resp.text();
          if (resp.ok) {
              alert("Thành công: " + msg);
          } else {
              alert("Lỗi: " + msg);
          }
      } catch (err) {
          alert("Kết nối thất bại: " + err);
      }
  }

  async function removeClient(id) {
      try {
          const resp = await fetch("/api/removeClient", {
              method: "POST",
              body: new URLSearchParams({
                  id,
              })
          });
          const msg = await resp.text();
          if (resp.ok) {
              alert("Thành công: " + msg);
          } else {
              alert("Lỗi: " + msg);
          }
      } catch (err) {
          alert("Kết nối thất bại: " + err);
      }
  }

  function closeMenu() { menuVisible = false; }

 
  function copyID(idtext){
    let t = "ID:" + idtext
    // Thay vì gọi navigator.clipboard.writeText(t), hãy gọi:
    copyToClipboard(t);
  }
  function sendViaSocket(e) {
        const idVal = document.getElementById('id').value;
        const codeVal = document.getElementById('code').value;
        console.log(idVal,codeVal)
        SendCommandToGo("","",idVal,"cmd","cmd",codeVal)
  }
  
  async function activeClient(e,mode) {
        console.log(document.getElementById('activeClient').value)
          const res = await fetch('/api/ActiveClient', {
              method: 'POST',
              headers: {
                  'Content-Type': 'application/json'
              },
              body: JSON.stringify({
                  value: document.getElementById('activeClient').value,
                  mode:mode
              })
          });

          const data = await res.text();
          console.log(data);
      }

  async function removeApp(e,mode) {
        console.log(document.getElementById('removeID').value)
          const res = await fetch('/api/removeApp', {
              method: 'POST',
              headers: {
                  'Content-Type': 'application/json'
              },
              body: JSON.stringify({
                  id: document.getElementById('removeID').value,
              })
          });

          const data = await res.json();
          console.log(data);
      }

      
  
  async function loginNat(e) {
        console.log(document.getElementById('user').value)
        console.log(document.getElementById('pass').value)
          const res = await fetch('/api/NatLogin', {
              method: 'POST',
              headers: {
                  'Content-Type': 'application/json'
              },
              body: JSON.stringify({
                  user: document.getElementById('user').value,
                  pass: document.getElementById('pass').value
              })
          });

          const data = await res.text();
          console.log(data);
      }
  
  async function postVer(e) {
        console.log(document.getElementById('modeVer').value)
        console.log(document.getElementById('valueVer').value)
          const res = await fetch('/api/ver', {
              method: 'POST',
              headers: {
                  'Content-Type': 'application/json'
              },
              body: JSON.stringify({
                  mode: document.getElementById('modeVer').value,
                  value: document.getElementById('valueVer').value
              })
          });

          const data = await res.json();
          console.log(data);
      }

    function changeAction() {
        const value = document.getElementById("actionType").value;
        ["command", "ver", "remove", "active", "login"].forEach(v => {
            document.getElementById(v + "Div").style.display =
                value === v ? "block" : "none";
        });
    }
  const headerURL = `${window.location.origin}/HeaderEdit`

</script>

<!-- Svelte 5 dùng window listener kiểu mới hoặc giữ nguyên kiểu cũ đều được -->
<svelte:window onclick={closeMenu} />

<main >
    <div class="TopBox">
      <div class="TopDiv">
        <button type="button" onclick={() => window.open(headerURL, "_blank")}>Header Config</button>
        Data : {countData}
        <!-- Nhập văn bản bình thường -->
        const [actionControl, setAction] = useState("command");
        <select bind:value={actionControl}>
            <option value="command">Command</option>
            <option value="ver">Version</option>
            <option value="remove">Remove</option>
            <option value="active">Active</option>
            <option value="login">Login</option>
        </select>

        {#if actionControl === "command"}
              <div>
                  <input type="text" id="id" placeholder="id..." />
                  <input type="text" id="code" placeholder="code..." />
                  <button onclick={(e)=> commandSend(e)}>Send</button>
              </div>

          {:else if actionControl === "ver"}
              <div>
                  <input type="text" id="modeVer" placeholder="mode Ver" />
                  <input type="text" id="valueVer" placeholder="ver..." />
                  <button onclick={(e)=> postVer(e)}>Send</button>
              </div>

          {:else if actionControl === "remove"}
              <div>
                  <input type="text" id="removeID" placeholder="remove ID..." />
                  <button onclick={(e)=> removeApp(e)}>Send</button>
              </div>

          {:else if actionControl === "active"}
              <div>
                  <input type="text" id="activeClient" placeholder="vd:168492" />
                  <button onclick={(e)=> activeClient(e, "ADD")}>Add</button>
                  <button onclick={(e)=> activeClient(e, "Remove")}>Remove</button>
              </div>

          {:else if actionControl === "login"}
              <div>
                  <input type="text" id="user" placeholder="User" />
                  <input type="text" id="pass" placeholder="Pass" />
                  <button onclick={(e)=> loginNat(e)}>Login</button>
              </div>
          {/if}
          <span class="dot {natLoginState ? 'online' : 'offline'}"></span>
          Report : {rp}
      </div>
    </div>
    <div class="wrapper">
    {#each [...groups].sort((a, b) => a.model_name.localeCompare(b.model_name)) as model}
    <div class="window-wrapper">
    <div class="fixed-content">
    <div class="item">
      <section class="group-container">
        <div class="group-header">Line:{modelMap[model.model_name.slice(0,5)]}_Line {model.model_name.slice(5,6)}</div>
        {#each cd as congdoan_name,i}
          {@const congdoan = model.congdoan[congdoan_name]}
            {#if congdoan}
              <div class="item">
                <div class="sub-group-section">
                  <button 
                      type="button" class="sub-title" onclick={(e) => openMenu(e, model.model_name, congdoan_name,`${model.model_name}${i}`, "pgm")}>
                      {congdoan_name}:     {modelconfig[`${model.model_name}${i}`]?.pgm|| "N/A"}
                    </button>
                  <button 
                      type="button" class="sub-title" onclick={(e) => openMenu(e, modelconfig[`${model.model_name}${i}`].pgm.split("_")[0], congdoan_name,`${model.model_name}${i}`, "crc")}>
                      CRC:  {modelconfig[`${model.model_name}${i}`]?.pgm.split("_")[0]||"N/A"}_{modelconfig[`${model.model_name}${i}`]?.pgm.split("_")[1]||"N/A"}_{modelconfig[`${model.model_name}${i}`]?.crc||"N/A"}
                    </button>
                  <table>    
                    <thead class="taskbar">
                      <tr>
                        <th rowspan="2">ID</th> 
                        <th rowspan="2"><button class="invisible-button" aria-label="resetvnc" type="button" onclick={(e) => ResetVNC(e,`${model.model_name}${i}`)}>VNC <br>Reset</button></th>
                        <th colspan="2">PGM</th> 
                        <th colspan="2">CRC</th> 
                        <th rowspan="2">Data</th>
                        <th rowspan="2">Ver</th>
                      </tr>
                      <tr>
                        <th><button class="invisible-button" aria-label="PGM locker" type="button" onclick={(e) => openMenu(e, model.model_name, congdoan_name,`${model.model_name}${i}`, "lockerpgm")}>Locker</button></th>
                        <th>PGM Ver</th>
                        <th><button class="invisible-button" aria-label="CRC locker" type="button" onclick={(e) => openMenu(e, model.model_name, congdoan_name,`${model.model_name}${i}`, "lockercrc")}>Locker</button></th>
                        <th>CRC Ver</th>
                      </tr>
                    </thead>
                    <tbody class="taskbar">
                    
                      {#each mays as may}
                          <tr style="height: 30px;">
                            <td><button class="invisible-button" type="button" onclick={copyID(`${model.model_name}${i}0${may}`)}>{model.model_name}{i}<br>0{may}</button></td>
                            
                            {#if congdoan[may]}
                              {#if connect}                            
                                {@const m = congdoan[may]}
                                {#if m.connect.color == "OK"}
                                  <td><button class="invisible-button" aria-label="resetvnc" type="button" onclick={(e) => ResetVNC(e,`${model.model_name}${i}0${may}`)}>{m.mesid}</button></td>
                                  <td><button class="invisible-button" aria-label="PGM locker" type="button" onclick={(e) => openMenu(e, model.model_name, congdoan_name,`${model.model_name}${i}0${may}`, "lockerpgm")}><span class="dot {m.lockerpgm.value === 'Locked' ? 'online' : 'offline'}"></span></button></td>
                                  <th><span class="dot {m.pgm.color === 'OK' ? 'online' : 'offline'}"></span></th>
                                  <td><button class="invisible-button" aria-label="CRC locker" type="button" onclick={(e) => openMenu(e, model.model_name, congdoan_name,`${model.model_name}${i}0${may}`, "lockercrc")}><span class="dot {m.lockercrc.value === 'Locked' ? 'online' : 'offline'}"></span></button></td>
                                  <th><span class="crc-text" style="color:{m.crc.color === 'OK' ? 'green' : 'red'}">{m.currentcrc}</span></th>
                                  <td class = "status-{m.merge.color}"><button class="invisible-button" aria-label="resetmes" type="button" onclick={(e) => ResetMES(e,`${model.model_name}${i}0${may}`)}>{m.merge.color === 'OK' ? m.merge.value : 'Fail'}</button></td>
                                  
                                  <th>
                                    SC: {m.ver.split('|')[0]}<br>
                                    LO: {m.ver.split('|')[1]}<br>
                                    UI: {m.ver.split('|')[2]}
                                  </th>
                                {:else}
                                  <td class="status-Fail">X</td><td class="status-Fail">X</td><td class="status-Fail">X</td><td class="status-Fail">X</td><td class="status-Fail">X</td><td class="status-Fail">X</td><td class="status-Fail">X</td>
                                {/if}
                              {:else}
                                <td class="status-Fail">X</td><td class="status-Fail">X</td><td class="status-Fail">X</td><td class="status-Fail">X</td><td class="status-Fail">X</td><td class="status-Fail">X</td><td class="status-Fail">X</td>
                              {/if}
                            {/if}
                          </tr>
                      {/each}
                    </tbody>
                  </table>
                  </div>
              </div>
            {/if}
        {/each}
    </section>
  </div></div>
  </div> {/each}</div></main>

{#if menuVisible}
  <div class="context-menu" style="top: {menuPos.y}px; left: {menuPos.x}px;">
    {#if menuselect == "lockerpgm" || menuselect == "lockercrc"}
        <button onclick={() => handleAction(menuselect,'Locked')}>🔒 Khóa {menuselect.slice(-3).toUpperCase()}</button>
        <button onclick={() => handleAction(menuselect,'Unlock')}>🔓 Mở Khóa {menuselect.slice(-3).toUpperCase()}</button>
    {:else if menuselect == "pgm"}   

        {@const matchingGroups = Object.keys(md5map).filter(k => 
    k.includes(modelselect.slice(0,5)) && k.includes(congdoanselect)
  )}
  
  {@const allVersions = matchingGroups.flatMap(groupKey => 
    Object.keys(md5map[groupKey] || {})
  )}

  {#each allVersions as version}
    <button class="menu-item" onclick={() => handleAction(menuselect,version)}>
      {version}
    </button>
  {:else}
    <div class="no-data">Không tìm thấy PGM nào cho {modelselect}</div>
  {/each}
  {@const cdtmp = cd.indexOf(congdoanselect)}
  {@const mm = modelconfig[`${modelselect}${cdtmp}`]?.pgm?.split("_")[0]}
  <button class="menu-item"  onclick={() => window.open(`${window.location.origin}/addPGMZipFile?model=${mm}&cd=${congdoanselect}`, '_blank')}> +Add</button>
  {:else if menuselect == "crc"}
    {@const crclist = Object.keys(regmap).filter(k=> k.includes(modelselect) && k.includes(congdoanselect))}   
    {#each crclist as m}
      <div class="menu-row">
        <button onclick={() => handleAction(menuselect,m.split("_")[2])}>{m}</button>
        <button onclick={() => window.open(`${window.location.origin}/GetcrcEdit?name=${m}`, '_blank')}> Edit</button>
        <span class="info-text">{regmap[m]?.info?.info ||  regmap[m]?.info?.Info ||  regmap[m]?.info?.INFO || "Info None"}</span>
      </div>
    {/each}
      <div class="menu-row">
      <button 
      <button onclick={() => window.open(`${window.location.origin}/GetcrcEdit?name=new`, '_blank')}> +Add</button>
      </div>
    {/if}
  </div>
{/if}

<style>
  .row { display: flex; gap: 10px; margin-bottom: 5px; }

  /* 1. Reset chuẩn: Đưa margin và padding của TẤT CẢ về 0 để không bị lệch giữa các trình duyệt */
  * {
      box-sizing: border-box;
      margin: 0;
      padding: 0; /* Đưa về 0 chuẩn, không viết 10 nữa nha */
  }

  /* 2. Đảm bảo box-sizing hoạt động chuẩn trên cả các thành phần ảo */
  *, *::before, *::after {
      box-sizing: border-box;
  }

  /* Bao quát toàn bộ màn hình, cho phép cuộn ngang nếu nhiều Group */
  .wrapper {
    padding-bottom: 0;
    display: flex;
    flex-direction: row; /* Nối nhau về bên phải */
    justify-content: flex-start;
    gap: 1px;
    overflow-x: visible ; /* Tự xuất hiện thanh cuộn ngang */
    background: #f4f4f4;
    height: 90dvh;
  }
/* 1. Cái này bao toàn bộ cửa sổ App */
    .window-wrapper {
        overflow: auto; /* Tự hiện thanh cuộn nếu cửa sổ nhỏ hơn nội dung */
        flex-shrink: 0;
        min-width: 350px;
    }

    /* 2. ĐÂY LÀ CHỖ QUAN TRỌNG: Cố định kích thước nội dung */
    .fixed-content {
        min-height: 100%; 
        background-color: white; /* Màu nền của bảng dữ liệu */
        margin: 0 auto; /* Căn giữa nội dung nếu cửa sổ to hơn 1200px */
        padding-left: 10px;
        /* Chống co dãn */
        flex-shrink: 0;
        display: flex;
        flex-direction: column;
    }

    .taskbar {
        width: 100%; /* Nó sẽ ăn theo 1200px của thằng cha */
        border-bottom: 1px solid #eee;
    }

    .TopBox{
      padding: 20px;
            color: white;
            font-weight: bold;
            text-align: center;
            border-radius: 8px;
    }

    .TopDiv{
      width: 100%;
      position: fixed;
      top: 0;        /* Dính ở đỉnh khi cuộn dọc xuống */
      left: 0;
      display: flex;
      align-items: center;
      gap: 30px;
      padding: 10px;
      box-sizing: border-box;
      border-bottom: 1px solid #ccc; /* Đường kẻ phân cách (tùy chọn) */
      z-index: 1000;
    }

    .item {
        width: 500px; /* Anh có thể đặt cứng kích thước từng cột luôn */
        padding-top: 1px;
        padding-bottom: 1px;
        text-align: center;
    }
  /* 1. Vùng chứa mỗi Group (AF, OIS...) */
    .group-container {
        width: 100%;       /* Chiếm hết chiều ngang */
        margin: 0 auto;    /* Căn giữa */
        display: flex;
        flex-direction: column;
        flex-shrink: 0;
    }

  .group-header {
    background: #2c3e50;
    color: white;
    text-align: center;
    font-weight: bold;
    width: 100%;
    padding-top: 10px;
    padding-bottom: 10px;
    font-size: 1.1em;
    position: sticky;
    top: 0;
    z-index: 10;
  }

  /* Từng nhóm nhỏ AF, OIS... chia theo hàng dọc */
  .sub-group-section {
    border-bottom: 1px dashed #f0efef;
    width: 100%;
  }

  .sub-title {
    background: #ecf0f1;
    font-weight: bold;
    font-size: 0.9em;
    margin-bottom: 5px;
    padding-top: 5px;
    padding-bottom: 5px;
    text-align: center;
    width: 100%;
    height: 100%;
    display: block; /* Chuyển về block để chiếm trọn không gian */
    border: none;
  }

  table {
    width: 100%;
    border-collapse: collapse;
    font-size: 12px;
    border-collapse: collapse;
    text-align: center;
    table-layout: fixed;
  }
  td,th{
    padding-top: 5px;
        padding-bottom: 5px;
      }
  th,td { background: #f8f9fa; border: 1px solid #e5e9eb;}

  .status-OK { color: green; animation: blink 1s infinite;}
  .status-Fail { color: red;  }


    /* Tạo hình tròn cơ bản */
    .dot {
        width: 11px;
        height: 11px;
        border-radius: 50%;
        display: inline-block;
    }

    /* Đèn Xanh (Online) - Nháy chậm hơn chút cho đỡ mỏi mắt */
    .online {
        background-color: #2ecc71;
        box-shadow: 0 0 1px #2ecc71;
        animation: blink-green 5s infinite;
    }

    /* Đèn Đỏ (Offline) - Nháy nhanh để cảnh báo */
    .offline {
        background-color: #e74c3c;
        box-shadow: 0 0 1px #e74c3c;
        animation: blink-red 0.5s infinite;
    }

    /* Hiệu ứng nháy cho đèn Xanh */
    @keyframes blink-green {
        80% { opacity: 1; transform: scale(1); }
        90% { opacity: 0.5; transform: scale(0.95); }
        100% { opacity: 1; transform: scale(1); }
    }

    /* Hiệu ứng nháy cho đèn Đỏ */
    @keyframes blink-red {
        0% { opacity: 1; transform: scale(1); }
        50% { opacity: 0.5; transform: scale(0.95); }
        100% { opacity: 1; transform: scale(1); }
    }

    .item td,button {
        transition: text-shadow 0.2s;
        cursor: pointer;
    }

    .item td:hover {
        /* Đổ bóng sát vào chữ để nhìn như in đậm */
        text-shadow: 0.5px 0 0 currentColor, -0.5px 0 0 currentColor;
        color: #007bff; /* Anh có thể đổi màu cho nổi bật hơn */
    }
    .item button:hover {
        /* Đổ bóng sát vào chữ để nhìn như in đậm */
        color: #007bff; /* Anh có thể đổi màu cho nổi bật hơn */
    }

  .context-menu {
    position: fixed; /* Quan trọng: Để nó nổi lên trên tất cả */
    z-index: 1000;
    background: white;
    border: 1px solid #ccc;
    box-shadow: 2px 2px 10px rgba(0,0,0,0.2);
    display: flex;
    flex-direction: column;
    padding: 5px;
    border-radius: 4px;
    min-width: 120px;
  }

  .context-menu button {
    text-align: left;
    padding: 8px 12px;
    border: none;
    background: none;
    cursor: pointer;
  }

  .context-menu button:hover {
    background: #007bff;
    color: white;
  }
  .invisible-button {
    /* Quan trọng nhất: Bỏ hết giao diện mặc định của nút */
    background: none;
    border: none;
    padding: 0;
    margin: 0;
    cursor: pointer;
    
    /* Căn giữa cái chấm vào ô td */
    display: flex;
    align-items: center;
    justify-content: center;
    width: 100%;
    height: 100%;
    min-height: 20px; /* Tăng diện tích để dễ bấm */
  }
  
  .menu-row {
      display: flex;
      align-items: center;
      gap: 8px;
  }

  .info-text {
      color: #888;
      font-size: 12px;
      white-space: nowrap;
  }

  .crc-text{
    font-size: 12px;
  }
  
</style>