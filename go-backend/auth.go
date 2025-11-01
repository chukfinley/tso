package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
)

var store *sessions.CookieStore

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
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

	var user User
	err = db.QueryRow(
		"SELECT id, username, email, password, full_name, role, is_active FROM users WHERE username = ?",
		req.Username,
	).Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.FullName, &user.Role, &user.IsActive)

	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	if !user.IsActive {
		http.Error(w, "Account is disabled", http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Create session
	session, _ := store.Get(r, "session")
	session.Values["user_id"] = user.ID
	session.Values["username"] = user.Username
	session.Values["role"] = user.Role
	session.Values["full_name"] = user.FullName
	session.Values["login_time"] = time.Now().Unix()
	session.Options.MaxAge = 3600 // 1 hour
	session.Save(r, w)

	// Update last login
	db.Exec("UPDATE users SET last_login = NOW() WHERE id = ?", user.ID)

	// Log activity
	logActivity(db, user.ID, "login", "User logged in", getIPAddress(r))

	user.Password = ""
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"user":    user,
	})
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	
	if userID, ok := session.Values["user_id"].(int); ok {
		db, _ := NewDatabase()
		if db != nil {
			defer db.Close()
			logActivity(db, userID, "logout", "User logged out", getIPAddress(r))
		}
	}

	session.Values = make(map[interface{}]interface{})
	session.Options.MaxAge = -1
	session.Save(r, w)

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func AuthCheckHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	userID, ok := session.Values["user_id"].(int)

	if !ok {
		http.Error(w, "Not authenticated", http.StatusUnauthorized)
		return
	}

	db, err := NewDatabase()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	var user User
	err = db.QueryRow(
		"SELECT id, username, email, full_name, role, is_active FROM users WHERE id = ?",
		userID,
	).Scan(&user.ID, &user.Username, &user.Email, &user.FullName, &user.Role, &user.IsActive)

	if err != nil {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"authenticated": true,
		"user":          user,
	})
}

func RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "session")
		userID, ok := session.Values["user_id"].(int)

		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Store user info in request context
		r.Header.Set("X-User-ID", string(rune(userID)))
		next(w, r)
	}
}

func RequireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "session")
		role, ok := session.Values["role"].(string)

		if !ok || role != "admin" {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		next(w, r)
	}
}

func getCurrentUser(r *http.Request) (*User, error) {
	session, _ := store.Get(r, "session")
	userID, ok := session.Values["user_id"].(int)
	if !ok {
		return nil, http.ErrNoCookie
	}

	db, err := NewDatabase()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	var user User
	err = db.QueryRow(
		"SELECT id, username, email, full_name, role, is_active FROM users WHERE id = ?",
		userID,
	).Scan(&user.ID, &user.Username, &user.Email, &user.FullName, &user.Role, &user.IsActive)

	return &user, err
}

func getIPAddress(r *http.Request) string {
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		return forwarded
	}
	return r.RemoteAddr
}

func logActivity(db *Database, userID int, action, description, ipAddress string) {
	db.Exec(
		"INSERT INTO activity_log (user_id, action, description, ip_address) VALUES (?, ?, ?, ?)",
		userID, action, description, ipAddress,
	)
}

