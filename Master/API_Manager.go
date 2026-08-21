package main

import (
	"Locker/config"
	"net/http"
	"os"
	"path/filepath"
)

func apiManager() {
	mux := http.NewServeMux() // Khuyên dùng Mux riêng thay vì http mặc định
	fileServer := http.FileServer(http.Dir(config.HttpDir))
	web := http.FileServer(http.Dir(config.WebPathDir))
	mux.Handle("/master/", http.StripPrefix("/master/", fileServer))
	mux.HandleFunc("/master", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/master/", http.StatusMovedPermanently)
	})

	mux.HandleFunc("/addPGMZipFile", PGMZipUpload)

	mux.HandleFunc("/GetcrcEdit", crcEditGet)
	mux.HandleFunc("/HeaderEdit", HeaderEdit)
	mux.HandleFunc("/api/HeaderEdit", apiHeaderEdit)
	mux.HandleFunc("/api/SetcrcEdit", apiEditGet)
	mux.HandleFunc("/api/ActiveClient", apiActiveClient)
	// 3. Tạo trình phục vụ file tĩnh
	mux.Handle("/", web)
	mux.HandleFunc("/editweb", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(
			w,
			r,
			filepath.Join(
				config.WebPathDir,
				"ApiWeb",
				"WebEditHTMLAPI.html",
			),
		)
	})
	mux.HandleFunc("/api/GetStatus", APIStatus)
	mux.HandleFunc("/api/upload", API_uploadHandler)
	mux.HandleFunc("/upload", uploadHandler)
	mux.HandleFunc("/uploadAOI", uploadAOIHandler)
	mux.HandleFunc("/api/update", updateHandler)
	mux.HandleFunc("/api/get-thead", getListRawDataVBA)
	mux.HandleFunc("/api/get-chetaoMap", getListChetao)
	mux.HandleFunc("/api/ver", verChange)
	mux.HandleFunc("/api/RawDataPost", HandleCSV)
	mux.HandleFunc("/api/GetTime", GetTime)
	mux.HandleFunc("/api/ClientReport", ClientReport)
	mux.HandleFunc("/api/GetPingPong", ClientGetPingPong)
	mux.HandleFunc("/api/resetVNC", resetVNC)
	mux.HandleFunc("/api/resetMES", resetMES)
	mux.HandleFunc("/api/LockerChange", LockerChange)
	mux.HandleFunc("/api/ModelConfigChange", ModelConfigChange)
	mux.HandleFunc("/api/removeClient", removeClient)
	mux.HandleFunc("/api/master/resetvnc", resVNC)
	mux.HandleFunc("/api/PGMZipUpload", API_PGMZipUpload)
	mux.HandleFunc("/api/command", apiCommand)
	mux.HandleFunc("/api/ClientCheckUpdate", clientGetVer)
	mux.HandleFunc("/api/GetWebEdit", Webedit)
	mux.HandleFunc("/api/GetListWebEdit", WebapiList)
	mux.HandleFunc("/api/SaveWebEdit", WebeditPost)
	mux.HandleFunc("POST /api/RemoveCRC", RemoveCRC)

	err := http.ListenAndServe(":"+config.MasterPort, mux)
	Fmt(fmtMaster, "MasterStart", "❌ Server lỗi:", err)
	os.Exit(0)
}
