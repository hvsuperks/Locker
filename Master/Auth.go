package main

import (
	"Locker/config"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const (
	perm_PGMUpload    = "Upload PGM"
	perm_CRCEdit      = "CRC Edit"
	perm_ClientActive = "Client Acctive"
	perm_HeaderEdit   = "Header RawDat Edit"
	perm_LockerChange = "Locker Change"
	perm_VerChange    = "Update App"
	perm_ModelChange  = "PGM Change"
	perm_RemoveClient = "Client Remove"
)

var sessions = map[string]string{}
var sessionsMu sync.RWMutex

// Context key custom để tránh trùng lặp
type contextKey string

const (
	contextUsername    contextKey = "username"
	contextPermissions contextKey = "permissions"
)

// --- CONFIG & MODELS ---

var jwtKey = []byte("super_secret_jwt_key_change_me")

type User struct {
	Username    string `json:"username"`
	Password    string `json:"Password"`
	Role        string `json:"role"`
	Permissions string `gorm:"type:text"` // Lưu mảng []string dưới dạng JSON String trong DB
	CreateBy    string `json:"createby"`
}

type LoginRequest struct {
	Username  string `json:"username"`
	Password  string `json:"password"`
	ReturnUrl string `json:"returnUrl"`
}

type CreateUserRequest struct {
	Username    string   `json:"username"`
	Password    string   `json:"password"`
	Permissions []string `json:"permissions"`
}

// Helper: Trả về dữ liệu JSON
func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

// --- HELPER FUNCTIONS --
func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func checkPasswordHash(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func initAdmin() {
	hashedPassword, _ := hashPassword("Saobang1993@")
	perm := []string{perm_PGMUpload,
		perm_CRCEdit,
		perm_ClientActive,
		perm_HeaderEdit,
		perm_LockerChange,
		perm_VerChange,
		perm_ModelChange,
		perm_RemoveClient,
	}
	p, _ := json.Marshal(perm)
	newUser := User{
		Username:    "admin",
		Password:    hashedPassword, // Lưu ý: Nếu làm thực tế nên Hash password (bcrypt)
		CreateBy:    "",
		Permissions: string(p),
	}

	if err := gormDB.Create(&newUser).Error; err != nil {
		LogInfo(&Logger.Debug, "gormDB.Create", err)
		os.Exit(1)
	}

}

// --- MIDDLEWARES ---
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Lấy cookie username
		userCookie, err := r.Cookie("session_user")
		if err != nil {
			http.Error(w, "Chưa đăng nhập!", http.StatusUnauthorized)
			return
		}

		// 2. Lấy cookie thời gian đăng nhập
		timeCookie, err := r.Cookie("login_time")
		if err != nil {
			http.Error(w, "Phiên làm việc không hợp lệ!", http.StatusUnauthorized)
			return
		}

		sessionID, err := r.Cookie(`session_id`)
		if err != nil {
			http.Error(w, "Phiên làm việc không hợp lệ!", http.StatusUnauthorized)
			return
		}
		sessionsMu.RLock()
		ID, ok := sessions[userCookie.Value]
		sessionsMu.RUnlock()
		if !ok || ID != sessionID.Value {
			http.Error(w, "Phiên làm việc không hợp lệ!", http.StatusUnauthorized)
			return
		}
		// Parse string thành số nguyên Timestamp
		loginTimestamp, _ := strconv.ParseInt(timeCookie.Value, 10, 64)
		loginTime := time.Unix(loginTimestamp, 0)

		// 3. KIỂM TRA QUÁ THỜI GIAN (VD: Quá 10 phút)
		if time.Since(loginTime) > 10*time.Minute {
			http.Error(w, "Phiên làm việc đã hết hạn (Quá 10 phút). Vui lòng đăng nhập lại!", http.StatusUnauthorized)
			return
		}

		// Nếu hợp lệ -> Kiểm tra User trong DB rồi cho qua
		var user User
		if err := gormDB.Where("username = ?", userCookie.Value).First(&user).Error; err != nil {
			http.Error(w, "Tài khoản không tồn tại!", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), contextUsername, userCookie.Value)
		ctx = context.WithValue(ctx, contextPermissions, user.Permissions)
		newtime := fmt.Sprintf("%d", time.Now().Unix())

		http.SetCookie(w, &http.Cookie{
			Name:     "login_time",
			Value:    newtime,
			Path:     "/",
			HttpOnly: true,
		})
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// 2. AuthorizePermission: Kiểm tra quyền dựa trên đường dẫn API
func AuthorizePermission(permission string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {

			role, _ := r.Context().Value(contextUsername).(string)
			if role == "admin" {
				next.ServeHTTP(w, r)
				return
			}
			permissions, _ := r.Context().Value(contextPermissions).(string)

			if strings.Contains(permissions, permission) {
				next.ServeHTTP(w, r)
				return
			}

			respondJSON(w, http.StatusForbidden,
				map[string]string{"error": "Permission denied"})
		}
	}
}

// --- HANDLERS ---
func POST_login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Định dạng JSON không hợp lệ", http.StatusBadRequest)
		return
	}

	// Tìm user trong Database
	var user User
	result := gormDB.Where("username = ?", strings.ToLower(req.Username)).First(&user)
	if result.Error != nil {
		http.Error(w, "Sai tên đăng nhập!", http.StatusUnauthorized)
		return
	}

	err := bcrypt.CompareHashAndPassword(
		[]byte(user.Password),
		[]byte(req.Password),
	)
	if err != nil {
		http.Error(w, "Sai mật khẩu!", http.StatusUnauthorized)
		return
	}
	ID := uuid.NewString()
	// Đăng nhập thành công -> Set Cookie Session
	http.SetCookie(w, &http.Cookie{
		Name:     "session_user",
		Value:    strings.ToLower(user.Username),
		Path:     "/",
		HttpOnly: true, // Bảo mật, chống JS độc hại lấy cookie.
	})
	// Lưu thời gian đăng nhập hiện tại (Unix Timestamp)
	loginTime := fmt.Sprintf("%d", time.Now().Unix())

	http.SetCookie(w, &http.Cookie{
		Name:     "login_time",
		Value:    loginTime,
		Path:     "/",
		HttpOnly: true,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    ID,
		Path:     "/",
		HttpOnly: true,
	})
	sessionsMu.Lock()
	sessions[strings.ToLower(user.Username)] = ID
	sessionsMu.Unlock()
	http.Redirect(w, r, "/", http.StatusFound)
}

