package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func GetSystemStatsHandler(w http.ResponseWriter, r *http.Request) {
	stats := map[string]interface{}{
		"cpu":         getCPUInfo(),
		"memory":      getMemoryInfo(),
		"swap":        getSwapInfo(),
		"uptime":      getUptime(),
		"motherboard": getMotherboardInfo(),
		"network":     getNetworkInfo(),
		"timestamp":   time.Now().Unix(),
	}

	json.NewEncoder(w).Encode(stats)
}

func getCPUInfo() map[string]interface{} {
	cmd := exec.Command("lscpu")
	output, _ := cmd.Output()
	lines := strings.Split(string(output), "\n")

	info := map[string]interface{}{
		"model":        "Unknown",
		"cores":        1,
		"architecture": "Unknown",
		"frequency":    "N/A",
		"usage":        0.0,
		"load_avg": map[string]float64{
			"1min":  0.0,
			"5min":  0.0,
			"15min": 0.0,
		},
	}

	for _, line := range lines {
		if strings.Contains(line, "Model name") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				info["model"] = strings.TrimSpace(parts[1])
			}
		}
		if strings.Contains(line, "CPU(s)") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				cores, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
				info["cores"] = cores
			}
		}
	}

	cmd = exec.Command("nproc")
	output, _ = cmd.Output()
	if cores, err := strconv.Atoi(strings.TrimSpace(string(output))); err == nil {
		info["cores"] = cores
	}

	cmd = exec.Command("uname", "-m")
	output, _ = cmd.Output()
	info["architecture"] = strings.TrimSpace(string(output))

	// Get CPU usage
	cmd = exec.Command("top", "-bn1")
	output, _ = cmd.Output()
	// Parse CPU usage from top output

	// Get load average
	loadAvg, _ := ioutil.ReadFile("/proc/loadavg")
	parts := strings.Fields(string(loadAvg))
	if len(parts) >= 3 {
		info["load_avg"] = map[string]float64{
			"1min":  parseFloat(parts[0]),
			"5min":  parseFloat(parts[1]),
			"15min": parseFloat(parts[2]),
		}
	}

	return info
}

func getMemoryInfo() map[string]interface{} {
	memData, _ := ioutil.ReadFile("/proc/meminfo")
	lines := strings.Split(string(memData), "\n")

	info := map[string]int64{}
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			key := strings.TrimSuffix(parts[0], ":")
			value, _ := strconv.ParseInt(parts[1], 10, 64)
			info[key] = value * 1024 // Convert KB to bytes
		}
	}

	total := info["MemTotal"]
	free := info["MemFree"]
	buffers := info["Buffers"]
	cached := info["Cached"]
	available := info["MemAvailable"]
	used := total - free - buffers - cached
	usagePercent := float64(used) / float64(total) * 100

	return map[string]interface{}{
		"total":          total,
		"total_formatted": formatBytes(total),
		"used":           used,
		"used_formatted": formatBytes(used),
		"free":           free,
		"free_formatted": formatBytes(free),
		"available":      available,
		"available_formatted": formatBytes(available),
		"usage_percent":  usagePercent,
	}
}

func getSwapInfo() map[string]interface{} {
	memData, _ := ioutil.ReadFile("/proc/meminfo")
	lines := strings.Split(string(memData), "\n")

	info := map[string]int64{}
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			key := strings.TrimSuffix(parts[0], ":")
			value, _ := strconv.ParseInt(parts[1], 10, 64)
			info[key] = value * 1024
		}
	}

	total := info["SwapTotal"]
	free := info["SwapFree"]
	used := total - free
	usagePercent := float64(0)
	if total > 0 {
		usagePercent = float64(used) / float64(total) * 100
	}

	return map[string]interface{}{
		"total":          total,
		"total_formatted": formatBytes(total),
		"used":           used,
		"used_formatted": formatBytes(used),
		"free":           free,
		"free_formatted": formatBytes(free),
		"usage_percent":  usagePercent,
	}
}

func getUptime() map[string]interface{} {
	uptimeData, _ := ioutil.ReadFile("/proc/uptime")
	parts := strings.Fields(string(uptimeData))
	if len(parts) == 0 {
		return map[string]interface{}{
			"seconds":  0,
			"formatted": "0m",
		}
	}

	seconds, _ := strconv.ParseFloat(parts[0], 64)
	secs := int64(seconds)

	days := secs / 86400
	hours := (secs % 86400) / 3600
	minutes := (secs % 3600) / 60

	parts2 := []string{}
	if days > 0 {
		parts2 = append(parts2, fmt.Sprintf("%dd", days))
	}
	if hours > 0 {
		parts2 = append(parts2, fmt.Sprintf("%dh", hours))
	}
	if minutes > 0 {
		parts2 = append(parts2, fmt.Sprintf("%dm", minutes))
	}

	formatted := "0m"
	if len(parts2) > 0 {
		formatted = strings.Join(parts2, " ")
	}

	return map[string]interface{}{
		"seconds":  secs,
		"formatted": formatted,
	}
}

func getMotherboardInfo() map[string]interface{} {
	vendor, _ := ioutil.ReadFile("/sys/class/dmi/id/board_vendor")
	name, _ := ioutil.ReadFile("/sys/class/dmi/id/board_name")

	return map[string]interface{}{
		"vendor": strings.TrimSpace(string(vendor)),
		"name":   strings.TrimSpace(string(name)),
		"version": "N/A",
	}
}

