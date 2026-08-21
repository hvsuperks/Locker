// Biến lưu trữ kết nối socket
let socket;
let activeMachine = null;
 let groups = [];
  let max_id = {
          marking: 0,
          af: 0,
          ois: 0,
          tilt: 0,
          prism_af: 0,
          prism_ois: 0, 
          fra_af: 0,
          fra_ois: 0,
          af_90: 0,
          aoi: 0
      };
  let cd = ["Marking","AF","OIS","Tilt","Prism_AF","Prism_OIS","Fra_AF","Fra_OIS","AF_90","AOI"];
  let mays = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10];
  let masterconfig = null;
  let md5map = null;
  let regmap = null;
  let connect = Boolean;
  let modelselect = "";
  let congdoanselect = "";
  
/**
 * Khởi tạo kết nối WebSocket
 */
function initConnection() {
    // Tự động nhận diện IP/Port của server Go từ thanh địa chỉ trình duyệt
    const host = window.location.host;
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    //const wsUrl = `${protocol}//${host}/ws/web`;
    const wsUrl = `ws://172.16.219.198:50001/ws/web`;
    console.log(wsUrl);
    socket = new WebSocket(wsUrl);

    socket.onopen = () => {
        console.log("Connected to Go Server");
        updateStatusIndicator(true);
    };

    socket.onclose = () => {
        console.log("Disconnected. Reconnecting...");
        updateStatusIndicator(false);
        // Thử kết nối lại sau 2 giây
        setTimeout(initConnection, 2000);
    };

    socket.onerror = (error) => {
        console.error("Socket Error: ", error);
    };

    // Lắng nghe dữ liệu từ Go gửi xuống
    socket.onmessage = (event) => {
        try {
            const message = JSON.parse(event.data);
            handleServerMessage(message);
        } catch (e) {
            console.error("Lỗi parse JSON:", e);
        }
    };
}

/**
 * Xử lý các loại tin nhắn từ Server
 */
function handleServerMessage(data) {
    switch (data.type) {
        case "status":
            renderTable(data.payload);
            break;
        case "notification":
            alert(data.message);
            break;
        default:
            console.log("Loại tin nhắn lạ:", data.type);
    }
}

/**
 * Gửi lệnh lên Go
 */
function sendCommand(action, details) {
    if (socket && socket.readyState === WebSocket.OPEN) {
        const payload = {
            type: "execute-command",
            payload: {
                action: action,
                ...details
            }
        };
        socket.send(JSON.stringify(payload));
    } else {
        alert("Chưa kết nối được tới server!");
    }
}

/**
 * Cập nhật giao diện (DOM Manipulation)
 */
function renderTable(payload) {
    const tableBody = document.getElementById('tableBody');
    if (!tableBody) return;

    // Ví dụ tạo row dựa trên dữ liệu nhận được
    let html = "";
    payload.model.forEach(m => {
        html += `
            <tr>
                <td>${m.model_name}</td>
                <td onclick="openActionMenu(event, '${m.model_name}')">Click để điều khiển</td>
            </tr>
        `;
    });
    tableBody.innerHTML = html;
}

function updateStatusIndicator(isOnline) {
    const statusEl = document.getElementById('status');
    if (statusEl) {
        statusEl.innerText = isOnline ? "Online" : "Offline";
        statusEl.style.color = isOnline ? "green" : "red";
    }
}

// Giả lập dữ liệu Map nhiều tầng từ Go
const mockData = {
    "Model_001": { "Công đoạn A": ["Option 1", "Option 2"] },
    "Model_002": { "Công đoạn B": ["Option A", "Option B"] },
    "Model_003": { "Công đoạn C": ["X", "Y", "Z"] },
    "Model_004": { "Công đoạn D": ["100", "200"] }
};

function renderGrid(data) {
    const container = document.getElementById('grid-container');
    let htmlContent = "";

    // Duyệt qua từng Model (Tương ứng với 1 cột)
    Object.entries(data).forEach(([modelName, congdoanObj]) => {
        
        // Trong mỗi Model, duyệt qua các Công đoạn
        Object.entries(congdoanObj).forEach(([cdName, options]) => {
            
            // Tạo template cho 1 cột
            htmlContent += `
                <div class="grid-column">
                    <div class="cell model-header">${modelName}</div>
                    <div class="cell congdoan-label">${cdName}</div>
                    <div class="cell select-box-cell">
                        <select onchange="handleSelect('${modelName}', this.value)">
                            <option value="">Select Box</option>
                            ${options.map(opt => `<option value="${opt}">${opt}</option>`).join('')}
                        </select>
                    </div>
                </div>
            `;
        });
    });

    container.innerHTML = htmlContent;
}


console.log("Script đã nạp thành công!"); // Phải thấy dòng này

window.onload = () => {
    console.log("Sự kiện onload đã kích hoạt");
    renderGrid(mockData)
    try {
        initConnection();
    } catch (err) {
        console.error("LỖI THỰC THI initConnection:", err);
    }
};