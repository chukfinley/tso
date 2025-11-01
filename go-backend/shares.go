package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
)

const (
	SambaConfPath = "/etc/samba/smb.conf"
	ShareBasePath = "/srv/samba"
)

func ListSharesHandler(w http.ResponseWriter, r *http.Request) {
	db, err := NewDatabase()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	rows, err := db.Query("SELECT * FROM shares ORDER BY share_name")
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var shares []Share
	for rows.Next() {
		var share Share
		var createdBy sql.NullInt64
		err := rows.Scan(
			&share.ID, &share.ShareName, &share.DisplayName, &share.Path, &share.Comment,
			&share.Browseable, &share.Readonly, &share.GuestOk, &share.CaseSensitive,
			&share.PreserveCase, &share.ShortPreserveCase, &share.ValidUsers, &share.WriteList,
			&share.ReadList, &share.AdminUsers, &share.CreateMask, &share.DirectoryMask,
			&share.ForceUser, &share.ForceGroup, &share.IsActive, &createdBy,
			&share.CreatedAt, &share.UpdatedAt,
		)
		if err != nil {
			continue
		}
		if createdBy.Valid {
			val := int(createdBy.Int64)
			share.CreatedBy = &val
		}
		shares = append(shares, share)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"shares":  shares,
	})
}

