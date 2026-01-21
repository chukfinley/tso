package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

type AlertRule struct {
	ID            int       `json:"id"`
	Name          string    `json:"name"`
	ConditionType string    `json:"condition_type"` // cpu, memory, disk, temperature
	Threshold     float64   `json:"threshold"`
	Comparison    string    `json:"comparison"` // gt, lt, eq
	Severity      string    `json:"severity"`   // info, warning, critical
	IsActive      bool      `json:"is_active"`
	CreatedBy     *int      `json:"created_by,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type ActiveAlert struct {
	RuleID      int       `json:"rule_id"`
	RuleName    string    `json:"rule_name"`
	Type        string    `json:"type"`
	Severity    string    `json:"severity"`
	CurrentVal  float64   `json:"current_value"`
	Threshold   float64   `json:"threshold"`
	Comparison  string    `json:"comparison"`
	Message     string    `json:"message"`
	TriggeredAt time.Time `json:"triggered_at"`
}

func GetAlertRulesHandler(w http.ResponseWriter, r *http.Request) {
	db, err := NewDatabase()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT id, name, condition_type, threshold, comparison, severity, is_active, created_by, created_at, updated_at
		FROM alert_rules
		ORDER BY created_at DESC
	`)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var rules []AlertRule
	for rows.Next() {
		var rule AlertRule
		err := rows.Scan(&rule.ID, &rule.Name, &rule.ConditionType, &rule.Threshold, &rule.Comparison, &rule.Severity, &rule.IsActive, &rule.CreatedBy, &rule.CreatedAt, &rule.UpdatedAt)
		if err != nil {
			continue
		}
		rules = append(rules, rule)
	}

	if rules == nil {
		rules = []AlertRule{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"rules":   rules,
	})
}

func CreateAlertRuleHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	userID, ok := session.Values["user_id"].(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var rule AlertRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate
	if rule.Name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}
	if rule.ConditionType == "" {
		http.Error(w, "Condition type is required", http.StatusBadRequest)
		return
	}

	validConditions := map[string]bool{
		"cpu": true, "memory": true, "disk": true, "temperature": true, "swap": true,
	}
	if !validConditions[rule.ConditionType] {
		http.Error(w, "Invalid condition type", http.StatusBadRequest)
		return
	}

	validComparisons := map[string]bool{"gt": true, "lt": true, "eq": true}
	if rule.Comparison == "" {
		rule.Comparison = "gt"
	}
	if !validComparisons[rule.Comparison] {
		http.Error(w, "Invalid comparison", http.StatusBadRequest)
		return
	}

	validSeverities := map[string]bool{"info": true, "warning": true, "critical": true}
	if rule.Severity == "" {
		rule.Severity = "warning"
	}
	if !validSeverities[rule.Severity] {
		http.Error(w, "Invalid severity", http.StatusBadRequest)
		return
	}

	db, err := NewDatabase()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	result, err := db.Exec(`
		INSERT INTO alert_rules (name, condition_type, threshold, comparison, severity, is_active, created_by)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, rule.Name, rule.ConditionType, rule.Threshold, rule.Comparison, rule.Severity, true, userID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	ruleID, _ := result.LastInsertId()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"rule_id": ruleID,
	})
}

func UpdateAlertRuleHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ruleID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid rule ID", http.StatusBadRequest)
		return
	}

	var rule AlertRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	db, err := NewDatabase()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	result, err := db.Exec(`
		UPDATE alert_rules
		SET name = ?, condition_type = ?, threshold = ?, comparison = ?, severity = ?, is_active = ?
		WHERE id = ?
	`, rule.Name, rule.ConditionType, rule.Threshold, rule.Comparison, rule.Severity, rule.IsActive, ruleID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Rule not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
	})
}

func DeleteAlertRuleHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ruleID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid rule ID", http.StatusBadRequest)
		return
	}

	db, err := NewDatabase()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	result, err := db.Exec("DELETE FROM alert_rules WHERE id = ?", ruleID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Rule not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
	})
}

func GetActiveAlertsHandler(w http.ResponseWriter, r *http.Request) {
	db, err := NewDatabase()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Get all active rules
	rows, err := db.Query(`
		SELECT id, name, condition_type, threshold, comparison, severity
		FROM alert_rules
		WHERE is_active = TRUE
	`)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var activeAlerts []ActiveAlert
	now := time.Now()

	// Get current system values
	stats := getCurrentSystemValues()

	for rows.Next() {
		var rule AlertRule
		err := rows.Scan(&rule.ID, &rule.Name, &rule.ConditionType, &rule.Threshold, &rule.Comparison, &rule.Severity)
		if err != nil {
			continue
		}

		// Get current value for this condition
		currentValue, ok := stats[rule.ConditionType]
		if !ok {
			continue
		}

		// Check if alert is triggered
		triggered := false
		switch rule.Comparison {
		case "gt":
			triggered = currentValue > rule.Threshold
		case "lt":
			triggered = currentValue < rule.Threshold
		case "eq":
			triggered = currentValue == rule.Threshold
		}

		if triggered {
			message := formatAlertMessage(rule.ConditionType, rule.Comparison, currentValue, rule.Threshold)
			activeAlerts = append(activeAlerts, ActiveAlert{
				RuleID:      rule.ID,
				RuleName:    rule.Name,
				Type:        rule.ConditionType,
				Severity:    rule.Severity,
				CurrentVal:  currentValue,
				Threshold:   rule.Threshold,
				Comparison:  rule.Comparison,
				Message:     message,
				TriggeredAt: now,
			})
		}
	}

	if activeAlerts == nil {
		activeAlerts = []ActiveAlert{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"alerts":  activeAlerts,
	})
}

func getCurrentSystemValues() map[string]float64 {
	values := make(map[string]float64)

	// Get CPU usage from load average
	cpuInfo := getCPUInfo()
	if loadAvg, ok := cpuInfo["load_avg"].(map[string]float64); ok {
		if cores, ok := cpuInfo["cores"].(int); ok && cores > 0 {
			// Convert load average to percentage (load / cores * 100)
			values["cpu"] = loadAvg["1min"] / float64(cores) * 100
			if values["cpu"] > 100 {
				values["cpu"] = 100
			}
		}
	}

	// Get memory usage
	memInfo := getMemoryInfo()
	if usagePercent, ok := memInfo["usage_percent"].(float64); ok {
		values["memory"] = usagePercent
	}

	// Get swap usage
	swapInfo := getSwapInfo()
	if usagePercent, ok := swapInfo["usage_percent"].(float64); ok {
		values["swap"] = usagePercent
	}

	// Get disk usage (root partition)
	partitions := getMountedPartitions()
	for _, p := range partitions {
		if p.MountPoint == "/" {
			values["disk"] = p.UsagePercent
			break
		}
	}

	// Get max temperature
	tempInfo := readThermalZones()
	maxTemp := 0.0
	for _, t := range tempInfo {
		if t.Temperature > maxTemp {
			maxTemp = t.Temperature
		}
	}
	hwmonInfo := readHwmon()
	for _, t := range hwmonInfo {
		if t.Temperature > maxTemp {
			maxTemp = t.Temperature
		}
	}
	values["temperature"] = maxTemp

	return values
}

func formatAlertMessage(conditionType, comparison string, current, threshold float64) string {
	comparisonText := ""
	switch comparison {
	case "gt":
		comparisonText = "exceeded"
	case "lt":
		comparisonText = "fell below"
	case "eq":
		comparisonText = "equals"
	}

	unit := "%"
	if conditionType == "temperature" {
		unit = "°C"
	}

	typeText := conditionType
	switch conditionType {
	case "cpu":
		typeText = "CPU usage"
	case "memory":
		typeText = "Memory usage"
	case "disk":
		typeText = "Disk usage"
	case "swap":
		typeText = "Swap usage"
	case "temperature":
		typeText = "Temperature"
	}

	return typeText + " " + comparisonText + " threshold: " + strconv.FormatFloat(current, 'f', 1, 64) + unit + " (threshold: " + strconv.FormatFloat(threshold, 'f', 1, 64) + unit + ")"
}