func getNetworkInfo() []map[string]interface{} {
	cmd := exec.Command("ls", "/sys/class/net/")
	output, _ := cmd.Output()
	interfaces := []string{}
	for _, line := range strings.Split(string(output), "\n") {
		if line != "" && line != "lo" {
			interfaces = append(interfaces, line)
		}
	}

	result := []map[string]interface{}{}
	for _, iface := range interfaces {
		info := map[string]interface{}{
			"name":   iface,
			"status": "Disconnected",
			"is_up": false,
			"type":   "Unknown",
			"ip":     "N/A",
			"mac":    "N/A",
			"speed":  "N/A",
			"rx_bytes": 0,
			"rx_formatted": "0 B",
			"tx_bytes": 0,
			"tx_formatted": "0 B",
		}

		operstate, _ := ioutil.ReadFile(fmt.Sprintf("/sys/class/net/%s/operstate", iface))
		if strings.TrimSpace(string(operstate)) == "up" {
			info["status"] = "Connected"
			info["is_up"] = true
		}

		mac, _ := ioutil.ReadFile(fmt.Sprintf("/sys/class/net/%s/address", iface))
		info["mac"] = strings.TrimSpace(string(mac))

		cmd := exec.Command("ip", "addr", "show", iface)
		output, _ := cmd.Output()
		for _, line := range strings.Split(string(output), "\n") {
			if strings.Contains(line, "inet ") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					ip := strings.Split(parts[1], "/")[0]
					info["ip"] = ip
				}
			}
		}

		rxBytes, _ := ioutil.ReadFile(fmt.Sprintf("/sys/class/net/%s/statistics/rx_bytes", iface))
		txBytes, _ := ioutil.ReadFile(fmt.Sprintf("/sys/class/net/%s/statistics/tx_bytes", iface))
		rx, _ := strconv.ParseInt(strings.TrimSpace(string(rxBytes)), 10, 64)
		tx, _ := strconv.ParseInt(strings.TrimSpace(string(txBytes)), 10, 64)
		info["rx_bytes"] = rx
		info["rx_formatted"] = formatBytes(rx)
		info["tx_bytes"] = tx
		info["tx_formatted"] = formatBytes(tx)

		if strings.Contains(iface, "eth") || strings.Contains(iface, "en") {
			info["type"] = "Ethernet"
		} else if strings.Contains(iface, "wlan") || strings.Contains(iface, "wl") {
			info["type"] = "Wireless"
		}

		result = append(result, info)
	}

	return result
}

func formatBytes(bytes int64) string {
	units := []string{"B", "KB", "MB", "GB", "TB"}
	value := float64(bytes)
	unitIndex := 0

	for value >= 1024 && unitIndex < len(units)-1 {
		value /= 1024
		unitIndex++
	}

	return fmt.Sprintf("%.2f %s", value, units[unitIndex])
}

func parseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func SystemUpdateHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Action string `json:"action"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Execute system update
	cmd := exec.Command("sudo", "apt", "update")
	cmd.Run()

	cmd = exec.Command("sudo", "apt", "upgrade", "-y")
	output, _ := cmd.CombinedOutput()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"output":  string(output),
	})
}

func SystemControlHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Action string `json:"action"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	var cmd *exec.Cmd
	switch req.Action {
	case "reboot":
		cmd = exec.Command("sudo", "reboot")
	case "shutdown":
		cmd = exec.Command("sudo", "shutdown", "-h", "now")
	case "suspend", "sleep":
		cmd = exec.Command("sudo", "systemctl", "suspend")
	case "hibernate":
		cmd = exec.Command("sudo", "systemctl", "hibernate")
	default:
		http.Error(w, "Invalid action", http.StatusBadRequest)
		return
	}

	cmd.Run()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("System %s initiated", req.Action),
	})
}

func ExecuteTerminalHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Command string `json:"command"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	cmd := exec.Command("bash", "-c", req.Command)
	output, err := cmd.CombinedOutput()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": err == nil,
		"output":  string(output),
		"error":   err != nil,
	})
}

func GetLogsHandler(w http.ResponseWriter, r *http.Request) {
	level := r.URL.Query().Get("level")
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

	query := "SELECT sl.*, u.username FROM system_logs sl LEFT JOIN users u ON sl.user_id = u.id"
	if level != "" {
		query += " WHERE sl.level = ?"
		query += " ORDER BY sl.created_at DESC LIMIT ?"
		rows, _ := db.Query(query, level, limit)
		defer rows.Close()
		// Process rows
	} else {
		query += " ORDER BY sl.created_at DESC LIMIT ?"
		rows, _ := db.Query(query, limit)
		defer rows.Close()
		// Process rows
	}

	logs := []map[string]interface{}{}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"logs":    logs,
	})
}

func GetActivityLogsHandler(w http.ResponseWriter, r *http.Request) {
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

	rows, err := db.Query(
		`SELECT al.*, u.username 
		 FROM activity_log al
		 LEFT JOIN users u ON al.user_id = u.id
		 ORDER BY al.created_at DESC
		 LIMIT ?`,
		limit,
	)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	logs := []ActivityLog{}
	for rows.Next() {
		var log ActivityLog
		var userID sql.NullInt64
		err := rows.Scan(
			&log.ID, &userID, &log.Action, &log.Description,
			&log.IPAddress, &log.CreatedAt, &log.Username,
		)
		if err != nil {
			continue
		}
		if userID.Valid {
			val := int(userID.Int64)
			log.UserID = &val
		}
		logs = append(logs, log)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"logs":    logs,
	})
}

