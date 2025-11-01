package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
)

func ListUsersHandler(w http.ResponseWriter, r *http.Request) {
	db, err := NewDatabase()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	rows, err := db.Query(
		"SELECT id, username, email, full_name, role, is_active, created_at, last_login FROM users ORDER BY created_at DESC",
	)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		var lastLogin sql.NullTime
		err := rows.Scan(
			&user.ID, &user.Username, &user.Email, &user.FullName,
			&user.Role, &user.IsActive, &user.CreatedAt, &lastLogin,
		)
		if err != nil {
			continue
		}
		if lastLogin.Valid {
			user.LastLogin = &lastLogin.Time
		}
		users = append(users, user)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"users":   users,
	})
}

func CreateUserHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
		FullName string `json:"full_name"`
		Role     string `json:"role"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Role == "" {
		req.Role = "user"
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}

	db, err := NewDatabase()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	result, err := db.Exec(
		"INSERT INTO users (username, email, password, full_name, role) VALUES (?, ?, ?, ?, ?)",
		req.Username, req.Email, hashedPassword, req.FullName, req.Role,
	)
	if err != nil {
		http.Error(w, "Failed to create user", http.StatusBadRequest)
		return
	}

	id, _ := result.LastInsertId()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"user_id": id,
	})
}

func GetUserHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	db, err := NewDatabase()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	var user User
	err = db.QueryRow(
		"SELECT id, username, email, full_name, role, is_active, created_at, updated_at, last_login FROM users WHERE id = ?",
		id,
	).Scan(&user.ID, &user.Username, &user.Email, &user.FullName, &user.Role, &user.IsActive,
		&user.CreatedAt, &user.UpdatedAt, &user.LastLogin)

	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"user":    user,
	})
}

func UpdateUserHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	if id == 1 {
		http.Error(w, "Cannot modify admin user", http.StatusForbidden)
		return
	}

	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		FullName string `json:"full_name"`
		Role     string `json:"role"`
		IsActive bool   `json:"is_active"`
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

	_, err = db.Exec(
		"UPDATE users SET username = ?, email = ?, full_name = ?, role = ?, is_active = ? WHERE id = ?",
		req.Username, req.Email, req.FullName, req.Role, req.IsActive, id,
	)
	if err != nil {
		http.Error(w, "Failed to update user", http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func DeleteUserHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	if id == 1 {
		http.Error(w, "Cannot delete admin user", http.StatusForbidden)
		return
	}

	db, err := NewDatabase()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	_, err = db.Exec("DELETE FROM users WHERE id = ?", id)
	if err != nil {
		http.Error(w, "Failed to delete user", http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func UpdatePasswordHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	var req struct {
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Check if user can update this password
	session, _ := store.Get(r, "session")
	currentUserID, _ := session.Values["user_id"].(int)
	currentRole, _ := session.Values["role"].(string)

	if currentRole != "admin" && currentUserID != id {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}

	db, err := NewDatabase()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	_, err = db.Exec("UPDATE users SET password = ? WHERE id = ?", hashedPassword, id)
	if err != nil {
		http.Error(w, "Failed to update password", http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func GetProfileHandler(w http.ResponseWriter, r *http.Request) {
	user, err := getCurrentUser(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"user":    user,
	})
}

func UpdateProfileHandler(w http.ResponseWriter, r *http.Request) {
	user, err := getCurrentUser(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Email    string `json:"email"`
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

	_, err = db.Exec(
		"UPDATE users SET email = ?, full_name = ? WHERE id = ?",
		req.Email, req.FullName, user.ID,
	)
	if err != nil {
		http.Error(w, "Failed to update profile", http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