func CreateShareHandler(w http.ResponseWriter, r *http.Request) {
	var req Share
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Path == "" {
		req.Path = filepath.Join(ShareBasePath, req.ShareName)
	}

	db, err := NewDatabase()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	user, _ := getCurrentUser(r)
	var createdBy *int
	if user != nil {
		createdBy = &user.ID
	}

	result, err := db.Exec(
		`INSERT INTO shares (share_name, display_name, path, comment, browseable, readonly, guest_ok,
			case_sensitive, preserve_case, short_preserve_case, valid_users, write_list, read_list,
			admin_users, create_mask, directory_mask, force_user, force_group, is_active, created_by)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		req.ShareName, req.DisplayName, req.Path, req.Comment, req.Browseable, req.Readonly,
		req.GuestOk, req.CaseSensitive, req.PreserveCase, req.ShortPreserveCase, req.ValidUsers,
		req.WriteList, req.ReadList, req.AdminUsers, req.CreateMask, req.DirectoryMask,
		req.ForceUser, req.ForceGroup, req.IsActive, createdBy,
	)
	if err != nil {
		http.Error(w, "Failed to create share: "+err.Error(), http.StatusBadRequest)
		return
	}

	id, _ := result.LastInsertId()
	updateSambaConfig(db)
	reloadSamba()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"share_id": id,
	})
}

func GetShareHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	db, err := NewDatabase()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	var share Share
	var createdBy sql.NullInt64
	err = db.QueryRow("SELECT * FROM shares WHERE id = ?", id).Scan(
		&share.ID, &share.ShareName, &share.DisplayName, &share.Path, &share.Comment,
		&share.Browseable, &share.Readonly, &share.GuestOk, &share.CaseSensitive,
		&share.PreserveCase, &share.ShortPreserveCase, &share.ValidUsers, &share.WriteList,
		&share.ReadList, &share.AdminUsers, &share.CreateMask, &share.DirectoryMask,
		&share.ForceUser, &share.ForceGroup, &share.IsActive, &createdBy,
		&share.CreatedAt, &share.UpdatedAt,
	)

	if err != nil {
		http.Error(w, "Share not found", http.StatusNotFound)
		return
	}

	if createdBy.Valid {
		val := int(createdBy.Int64)
		share.CreatedBy = &val
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"share":   share,
	})
}

func UpdateShareHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	db, err := NewDatabase()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	updates := []string{}
	values := []interface{}{}

	allowedFields := map[string]bool{
		"display_name": true, "path": true, "comment": true, "browseable": true,
		"readonly": true, "guest_ok": true, "case_sensitive": true,
		"preserve_case": true, "short_preserve_case": true, "valid_users": true,
		"write_list": true, "read_list": true, "admin_users": true,
		"create_mask": true, "directory_mask": true, "force_user": true,
		"force_group": true, "is_active": true,
	}

	for field, val := range req {
		if allowedFields[field] {
			updates = append(updates, field+" = ?")
			values = append(values, val)
		}
	}

	if len(updates) == 0 {
		json.NewEncoder(w).Encode(map[string]bool{"success": true})
		return
	}

	values = append(values, id)
	query := "UPDATE shares SET " + strings.Join(updates, ", ") + " WHERE id = ?"
	_, err = db.Exec(query, values...)
	if err != nil {
		http.Error(w, "Failed to update share", http.StatusBadRequest)
		return
	}

	updateSambaConfig(db)
	reloadSamba()

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func DeleteShareHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	db, err := NewDatabase()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	_, err = db.Exec("DELETE FROM shares WHERE id = ?", id)
	if err != nil {
		http.Error(w, "Failed to delete share", http.StatusBadRequest)
		return
	}

	updateSambaConfig(db)
	reloadSamba()

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func ToggleShareHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	db, err := NewDatabase()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	var isActive bool
	err = db.QueryRow("SELECT is_active FROM shares WHERE id = ?", id).Scan(&isActive)
	if err != nil {
		http.Error(w, "Share not found", http.StatusNotFound)
		return
	}

	_, err = db.Exec("UPDATE shares SET is_active = ? WHERE id = ?", !isActive, id)
	if err != nil {
		http.Error(w, "Failed to toggle share", http.StatusBadRequest)
		return
	}

	updateSambaConfig(db)
	reloadSamba()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"is_active": !isActive,
	})
}

func GetSharePermissionsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shareID, _ := strconv.Atoi(vars["id"])

	db, err := NewDatabase()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	rows, err := db.Query(
		`SELECT sp.*, su.username, su.full_name 
		 FROM share_permissions sp
		 JOIN share_users su ON sp.share_user_id = su.id
		 WHERE sp.share_id = ?
		 ORDER BY su.username`,
		shareID,
	)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var permissions []map[string]interface{}
	for rows.Next() {
		var perm SharePermission
		var username, fullName string
		var createdBy sql.NullInt64
		err := rows.Scan(
			&perm.ID, &perm.ShareID, &perm.ShareUserID, &perm.PermissionLevel,
			&createdBy, &perm.CreatedAt, &perm.UpdatedAt, &username, &fullName,
		)
		if err != nil {
			continue
		}
		permissions = append(permissions, map[string]interface{}{
			"id":               perm.ID,
			"share_id":         perm.ShareID,
			"share_user_id":   perm.ShareUserID,
			"permission_level": perm.PermissionLevel,
			"username":         username,
			"full_name":        fullName,
		})
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"permissions": permissions,
	})
}

func SetSharePermissionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shareID, _ := strconv.Atoi(vars["id"])

	var req struct {
		UserID          int    `json:"user_id"`
		PermissionLevel string `json:"permission_level"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	db, err := NewDatabase()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	user, _ := getCurrentUser(r)
	var createdBy *int
	if user != nil {
		createdBy = &user.ID
	}

	_, err = db.Exec(
		`INSERT INTO share_permissions (share_id, share_user_id, permission_level, created_by)
		 VALUES (?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE permission_level = VALUES(permission_level)`,
		shareID, req.UserID, req.PermissionLevel, createdBy,
	)
	if err != nil {
		http.Error(w, "Failed to set permission", http.StatusBadRequest)
		return
	}

	updateSharePermissions(db, shareID)
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func RemoveSharePermissionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shareID, _ := strconv.Atoi(vars["id"])
	userID, _ := strconv.Atoi(vars["userId"])

	db, err := NewDatabase()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	_, err = db.Exec(
		"DELETE FROM share_permissions WHERE share_id = ? AND share_user_id = ?",
		shareID, userID,
	)
	if err != nil {
		http.Error(w, "Failed to remove permission", http.StatusBadRequest)
		return
	}

	updateSharePermissions(db, shareID)
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func ListShareUsersHandler(w http.ResponseWriter, r *http.Request) {
	db, err := NewDatabase()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	rows, err := db.Query("SELECT * FROM share_users ORDER BY username")
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []ShareUser
	for rows.Next() {
		var user ShareUser
		var createdBy sql.NullInt64
		err := rows.Scan(
			&user.ID, &user.Username, &user.FullName, &user.PasswordHash,
			&user.IsActive, &createdBy, &user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			continue
		}
		if createdBy.Valid {
			val := int(createdBy.Int64)
			user.CreatedBy = &val
		}
		users = append(users, user)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"users":   users,
	})
}

func CreateShareUserHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		FullName string `json:"full_name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	db, err := NewDatabase()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	user, _ := getCurrentUser(r)
	var createdBy *int
	if user != nil {
		createdBy = &user.ID
	}

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)

	result, err := db.Exec(
		"INSERT INTO share_users (username, full_name, password_hash, is_active, created_by) VALUES (?, ?, ?, ?, ?)",
		req.Username, req.FullName, hashedPassword, true, createdBy,
	)
	if err != nil {
		http.Error(w, "Failed to create user", http.StatusBadRequest)
		return
	}

	id, _ := result.LastInsertId()

	// Create Samba user
	createSambaUser(req.Username, req.Password)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"user_id": id,
	})
}

func GetShareUserHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	db, err := NewDatabase()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	var user ShareUser
	var createdBy sql.NullInt64
	err = db.QueryRow("SELECT * FROM share_users WHERE id = ?", id).Scan(
		&user.ID, &user.Username, &user.FullName, &user.PasswordHash,
		&user.IsActive, &createdBy, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	if createdBy.Valid {
		val := int(createdBy.Int64)
		user.CreatedBy = &val
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"user":    user,
	})
}

func UpdateShareUserPasswordHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	var req struct {
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	db, err := NewDatabase()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	var username string
	err = db.QueryRow("SELECT username FROM share_users WHERE id = ?", id).Scan(&username)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	_, err = db.Exec("UPDATE share_users SET password_hash = ? WHERE id = ?", hashedPassword, id)
	if err != nil {
		http.Error(w, "Failed to update password", http.StatusBadRequest)
		return
	}

	updateSambaPassword(username, req.Password)

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func DeleteShareUserHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	db, err := NewDatabase()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	var username string
	err = db.QueryRow("SELECT username FROM share_users WHERE id = ?", id).Scan(&username)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	_, err = db.Exec("DELETE FROM share_users WHERE id = ?", id)
	if err != nil {
		http.Error(w, "Failed to delete user", http.StatusBadRequest)
		return
	}

	deleteSambaUser(username)

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func ToggleShareUserHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	db, err := NewDatabase()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	var isActive bool
	var username string
	err = db.QueryRow("SELECT is_active, username FROM share_users WHERE id = ?", id).Scan(&isActive, &username)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	_, err = db.Exec("UPDATE share_users SET is_active = ? WHERE id = ?", !isActive, id)
	if err != nil {
		http.Error(w, "Failed to toggle user", http.StatusBadRequest)
		return
	}

	toggleSambaUser(username, !isActive)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"is_active": !isActive,
	})
}

func TestSambaConfigHandler(w http.ResponseWriter, r *http.Request) {
	cmd := exec.Command("sudo", "testparm", "-s", SambaConfPath)
	output, err := cmd.CombinedOutput()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": err == nil,
		"output":  string(output),
	})
}

func GetSambaStatusHandler(w http.ResponseWriter, r *http.Request) {
	cmd := exec.Command("sudo", "systemctl", "is-active", "smbd")
	output, _ := cmd.Output()
	running := strings.TrimSpace(string(output)) == "active"

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"status": map[string]interface{}{
			"running": running,
			"status":  map[bool]string{true: "running", false: "stopped"}[running],
		},
	})
}

func GetConnectedClientsHandler(w http.ResponseWriter, r *http.Request) {
	cmd := exec.Command("sudo", "smbstatus", "-b")
	_, _ = cmd.Output()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"clients": []interface{}{}, // Parse output if needed
	})
}

func GetShareLogsHandler(w http.ResponseWriter, r *http.Request) {
	shareID := r.URL.Query().Get("share_id")
	limit := r.URL.Query().Get("limit")
	if limit == "" {
		limit = "100"
	}

	db, err := NewDatabase()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	var rows *sql.Rows
	if shareID != "" {
		sid, _ := strconv.Atoi(shareID)
		rows, err = db.Query(
			`SELECT sal.*, s.share_name 
			 FROM share_access_log sal
			 JOIN shares s ON sal.share_id = s.id
			 WHERE sal.share_id = ?
			 ORDER BY sal.created_at DESC
			 LIMIT ?`,
			sid, limit,
		)
	} else {
		rows, err = db.Query(
			`SELECT sal.*, s.share_name 
			 FROM share_access_log sal
			 LEFT JOIN shares s ON sal.share_id = s.id
			 ORDER BY sal.created_at DESC
			 LIMIT ?`,
			limit,
		)
	}

	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var logs []map[string]interface{}
	for rows.Next() {
		var log map[string]interface{} = make(map[string]interface{})
		// Scan log fields
		logs = append(logs, log)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"logs":    logs,
	})
}

// Helper functions

func updateSambaConfig(db *Database) {
	rows, _ := db.Query("SELECT * FROM shares WHERE is_active = 1")
	shares := []Share{}
	for rows.Next() {
		var share Share
		// Scan share
		shares = append(shares, share)
	}
	rows.Close()

	config := `[global]
   workgroup = WORKGROUP
   server string = Samba Server
   netbios name = SERVER
   security = user
   map to guest = bad user
   dns proxy = no

`

	for _, share := range shares {
		config += fmt.Sprintf("[%s]\n", share.ShareName)
		config += fmt.Sprintf("   comment = %s\n", share.Comment)
		config += fmt.Sprintf("   path = %s\n", share.Path)
		config += fmt.Sprintf("   browseable = %s\n", map[bool]string{true: "yes", false: "no"}[share.Browseable])
		config += fmt.Sprintf("   read only = %s\n", map[bool]string{true: "yes", false: "no"}[share.Readonly])
		config += fmt.Sprintf("   guest ok = %s\n", map[bool]string{true: "yes", false: "no"}[share.GuestOk])
		if share.ValidUsers != "" {
			config += fmt.Sprintf("   valid users = %s\n", share.ValidUsers)
		}
		if share.WriteList != "" {
			config += fmt.Sprintf("   write list = %s\n", share.WriteList)
		}
		config += "\n"
	}

	// Write to temp file and move
	exec.Command("sudo", "bash", "-c", fmt.Sprintf("echo '%s' > /tmp/smb.conf && sudo mv /tmp/smb.conf %s", config, SambaConfPath)).Run()
}

func reloadSamba() {
	exec.Command("sudo", "systemctl", "reload", "smbd").Run()
}

func updateSharePermissions(db *Database, shareID int) {
	rows, _ := db.Query(
		`SELECT su.username, sp.permission_level
		 FROM share_permissions sp
		 JOIN share_users su ON sp.share_user_id = su.id
		 WHERE sp.share_id = ?`,
		shareID,
	)
	defer rows.Close()

	validUsers := []string{}
	readList := []string{}
	writeList := []string{}
	adminUsers := []string{}

	for rows.Next() {
		var username, level string
		rows.Scan(&username, &level)
		validUsers = append(validUsers, username)
		if level == "admin" {
			adminUsers = append(adminUsers, username)
			writeList = append(writeList, username)
		} else if level == "write" {
			writeList = append(writeList, username)
		} else {
			readList = append(readList, username)
		}
	}

	db.Exec(
		"UPDATE shares SET valid_users = ?, read_list = ?, write_list = ?, admin_users = ? WHERE id = ?",
		strings.Join(validUsers, ","), strings.Join(readList, ","),
		strings.Join(writeList, ","), strings.Join(adminUsers, ","), shareID,
	)

	updateSambaConfig(db)
	reloadSamba()
}

func createSambaUser(username, password string) {
	exec.Command("sudo", "useradd", "-M", "-s", "/sbin/nologin", username).Run()
	cmd := exec.Command("sudo", "smbpasswd", "-a", "-s", username)
	cmd.Stdin = strings.NewReader(password + "\n" + password + "\n")
	cmd.Run()
}

func updateSambaPassword(username, password string) {
	cmd := exec.Command("sudo", "smbpasswd", "-s", username)
	cmd.Stdin = strings.NewReader(password + "\n" + password + "\n")
	cmd.Run()
}

func deleteSambaUser(username string) {
	exec.Command("sudo", "smbpasswd", "-x", username).Run()
	exec.Command("sudo", "userdel", username).Run()
}

func toggleSambaUser(username string, enable bool) {
	flag := "-d"
	if enable {
		flag = "-e"
	}
	exec.Command("sudo", "smbpasswd", flag, username).Run()
}