func POST_CreateUser(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON không hợp lệ", http.StatusBadRequest)
		return
	}

	// 1. Kiểm tra dữ liệu đầu vào
	if req.Username == "" || req.Password == "" {
		http.Error(w, "Tên đăng nhập và mật khẩu không được để trống!", http.StatusBadRequest)
		return
	}

	// 2. Kiểm tra xem Username đã tồn tại trong Database chưa
	var existingUser, parent User
	err := gormDB.Where("username = ?", strings.ToLower(req.Username)).First(&existingUser).Error
	if err == nil || strings.ToLower(req.Username) == `master` {
		http.Error(w, "Tên đăng nhập đã tồn tại!", http.StatusConflict)
		return
	}

	parentName := r.Context().Value(contextUsername).(string)
	err = gormDB.Where("username = ?", parentName).First(&parent).Error
	if err == nil {
		http.Error(w, "Account hiện tại không tồn tại", http.StatusConflict)
		return
	}
	// 4. Tạo record mới trong DB
	pa, _ := hashPassword(req.Password)
	newUser := User{
		Username:    strings.ToLower(req.Username),
		Password:    pa, // Lưu ý: Nếu làm thực tế nên Hash password (bcrypt)
		CreateBy:    parentName + "|" + parent.CreateBy,
		Permissions: "",
	}

	if err := gormDB.Create(&newUser).Error; err != nil {
		http.Error(w, "Lỗi khi lưu vào Database: "+err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusCreated, map[string]any{
		"message":     "Tạo tài khoản thành công",
		"username":    req.Username,
		"permissions": "",
	})
}

func GET_CreateUser(w http.ResponseWriter, r *http.Request) {
	file := filepath.Join(config.WebPathDir, "ApiWeb", "Register.html")
	data, err := os.ReadFile(file)
	if err != nil {
		http.Error(w, file+" fail", http.StatusNotFound)
		return
	}
	w.Write(data)
}

func GET_login(w http.ResponseWriter, r *http.Request) {
	ret := r.URL.Query().Get("return")
	tmpl, _ := template.ParseFiles(filepath.Join(config.WebPathDir, "ApiWeb", "login.html"))
	tmpl.Execute(w, map[string]any{
		"Return": ret,
	})
}

