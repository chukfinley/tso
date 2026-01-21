package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

type DashboardConfig struct {
	Layout []WidgetConfig `json:"layout"`
}

type WidgetConfig struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Visible bool   `json:"visible"`
	Order   int    `json:"order"`
}

func GetDashboardConfigHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	userID, ok := session.Values["user_id"].(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	db, err := NewDatabase()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	var layoutJSON sql.NullString
	err = db.QueryRow("SELECT layout FROM dashboard_config WHERE user_id = ?", userID).Scan(&layoutJSON)
	if err != nil && err != sql.ErrNoRows {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	config := getDefaultDashboardConfig()
	if layoutJSON.Valid && layoutJSON.String != "" {
		json.Unmarshal([]byte(layoutJSON.String), &config)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"config":  config,
	})
}

func SaveDashboardConfigHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	userID, ok := session.Values["user_id"].(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var config DashboardConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	db, err := NewDatabase()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	layoutJSON, err := json.Marshal(config)
	if err != nil {
		http.Error(w, "Failed to encode config", http.StatusInternalServerError)
		return
	}

	// Use INSERT ... ON DUPLICATE KEY UPDATE for upsert
	_, err = db.Exec(`
		INSERT INTO dashboard_config (user_id, layout)
		VALUES (?, ?)
		ON DUPLICATE KEY UPDATE layout = VALUES(layout)
	`, userID, string(layoutJSON))
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
	})
}

func getDefaultDashboardConfig() DashboardConfig {
	return DashboardConfig{
		Layout: []WidgetConfig{
			{ID: "system-overview", Type: "system-overview", Visible: true, Order: 1},
			{ID: "temperature", Type: "temperature", Visible: true, Order: 2},
			{ID: "storage", Type: "storage", Visible: true, Order: 3},
			{ID: "network", Type: "network", Visible: true, Order: 4},
			{ID: "vms", Type: "vms", Visible: true, Order: 5},
			{ID: "logs", Type: "logs", Visible: true, Order: 6},
			{ID: "alerts", Type: "alerts", Visible: true, Order: 7},
		},
	}
}
