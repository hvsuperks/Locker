package main

import (
	"Locker/config"
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func CaseInsensitive(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/api/") {
			i := r.URL.Path
			r.URL.Path = strings.ToLower(r.URL.Path)
			LogInfo(&Logger.Debug, i, " => ", r.URL.Path)
		} else {
			LogInfo(&Logger.Debug, r.URL.Path)
		}
		next.ServeHTTP(w, r)
	})
}

func apiManager(ctx context.Context, cancal context.CancelFunc) {
	mux := http.NewServeMux() // Khuyên dùng Mux riêng thay vì http mặc định

	// Control Web
	mux.Handle("/", http.FileServer(http.Dir(config.WebPathDir)))
	// Web Dir File
	mux.Handle("/master/", http.StripPrefix("/master/", http.FileServer(http.Dir(config.HttpDir))))
	mux.HandleFunc("/master", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/master/", http.StatusMovedPermanently)
	})

	// Upload PGM đặc tính
	mux.HandleFunc("GET /api/pgmupload", Get_pgmUpload)
	mux.HandleFunc("POST /api/pgmUpload", Protect(perm_PGMUpload, POST_pgmUpload))

	// CRC Edit
	mux.HandleFunc("GET /api/crcedit", GET_crcEdit)
	mux.HandleFunc("POST /api/crcedit", Protect(perm_CRCEdit, POST_crcEdit))
	mux.HandleFunc("POST /api/crcremove", POST_crcRemove)

	// Header RowData Edit
	mux.HandleFunc("GET /api/headeredit", GET_headerEdit)
	mux.HandleFunc("POST /api/headeredit", Protect(perm_HeaderEdit, POST_headerEdit))

	// Active vs Remove Client
	mux.HandleFunc("POST /api/clientactive", Protect(perm_ClientActive, POST_clientActive))

	// Get Client Status
	mux.HandleFunc("GET /api/clientstatus", AuthMiddleware(GET_clientStatus))

	// Upload File to Master
	mux.HandleFunc("POST /api/uploadfile", POST_uploadFile)

	// Upload AOI
	mux.HandleFunc("POST /api/uploadaoi", POST_uploadAOI)

	// Update PGM
	mux.HandleFunc("POST /api/update", POST_updatePGM)

	// VBA get RawData
	mux.HandleFunc("/api/get-thead", getListRawDataVBA)
	mux.HandleFunc("/api/get-chetaoMap", getListChetao)
	mux.HandleFunc("/api/rawdatapost", HandleCSV)

	// Version Change
	mux.HandleFunc("POST /api/verchange", Protect(perm_VerChange, verChange))

	// Client Sync Time
	mux.HandleFunc("/api/getTime", GetTime)

	// Client Report
	mux.HandleFunc("POST /api/clientreport", POST_clientReport)

	// Client Ping Pong, Get control
	mux.HandleFunc("GET /api/getpingpong", Get_clientPingPong)

	// Api Control
	mux.HandleFunc("POST /api/resetvnc", POST_resetVNC)
	mux.HandleFunc("POST /api/resetmes", POST_resetMES)
	mux.HandleFunc("POST /api/lockerchange", Protect(perm_LockerChange, POST_LockerChange))
	mux.HandleFunc("POST /api/modelconfig", Protect(perm_ModelChange, POST_modelConfig))
	mux.HandleFunc("POST /api/clientremove", Protect(perm_RemoveClient, POST_clientRemove))
	mux.HandleFunc("POST /api/command", POST_apiCommand)

	// Edit Web API
	mux.HandleFunc("GET /api/webedit", func(w http.ResponseWriter, r *http.Request) {
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
	mux.HandleFunc("GET /api/editwebapi", GET_editWebAPI)
	mux.HandleFunc("POST /api/editwebapi", POST_editWebAPI)
	mux.HandleFunc("GET /api/listwebapi", GET_listWebAPI)

	mux.HandleFunc("GET /shutdown", func(w http.ResponseWriter, r *http.Request) {
		cancal()
		cleanupAndExit()
	})
	mux.HandleFunc("GET /api/me", AuthMiddleware(GET_meCheck))
	mux.HandleFunc("GET /login", GET_login)
	mux.HandleFunc("POST /login", POST_login)
	mux.HandleFunc("POST /register", AuthMiddleware(POST_CreateUser))
	mux.HandleFunc("POST /logout", POST_logout)
	mux.HandleFunc("GET /api/permissions", AuthMiddleware(GET_permissions))
	mux.HandleFunc("PUT /api/permissions", AuthMiddleware(PUT_permissions))
	mux.HandleFunc("PUT /api/changepassword", AuthMiddleware(PUT_changePassword))
	mux.HandleFunc("POST /api/resetpassword", AuthMiddleware(POST_reset_password))
	mux.HandleFunc("POST /api/deleteuser", AuthMiddleware(POST_deleteUser))

	err := http.ListenAndServe(":"+config.MasterPort, CaseInsensitive(mux))
	LogInfo(&Logger.API, "MasterStart", "❌ Server lỗi:", err)
	os.Exit(0)
}

// Helper gộp chung 2 lớp Middleware lại cho ngắn
func Protect(permission string, handler http.HandlerFunc) http.HandlerFunc {
	return AuthMiddleware(
		AuthorizePermission(permission)(handler),
	)
}

func cleanupAndExit() {
	wdChan := make(chan struct{}, 1)
	go func() {
		wg.Wait()
		close(wdChan)
	}()
	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()
	select {
	case <-timer.C:
	case <-wdChan:
	}
	os.Exit(0)
}