func POST_logout(w http.ResponseWriter, r *http.Request) {
	// Xóa Cookie bằng cách set Hạn sử dụng về quá khứ (Negative MaxAge)
	http.SetCookie(w, &http.Cookie{
		Name:     "session_user",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
	http.Redirect(w, r, "/login", http.StatusFound)
}

func GET_meCheck(w http.ResponseWriter, r *http.Request) {
	name := r.Context().Value(contextUsername).(string)
	respondJSON(w, http.StatusOK, map[string]string{
		"name": name,
	})
}

func GET_permissions(w http.ResponseWriter, r *http.Request) {
	var users []User
	// Lấy tất cả user trong database
	if err := gormDB.Find(&users).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	userName := r.Context().Value(contextUsername).(string)
	currentPerm := r.Context().Value(contextPermissions).(string)
	j := make(map[string][]string)

	for _, datas := range users {
		listParent := strings.Split(datas.CreateBy, "|")
		if slices.Contains(listParent, userName) {
			var perms []string
			json.Unmarshal([]byte(datas.Permissions), &perms) // Unmarshal JSON String thành []string
			j[datas.Username] = perms
		}
	}
	var perms []string
	json.Unmarshal([]byte(currentPerm), &perms)
	j[`master`] = perms
	respondJSON(w, http.StatusOK, j)
}

func GET_webPermission(w http.ResponseWriter, r *http.Request) {
	file := filepath.Join(config.WebPathDir, "ApiWeb", `webPermission.html`)
	data, err := os.ReadFile(file)
	if err != nil {
		http.Error(w, file+" fail", http.StatusNotFound)
		return
	}
	w.Write(data)
}

type UpdatePermissionsReq struct {
	TargetUser  string   `json:"target_user"` // User được Admin chọn để sửa quyền
	Permissions []string `json:"permissions"` // Mảng []string quyền mới
}

func PUT_permissions(w http.ResponseWriter, r *http.Request) {
	var req UpdatePermissionsReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	parent := r.Context().Value(contextUsername).(string)
	var user User
	if err := gormDB.Where("username = ?", req.TargetUser).First(&user).Error; err != nil {
		http.Error(w, "Tài khoản không tồn tại!", http.StatusNotFound)
		return
	}

	CreateByDB := user.CreateBy
	if !strings.Contains(CreateByDB, parent) {
		http.Error(w, "Tài khoản không có quyền hạn!", http.StatusUnauthorized)
		return
	}
	err := updatePermsGORM(req.TargetUser, req.Permissions)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// Dynamic request struct dùng chung
type ActionUserReq struct {
	TargetUser string `json:"target_user"`
}

type ActionPassReq struct {
	TargetPass string `json:"target_pass"`
}

func PUT_changePassword(w http.ResponseWriter, r *http.Request) {
	var req ActionPassReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	user := r.Context().Value(contextUsername).(string)
	err := changePassGORM(user, req.TargetPass)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// 1. API Reset Password
func POST_reset_password(w http.ResponseWriter, r *http.Request) {
	var req ActionUserReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	var user User
	result := gormDB.Where("username = ?", req.TargetUser).First(&user)

	if result.Error != nil {
		http.Error(w, "Tài khoản không tồn tại", http.StatusNotFound)
		return
	}
	parent := r.Context().Value(contextUsername).(string)

	CreateByDB := user.CreateBy
	if !strings.Contains(CreateByDB, parent) {
		http.Error(w, "Tài khoản không có quyền hạn!", http.StatusUnauthorized)
		return
	}
	err := resetPassGORM(req.TargetUser)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Reset mật khẩu thành công về: 123456"))
}

// 2. API Xóa Account
func POST_deleteUser(w http.ResponseWriter, r *http.Request) {
	var req ActionUserReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}
	var user User
	result := gormDB.Where("username = ?", req.TargetUser).First(&user)
	if result != nil {
		http.Error(w, "User không tồn tại", http.StatusNotFound)
		return
	}

	parent := r.Context().Value(contextUsername).(string)

	CreateByDB := user.CreateBy
	if !strings.Contains(CreateByDB, parent) {
		http.Error(w, "Tài khoản không có quyền hạn!", http.StatusUnauthorized)
		return
	}
	// Xóa khỏi DB
	err := deleteUserGORM(req.TargetUser)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Xóa tài khoản thành công"))
}

var gormDB *gorm.DB

func initGORM() {
	var err error
	gormDB, err = gorm.Open(sqlite.Open("app_gorm.db"), &gorm.Config{})
	if err != nil {
		LogInfo(&Logger.MAIN, "Không thể kết nối Database!")
		os.Exit(1)
	}
	// Tự động tạo/cập nhật Bảng theo Struct User
	gormDB.AutoMigrate(&User{})
}

// 1. Cập nhật Quyền
func updatePermsGORM(targetUser string, perms []string) error {
	permsJSON, _ := json.Marshal(perms)
	return gormDB.Model(&User{}).Where("username = ?", targetUser).Update("permissions", string(permsJSON)).Error
}

// 2. Reset Password
func resetPassGORM(targetUser string) error {
	return gormDB.Model(&User{}).Where("username = ?", targetUser).Update("password", "123456").Error
}

// 3. Xóa Account
func deleteUserGORM(targetUser string) error {
	return gormDB.Where("username = ?", targetUser).Delete(&User{}).Error
}

// 2. Reset Password
func changePassGORM(targetUser, password string) error {
	p, _ := hashPassword(password)
	return gormDB.Model(&User{}).Where("username = ?", targetUser).Update("password", p).Error
}
