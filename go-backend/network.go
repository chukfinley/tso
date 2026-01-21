package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/mux"
)

// Store previous bandwidth readings for calculating speed
var (
	prevBandwidth       = make(map[string]bandwidthReading)
	prevBandwidthLock   sync.RWMutex
	sessionStart        = time.Now()
	sessionBaseline     = make(map[string]bandwidthReading)
	sessionBaselineLock sync.RWMutex

	// Bandwidth history for charts (last 60 readings = ~1 minute at 1s intervals)
	bandwidthHistory     []BandwidthHistoryEntry
	bandwidthHistoryLock sync.RWMutex
	maxHistorySize       = 120 // 2 minutes of history

	// Process bandwidth tracking
	prevProcessBandwidth     = make(map[int]processBandwidthReading)
	prevProcessBandwidthLock sync.RWMutex

	// Per-process bandwidth history
	processHistory          = make(map[int][]ProcessHistoryEntry)
	processHistoryLock      sync.RWMutex
	maxProcessHistorySize   = 120 // 2 minutes of history per process
	maxTrackedProcesses     = 100 // Limit number of tracked processes

	// Network event log
	networkEvents     []NetworkEvent
	networkEventsLock sync.RWMutex
	maxNetworkEvents  = 500
)

type bandwidthReading struct {
	RxBytes   int64
	TxBytes   int64
	Timestamp time.Time
}

type processBandwidthReading struct {
	RxBytes   int64
	TxBytes   int64
	Timestamp time.Time
}

type ProcessHistoryEntry struct {
	Timestamp time.Time `json:"timestamp"`
	RxSpeed   float64   `json:"rx_speed"`
	TxSpeed   float64   `json:"tx_speed"`
	RxBytes   int64     `json:"rx_bytes"`
	TxBytes   int64     `json:"tx_bytes"`
}

type ProcessDetails struct {
	PID         int                   `json:"pid"`
	Name        string                `json:"name"`
	Cmdline     string                `json:"cmdline"`
	User        string                `json:"user"`
	State       string                `json:"state"`
	Threads     int                   `json:"threads"`
	MemoryRSS   int64                 `json:"memory_rss"`
	MemoryVMS   int64                 `json:"memory_vms"`
	CPUPercent  float64               `json:"cpu_percent"`
	StartTime   string                `json:"start_time"`
	Connections int                   `json:"connections"`
	RxBytes     int64                 `json:"rx_bytes"`
	TxBytes     int64                 `json:"tx_bytes"`
	RxSpeed     float64               `json:"rx_speed"`
	TxSpeed     float64               `json:"tx_speed"`
	History     []ProcessHistoryEntry `json:"history"`
}

type BandwidthHistoryEntry struct {
	Timestamp   time.Time          `json:"timestamp"`
	TotalRx     float64            `json:"total_rx_speed"`
	TotalTx     float64            `json:"total_tx_speed"`
	Interfaces  map[string]float64 `json:"interfaces_rx"`
	InterfacesTx map[string]float64 `json:"interfaces_tx"`
}

type NetworkEvent struct {
	ID          int       `json:"id"`
	Timestamp   time.Time `json:"timestamp"`
	Type        string    `json:"type"`
	Interface   string    `json:"interface"`
	Description string    `json:"description"`
	Details     string    `json:"details"`
}

type NetworkInterface struct {
	Name          string  `json:"name"`
	DisplayName   string  `json:"display_name"`
	Type          string  `json:"type"`
	Status        string  `json:"status"`
	IsUp          bool    `json:"is_up"`
	IP            string  `json:"ip"`
	IPv6          string  `json:"ipv6"`
	Subnet        string  `json:"subnet"`
	Gateway       string  `json:"gateway"`
	MAC           string  `json:"mac"`
	MTU           int     `json:"mtu"`
	Speed         string  `json:"speed"`
	Duplex        string  `json:"duplex"`
	RxBytes       int64   `json:"rx_bytes"`
	TxBytes       int64   `json:"tx_bytes"`
	RxFormatted   string  `json:"rx_formatted"`
	TxFormatted   string  `json:"tx_formatted"`
	RxSession     int64   `json:"rx_session"`
	TxSession     int64   `json:"tx_session"`
	RxSessionFmt  string  `json:"rx_session_formatted"`
	TxSessionFmt  string  `json:"tx_session_formatted"`
	RxSpeed       float64 `json:"rx_speed"`
	TxSpeed       float64 `json:"tx_speed"`
	RxSpeedFmt    string  `json:"rx_speed_formatted"`
	TxSpeedFmt    string  `json:"tx_speed_formatted"`
	RxPackets     int64   `json:"rx_packets"`
	TxPackets     int64   `json:"tx_packets"`
	RxErrors      int64   `json:"rx_errors"`
	TxErrors      int64   `json:"tx_errors"`
	RxDropped     int64   `json:"rx_dropped"`
	TxDropped     int64   `json:"tx_dropped"`
	Driver        string  `json:"driver"`
	IsVirtual     bool    `json:"is_virtual"`
	IsWireless    bool    `json:"is_wireless"`
}

type NetworkProcess struct {
	PID         int     `json:"pid"`
	Name        string  `json:"name"`
	User        string  `json:"user"`
	Connections int     `json:"connections"`
	RxBytes     int64   `json:"rx_bytes"`
	TxBytes     int64   `json:"tx_bytes"`
	RxSpeed     float64 `json:"rx_speed"`
	TxSpeed     float64 `json:"tx_speed"`
	RxSpeedFmt  string  `json:"rx_speed_formatted"`
	TxSpeedFmt  string  `json:"tx_speed_formatted"`
}

// ListNetworkInterfacesHandler returns all network interfaces with detailed info
func ListNetworkInterfacesHandler(w http.ResponseWriter, r *http.Request) {
	interfaces := getDetailedNetworkInterfaces()
	json.NewEncoder(w).Encode(map[string]any{
		"success":      true,
		"interfaces":   interfaces,
		"session_start": sessionStart.Format(time.RFC3339),
	})
}

// GetNetworkInterfaceHandler returns a single interface
func GetNetworkInterfaceHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	interfaces := getDetailedNetworkInterfaces()
	for _, iface := range interfaces {
		if iface.Name == name {
			json.NewEncoder(w).Encode(map[string]any{
				"success":   true,
				"interface": iface,
			})
			return
		}
	}

	http.Error(w, "Interface not found", http.StatusNotFound)
}

// ToggleNetworkInterfaceHandler enables or disables a network interface
func ToggleNetworkInterfaceHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	var req struct {
		Action string `json:"action"` // "up" or "down"
		Force  bool   `json:"force"`  // Force even if it's the active connection
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Action != "up" && req.Action != "down" {
		http.Error(w, "Action must be 'up' or 'down'", http.StatusBadRequest)
		return
	}

	// Safety check: Prevent disabling the interface being used for this connection
	if req.Action == "down" && !req.Force {
		clientIP := getClientIP(r)
		if isInterfaceUsedByIP(name, clientIP) {
			json.NewEncoder(w).Encode(map[string]any{
				"success":        false,
				"error":          "Cannot disable this interface - it's being used for your current connection!",
				"is_active_conn": true,
				"warning":        fmt.Sprintf("Disabling %s would disconnect you from the server. Use force=true to override.", name),
			})
			return
		}
	}

	// Use ip link to change interface state
	cmd := exec.Command("ip", "link", "set", name, req.Action)
	output, err := cmd.CombinedOutput()

	if err != nil {
		logNetworkEvent("error", name, fmt.Sprintf("Failed to set interface %s", req.Action), string(output))
		json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"error":   string(output),
		})
		return
	}

	logNetworkEvent("interface_toggle", name, fmt.Sprintf("Interface set to %s", req.Action), "")
	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"message": fmt.Sprintf("Interface %s set to %s", name, req.Action),
	})
}

// getClientIP extracts the client IP from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	// Check X-Real-IP
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	if colonIdx := strings.LastIndex(ip, ":"); colonIdx != -1 {
		ip = ip[:colonIdx]
	}
	// Remove brackets for IPv6
	ip = strings.Trim(ip, "[]")
	return ip
}

// isInterfaceUsedByIP checks if the given interface is used to reach the given IP
func isInterfaceUsedByIP(ifaceName, clientIP string) bool {
	// Get the interface's IP address
	cmd := exec.Command("ip", "addr", "show", ifaceName)
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	// Check if this interface has an IP that could be used to reach the client
	// We check if the client IP is in the same subnet or if this is the default route interface
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "inet ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				ifaceIP := strings.Split(parts[1], "/")[0]
				// Simple check: if client IP starts with the same first 3 octets
				// This is a simplified check; for production you'd want proper subnet matching
				if sameNetwork(ifaceIP, clientIP) {
					return true
				}
			}
		}
	}

	// Also check if this is the default route interface
	routeCmd := exec.Command("ip", "route", "get", clientIP)
	routeOutput, _ := routeCmd.Output()
	if strings.Contains(string(routeOutput), "dev "+ifaceName) {
		return true
	}

	return false
}

// sameNetwork checks if two IPs are likely in the same network (simplified)
func sameNetwork(ip1, ip2 string) bool {
	parts1 := strings.Split(ip1, ".")
	parts2 := strings.Split(ip2, ".")
	if len(parts1) != 4 || len(parts2) != 4 {
		return false
	}
	// Compare first 3 octets (assumes /24 network - simplified)
	return parts1[0] == parts2[0] && parts1[1] == parts2[1] && parts1[2] == parts2[2]
}

// GetNetworkBandwidthHandler returns real-time bandwidth for all interfaces
func GetNetworkBandwidthHandler(w http.ResponseWriter, r *http.Request) {
	interfaces := getDetailedNetworkInterfaces()

	totalRx := int64(0)
	totalTx := int64(0)
	totalRxSpeed := float64(0)
	totalTxSpeed := float64(0)

	interfacesRx := make(map[string]float64)
	interfacesTx := make(map[string]float64)

	for _, iface := range interfaces {
		totalRx += iface.RxBytes
		totalTx += iface.TxBytes
		totalRxSpeed += iface.RxSpeed
		totalTxSpeed += iface.TxSpeed
		interfacesRx[iface.Name] = iface.RxSpeed
		interfacesTx[iface.Name] = iface.TxSpeed
	}

	// Store in history
	addBandwidthHistory(totalRxSpeed, totalTxSpeed, interfacesRx, interfacesTx)

	// Get history for response
	bandwidthHistoryLock.RLock()
	history := make([]BandwidthHistoryEntry, len(bandwidthHistory))
	copy(history, bandwidthHistory)
	bandwidthHistoryLock.RUnlock()

	json.NewEncoder(w).Encode(map[string]any{
		"success":            true,
		"interfaces":         interfaces,
		"total_rx":           totalRx,
		"total_tx":           totalTx,
		"total_rx_formatted": formatBytes(totalRx),
		"total_tx_formatted": formatBytes(totalTx),
		"total_rx_speed":     totalRxSpeed,
		"total_tx_speed":     totalTxSpeed,
		"total_rx_speed_fmt": formatBytesPerSec(totalRxSpeed),
		"total_tx_speed_fmt": formatBytesPerSec(totalTxSpeed),
		"history":            history,
	})
}

// GetNetworkHistoryHandler returns bandwidth history for charts
func GetNetworkHistoryHandler(w http.ResponseWriter, r *http.Request) {
	bandwidthHistoryLock.RLock()
	history := make([]BandwidthHistoryEntry, len(bandwidthHistory))
	copy(history, bandwidthHistory)
	bandwidthHistoryLock.RUnlock()

	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"history": history,
	})
}

// GetNetworkEventsHandler returns network event log
func GetNetworkEventsHandler(w http.ResponseWriter, r *http.Request) {
	networkEventsLock.RLock()
	events := make([]NetworkEvent, len(networkEvents))
	copy(events, networkEvents)
	networkEventsLock.RUnlock()

	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"events":  events,
	})
}

// GetNetworkProcessesHandler returns processes using network
func GetNetworkProcessesHandler(w http.ResponseWriter, r *http.Request) {
	processes := getNetworkProcesses()
	json.NewEncoder(w).Encode(map[string]any{
		"success":   true,
		"processes": processes,
	})
}

// GetNetworkConnectionsHandler returns active network connections
func GetNetworkConnectionsHandler(w http.ResponseWriter, r *http.Request) {
	connections := getNetworkConnections()
	json.NewEncoder(w).Encode(map[string]any{
		"success":     true,
		"connections": connections,
	})
}

// ResetSessionStatsHandler resets session bandwidth counters
func ResetSessionStatsHandler(w http.ResponseWriter, r *http.Request) {
	sessionBaselineLock.Lock()
	sessionStart = time.Now()
	sessionBaseline = make(map[string]bandwidthReading)
	sessionBaselineLock.Unlock()

	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"message": "Session statistics reset",
	})
}

// GetProcessDetailsHandler returns detailed info about a process including network history
func GetProcessDetailsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pidStr := vars["pid"]
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		http.Error(w, "Invalid PID", http.StatusBadRequest)
		return
	}

	// Check if process exists
	if _, err := os.Stat(fmt.Sprintf("/proc/%d", pid)); os.IsNotExist(err) {
		json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"error":   "Process not found",
		})
		return
	}

	details := getProcessDetails(pid)
	if details == nil {
		json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"error":   "Failed to get process details",
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"process": details,
	})
}

// GetProcessHistoryHandler returns network history for a specific process
func GetProcessHistoryHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pidStr := vars["pid"]
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		http.Error(w, "Invalid PID", http.StatusBadRequest)
		return
	}

	processHistoryLock.RLock()
	history := processHistory[pid]
	processHistoryLock.RUnlock()

	if history == nil {
		history = []ProcessHistoryEntry{}
	}

	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"pid":     pid,
		"history": history,
	})
}

// KillProcessHandler kills a process by PID
func KillProcessHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pidStr := vars["pid"]
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		http.Error(w, "Invalid PID", http.StatusBadRequest)
		return
	}

	var req struct {
		Signal string `json:"signal"` // "TERM", "KILL", "INT", etc.
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Default to TERM
		req.Signal = "TERM"
	}

	// Map signal name to syscall signal
	var sig os.Signal
	switch strings.ToUpper(req.Signal) {
	case "KILL", "9":
		sig = syscall.SIGKILL
	case "INT", "2":
		sig = syscall.SIGINT
	case "HUP", "1":
		sig = syscall.SIGHUP
	case "QUIT", "3":
		sig = syscall.SIGQUIT
	default:
		sig = syscall.SIGTERM
	}

	// Get process name for logging
	procName := ""
	if data, err := os.ReadFile(fmt.Sprintf("/proc/%d/comm", pid)); err == nil {
		procName = strings.TrimSpace(string(data))
	}

	// Find the process
	process, err := os.FindProcess(pid)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"error":   fmt.Sprintf("Process not found: %v", err),
		})
		return
	}

	// Send signal
	err = process.Signal(sig)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"error":   fmt.Sprintf("Failed to send signal: %v", err),
		})
		return
	}

	logNetworkEvent("process_killed", "", fmt.Sprintf("Process %s (PID %d) sent signal %s", procName, pid, req.Signal), "")

	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"message": fmt.Sprintf("Signal %s sent to process %d", req.Signal, pid),
	})
}

// getProcessDetails fetches detailed information about a process
func getProcessDetails(pid int) *ProcessDetails {
	details := &ProcessDetails{
		PID: pid,
	}

	// Get process name
	if data, err := os.ReadFile(fmt.Sprintf("/proc/%d/comm", pid)); err == nil {
		details.Name = strings.TrimSpace(string(data))
	}

	// Get cmdline
	if data, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid)); err == nil {
		// Replace null bytes with spaces
		details.Cmdline = strings.ReplaceAll(string(data), "\x00", " ")
		details.Cmdline = strings.TrimSpace(details.Cmdline)
	}

	// Get status info
	if data, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid)); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "State:") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					details.State = parts[1]
				}
			} else if strings.HasPrefix(line, "Uid:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					uid := fields[1]
					details.User = uid
					if userData, err := exec.Command("id", "-nu", uid).Output(); err == nil {
						details.User = strings.TrimSpace(string(userData))
					}
				}
			} else if strings.HasPrefix(line, "Threads:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					details.Threads, _ = strconv.Atoi(fields[1])
				}
			} else if strings.HasPrefix(line, "VmRSS:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					val, _ := strconv.ParseInt(fields[1], 10, 64)
					details.MemoryRSS = val * 1024 // Convert from kB to bytes
				}
			} else if strings.HasPrefix(line, "VmSize:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					val, _ := strconv.ParseInt(fields[1], 10, 64)
					details.MemoryVMS = val * 1024
				}
			}
		}
	}

	// Get start time
	if data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid)); err == nil {
		fields := strings.Fields(string(data))
		if len(fields) > 21 {
			starttime, _ := strconv.ParseInt(fields[21], 10, 64)
			// Convert from clock ticks to time
			if uptimeData, err := os.ReadFile("/proc/uptime"); err == nil {
				uptimeParts := strings.Fields(string(uptimeData))
				if len(uptimeParts) > 0 {
					uptime, _ := strconv.ParseFloat(uptimeParts[0], 64)
					ticksPerSec := int64(100) // Usually 100 Hz
					startSeconds := float64(starttime) / float64(ticksPerSec)
					processUptime := uptime - startSeconds
					startTime := time.Now().Add(-time.Duration(processUptime) * time.Second)
					details.StartTime = startTime.Format(time.RFC3339)
				}
			}
		}
	}

	// Get network stats from our tracking
	prevProcessBandwidthLock.RLock()
	if prev, ok := prevProcessBandwidth[pid]; ok {
		details.RxBytes = prev.RxBytes
		details.TxBytes = prev.TxBytes
	}
	prevProcessBandwidthLock.RUnlock()

	// Get history
	processHistoryLock.RLock()
	if hist, ok := processHistory[pid]; ok {
		details.History = hist
		// Calculate current speed from last entries
		if len(hist) >= 2 {
			last := hist[len(hist)-1]
			details.RxSpeed = last.RxSpeed
			details.TxSpeed = last.TxSpeed
		}
	} else {
		details.History = []ProcessHistoryEntry{}
	}
	processHistoryLock.RUnlock()

	// Count connections
	cmd := exec.Command("ss", "-tunap")
	if output, err := cmd.Output(); err == nil {
		pidStr := fmt.Sprintf("pid=%d,", pid)
		for _, line := range strings.Split(string(output), "\n") {
			if strings.Contains(line, pidStr) {
				details.Connections++
			}
		}
	}

	return details
}

// addProcessHistory adds a history entry for a process
func addProcessHistory(pid int, rxSpeed, txSpeed float64, rxBytes, txBytes int64) {
	processHistoryLock.Lock()
	defer processHistoryLock.Unlock()

	entry := ProcessHistoryEntry{
		Timestamp: time.Now(),
		RxSpeed:   rxSpeed,
		TxSpeed:   txSpeed,
		RxBytes:   rxBytes,
		TxBytes:   txBytes,
	}

	history := processHistory[pid]
	history = append(history, entry)

	// Keep only last maxProcessHistorySize entries
	if len(history) > maxProcessHistorySize {
		history = history[len(history)-maxProcessHistorySize:]
	}

	processHistory[pid] = history

	// Cleanup old processes if too many tracked
	if len(processHistory) > maxTrackedProcesses {
		// Remove oldest process (simple cleanup)
		var oldestPid int
		var oldestTime time.Time
		for p, h := range processHistory {
			if len(h) > 0 && (oldestTime.IsZero() || h[0].Timestamp.Before(oldestTime)) {
				oldestPid = p
				oldestTime = h[0].Timestamp
			}
		}
		if oldestPid != 0 {
			delete(processHistory, oldestPid)
		}
	}
}

func getDetailedNetworkInterfaces() []NetworkInterface {
	cmd := exec.Command("ls", "/sys/class/net/")
	output, _ := cmd.Output()
	ifaceNames := []string{}
	for _, line := range strings.Split(string(output), "\n") {
		if line != "" {
			ifaceNames = append(ifaceNames, line)
		}
	}

	result := []NetworkInterface{}
	now := time.Now()

	for _, name := range ifaceNames {
		iface := NetworkInterface{
			Name:        name,
			DisplayName: name,
			Type:        "Unknown",
			Status:      "Down",
			IsUp:        false,
			IP:          "-",
			IPv6:        "-",
			Subnet:      "-",
			Gateway:     "-",
			MAC:         "-",
			Speed:       "-",
			Duplex:      "-",
			Driver:      "-",
		}

		// Get operstate
		if data, err := os.ReadFile(fmt.Sprintf("/sys/class/net/%s/operstate", name)); err == nil {
			state := strings.TrimSpace(string(data))
			if state == "up" {
				iface.Status = "Up"
				iface.IsUp = true
			} else if state == "down" {
				iface.Status = "Down"
			} else {
				iface.Status = strings.ToUpper(state[:1]) + state[1:]
			}
		}

		// Get MAC address
		if data, err := os.ReadFile(fmt.Sprintf("/sys/class/net/%s/address", name)); err == nil {
			iface.MAC = strings.TrimSpace(string(data))
		}

		// Get MTU
		if data, err := os.ReadFile(fmt.Sprintf("/sys/class/net/%s/mtu", name)); err == nil {
			iface.MTU, _ = strconv.Atoi(strings.TrimSpace(string(data)))
		}

		// Get link speed (for physical interfaces)
		if data, err := os.ReadFile(fmt.Sprintf("/sys/class/net/%s/speed", name)); err == nil {
			speed, _ := strconv.Atoi(strings.TrimSpace(string(data)))
			if speed > 0 {
				if speed >= 1000 {
					iface.Speed = fmt.Sprintf("%d Gbps", speed/1000)
				} else {
					iface.Speed = fmt.Sprintf("%d Mbps", speed)
				}
			}
		}

		// Get duplex
		if data, err := os.ReadFile(fmt.Sprintf("/sys/class/net/%s/duplex", name)); err == nil {
			iface.Duplex = strings.TrimSpace(string(data))
		}

		// Get driver
		if link, err := os.Readlink(fmt.Sprintf("/sys/class/net/%s/device/driver", name)); err == nil {
			parts := strings.Split(link, "/")
			if len(parts) > 0 {
				iface.Driver = parts[len(parts)-1]
			}
		}

		// Check if virtual
		if _, err := os.Stat(fmt.Sprintf("/sys/devices/virtual/net/%s", name)); err == nil {
			iface.IsVirtual = true
		}

		// Check if wireless
		if _, err := os.Stat(fmt.Sprintf("/sys/class/net/%s/wireless", name)); err == nil {
			iface.IsWireless = true
			iface.Type = "Wireless"
		}

		// Determine type
		if name == "lo" {
			iface.Type = "Loopback"
			iface.DisplayName = "Loopback"
		} else if strings.HasPrefix(name, "eth") || strings.HasPrefix(name, "en") {
			iface.Type = "Ethernet"
		} else if strings.HasPrefix(name, "wlan") || strings.HasPrefix(name, "wl") {
			iface.Type = "Wireless"
			iface.IsWireless = true
		} else if strings.HasPrefix(name, "br") {
			iface.Type = "Bridge"
		} else if strings.HasPrefix(name, "veth") {
			iface.Type = "Virtual Ethernet"
			iface.IsVirtual = true
		} else if strings.HasPrefix(name, "docker") {
			iface.Type = "Docker"
			iface.IsVirtual = true
		} else if strings.HasPrefix(name, "virbr") {
			iface.Type = "Virtual Bridge"
			iface.IsVirtual = true
		} else if strings.HasPrefix(name, "vnet") {
			iface.Type = "Virtual Network"
			iface.IsVirtual = true
		} else if strings.HasPrefix(name, "tap") || strings.HasPrefix(name, "tun") {
			iface.Type = "Tunnel"
			iface.IsVirtual = true
		}

		// Get IP addresses using ip command
		ipCmd := exec.Command("ip", "addr", "show", name)
		ipOutput, _ := ipCmd.Output()
		for _, line := range strings.Split(string(ipOutput), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "inet ") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					ipWithMask := parts[1]
					ipParts := strings.Split(ipWithMask, "/")
					iface.IP = ipParts[0]
					if len(ipParts) > 1 {
						iface.Subnet = "/" + ipParts[1]
					}
				}
			} else if strings.HasPrefix(line, "inet6 ") && !strings.Contains(line, "fe80") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					iface.IPv6 = strings.Split(parts[1], "/")[0]
				}
			}
		}

		// Get default gateway for this interface
		routeCmd := exec.Command("ip", "route", "show", "dev", name)
		routeOutput, _ := routeCmd.Output()
		for _, line := range strings.Split(string(routeOutput), "\n") {
			if strings.HasPrefix(line, "default via ") {
				parts := strings.Fields(line)
				if len(parts) >= 3 {
					iface.Gateway = parts[2]
				}
			}
		}

		// Get traffic statistics
		if data, err := os.ReadFile(fmt.Sprintf("/sys/class/net/%s/statistics/rx_bytes", name)); err == nil {
			iface.RxBytes, _ = strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
		}
		if data, err := os.ReadFile(fmt.Sprintf("/sys/class/net/%s/statistics/tx_bytes", name)); err == nil {
			iface.TxBytes, _ = strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
		}
		if data, err := os.ReadFile(fmt.Sprintf("/sys/class/net/%s/statistics/rx_packets", name)); err == nil {
			iface.RxPackets, _ = strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
		}
		if data, err := os.ReadFile(fmt.Sprintf("/sys/class/net/%s/statistics/tx_packets", name)); err == nil {
			iface.TxPackets, _ = strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
		}
		if data, err := os.ReadFile(fmt.Sprintf("/sys/class/net/%s/statistics/rx_errors", name)); err == nil {
			iface.RxErrors, _ = strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
		}
		if data, err := os.ReadFile(fmt.Sprintf("/sys/class/net/%s/statistics/tx_errors", name)); err == nil {
			iface.TxErrors, _ = strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
		}
		if data, err := os.ReadFile(fmt.Sprintf("/sys/class/net/%s/statistics/rx_dropped", name)); err == nil {
			iface.RxDropped, _ = strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
		}
		if data, err := os.ReadFile(fmt.Sprintf("/sys/class/net/%s/statistics/tx_dropped", name)); err == nil {
			iface.TxDropped, _ = strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
		}

		iface.RxFormatted = formatBytes(iface.RxBytes)
		iface.TxFormatted = formatBytes(iface.TxBytes)

		// Calculate session traffic
		sessionBaselineLock.RLock()
		baseline, hasBaseline := sessionBaseline[name]
		sessionBaselineLock.RUnlock()

		if !hasBaseline {
			sessionBaselineLock.Lock()
			sessionBaseline[name] = bandwidthReading{
				RxBytes:   iface.RxBytes,
				TxBytes:   iface.TxBytes,
				Timestamp: now,
			}
			sessionBaselineLock.Unlock()
			iface.RxSession = 0
			iface.TxSession = 0
		} else {
			iface.RxSession = iface.RxBytes - baseline.RxBytes
			iface.TxSession = iface.TxBytes - baseline.TxBytes
			if iface.RxSession < 0 {
				iface.RxSession = iface.RxBytes
			}
			if iface.TxSession < 0 {
				iface.TxSession = iface.TxBytes
			}
		}
		iface.RxSessionFmt = formatBytes(iface.RxSession)
		iface.TxSessionFmt = formatBytes(iface.TxSession)

		// Calculate speed (bytes per second)
		prevBandwidthLock.RLock()
		prev, hasPrev := prevBandwidth[name]
		prevBandwidthLock.RUnlock()

		if hasPrev {
			duration := now.Sub(prev.Timestamp).Seconds()
			if duration > 0 {
				iface.RxSpeed = float64(iface.RxBytes-prev.RxBytes) / duration
				iface.TxSpeed = float64(iface.TxBytes-prev.TxBytes) / duration
				if iface.RxSpeed < 0 {
					iface.RxSpeed = 0
				}
				if iface.TxSpeed < 0 {
					iface.TxSpeed = 0
				}
			}
		}

		iface.RxSpeedFmt = formatBytesPerSec(iface.RxSpeed)
		iface.TxSpeedFmt = formatBytesPerSec(iface.TxSpeed)

		// Update previous reading
		prevBandwidthLock.Lock()
		prevBandwidth[name] = bandwidthReading{
			RxBytes:   iface.RxBytes,
			TxBytes:   iface.TxBytes,
			Timestamp: now,
		}
		prevBandwidthLock.Unlock()

		result = append(result, iface)
	}

	return result
}

func getNetworkProcesses() []NetworkProcess {
	processes := []NetworkProcess{}
	now := time.Now()

	// Use ss with extended info to get per-process bandwidth
	// ss -tunapi gives detailed info including bytes_received/bytes_acked
	cmd := exec.Command("ss", "-tunap", "-i")
	output, err := cmd.Output()
	if err != nil {
		return processes
	}

	// Parse ss output - aggregate by PID
	type procData struct {
		pid         int
		name        string
		user        string
		connections int
		rxBytes     int64
		txBytes     int64
	}
	procMap := make(map[int]*procData)

	lines := strings.Split(string(output), "\n")
	var currentPid int
	var currentLine string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check if this is a connection line (starts with tcp/udp)
		if strings.HasPrefix(line, "tcp") || strings.HasPrefix(line, "udp") {
			currentLine = line
			// Extract PID from users:((\"name\",pid=123,fd=4))
			if idx := strings.Index(line, "pid="); idx != -1 {
				pidStr := line[idx+4:]
				if commaIdx := strings.Index(pidStr, ","); commaIdx != -1 {
					pidStr = pidStr[:commaIdx]
				}
				if pid, err := strconv.Atoi(pidStr); err == nil && pid > 0 {
					currentPid = pid
					if procMap[pid] == nil {
						procMap[pid] = &procData{pid: pid}
						// Extract process name
						if nameIdx := strings.Index(line, "users:((\""); nameIdx != -1 {
							nameStr := line[nameIdx+9:]
							if endIdx := strings.Index(nameStr, "\""); endIdx != -1 {
								procMap[pid].name = nameStr[:endIdx]
							}
						}
					}
					procMap[pid].connections++
				}
			}
			continue
		}

		// Check if this is an info line with bytes data
		if currentPid > 0 && strings.Contains(line, "bytes_received:") {
			// Parse bytes_received
			if idx := strings.Index(line, "bytes_received:"); idx != -1 {
				valueStr := line[idx+15:]
				if spaceIdx := strings.Index(valueStr, " "); spaceIdx != -1 {
					valueStr = valueStr[:spaceIdx]
				}
				if bytes, err := strconv.ParseInt(valueStr, 10, 64); err == nil {
					procMap[currentPid].rxBytes += bytes
				}
			}
			// Parse bytes_acked (approximation of sent)
			if idx := strings.Index(line, "bytes_acked:"); idx != -1 {
				valueStr := line[idx+12:]
				if spaceIdx := strings.Index(valueStr, " "); spaceIdx != -1 {
					valueStr = valueStr[:spaceIdx]
				}
				if bytes, err := strconv.ParseInt(valueStr, 10, 64); err == nil {
					procMap[currentPid].txBytes += bytes
				}
			}
		}
		_ = currentLine // avoid unused variable warning
	}

	// Convert map to slice and calculate speeds
	for pid, data := range procMap {
		if data.name == "" {
			// Get process name from /proc if not available
			if nameData, err := os.ReadFile(fmt.Sprintf("/proc/%d/comm", pid)); err == nil {
				data.name = strings.TrimSpace(string(nameData))
			}
		}

		// Get user
		user := ""
		if statusData, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid)); err == nil {
			for _, line := range strings.Split(string(statusData), "\n") {
				if strings.HasPrefix(line, "Uid:") {
					fields := strings.Fields(line)
					if len(fields) >= 2 {
						uid := fields[1]
						user = uid
						if userData, err := exec.Command("id", "-nu", uid).Output(); err == nil {
							user = strings.TrimSpace(string(userData))
						}
					}
					break
				}
			}
		}

		proc := NetworkProcess{
			PID:         pid,
			Name:        data.name,
			User:        user,
			Connections: data.connections,
			RxBytes:     data.rxBytes,
			TxBytes:     data.txBytes,
		}

		// Calculate speed from previous reading
		prevProcessBandwidthLock.RLock()
		prev, hasPrev := prevProcessBandwidth[pid]
		prevProcessBandwidthLock.RUnlock()

		if hasPrev {
			duration := now.Sub(prev.Timestamp).Seconds()
			if duration > 0 {
				proc.RxSpeed = float64(data.rxBytes-prev.RxBytes) / duration
				proc.TxSpeed = float64(data.txBytes-prev.TxBytes) / duration
				if proc.RxSpeed < 0 {
					proc.RxSpeed = 0
				}
				if proc.TxSpeed < 0 {
					proc.TxSpeed = 0
				}
			}
		}

		proc.RxSpeedFmt = formatBytesPerSec(proc.RxSpeed)
		proc.TxSpeedFmt = formatBytesPerSec(proc.TxSpeed)

		// Update previous reading
		prevProcessBandwidthLock.Lock()
		prevProcessBandwidth[pid] = processBandwidthReading{
			RxBytes:   data.rxBytes,
			TxBytes:   data.txBytes,
			Timestamp: now,
		}
		prevProcessBandwidthLock.Unlock()

		// Add to process history (only for active processes with some bandwidth)
		if proc.RxSpeed > 0 || proc.TxSpeed > 0 || data.rxBytes > 0 || data.txBytes > 0 {
			addProcessHistory(pid, proc.RxSpeed, proc.TxSpeed, data.rxBytes, data.txBytes)
		}

		processes = append(processes, proc)
	}

	// Sort by total bandwidth (rx + tx)
	for i := 0; i < len(processes)-1; i++ {
		for j := i + 1; j < len(processes); j++ {
			totalI := processes[i].RxSpeed + processes[i].TxSpeed
			totalJ := processes[j].RxSpeed + processes[j].TxSpeed
			if totalJ > totalI {
				processes[i], processes[j] = processes[j], processes[i]
			}
		}
	}

	return processes
}

func findPidByInode(inode string) int {
	// Search /proc/*/fd/* for matching inode
	procs, _ := os.ReadDir("/proc")
	for _, proc := range procs {
		if !proc.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(proc.Name())
		if err != nil {
			continue
		}

		fdPath := fmt.Sprintf("/proc/%d/fd", pid)
		fds, err := os.ReadDir(fdPath)
		if err != nil {
			continue
		}

		for _, fd := range fds {
			link, err := os.Readlink(fmt.Sprintf("%s/%s", fdPath, fd.Name()))
			if err != nil {
				continue
			}
			if strings.Contains(link, fmt.Sprintf("socket:[%s]", inode)) {
				return pid
			}
		}
	}
	return 0
}

func getNetworkConnections() []map[string]any {
	connections := []map[string]any{}

	// Use ss command for detailed connection info
	cmd := exec.Command("ss", "-tunap")
	output, err := cmd.Output()
	if err != nil {
		return connections
	}

	lines := strings.Split(string(output), "\n")
	for i, line := range lines {
		if i == 0 || line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		conn := map[string]any{
			"protocol":   fields[0],
			"state":      fields[1],
			"recv_q":     fields[2],
			"send_q":     fields[3],
			"local":      fields[4],
		}

		if len(fields) > 5 {
			conn["remote"] = fields[5]
		}
		if len(fields) > 6 {
			// Parse process info
			procInfo := fields[6]
			if strings.Contains(procInfo, "pid=") {
				conn["process"] = procInfo
			}
		}

		connections = append(connections, conn)
	}

	return connections
}

func formatBytesPerSec(bytesPerSec float64) string {
	if bytesPerSec < 0 {
		bytesPerSec = 0
	}

	units := []string{"B/s", "KB/s", "MB/s", "GB/s"}
	value := bytesPerSec
	unitIndex := 0

	for value >= 1024 && unitIndex < len(units)-1 {
		value /= 1024
		unitIndex++
	}

	if value < 10 {
		return fmt.Sprintf("%.2f %s", value, units[unitIndex])
	} else if value < 100 {
		return fmt.Sprintf("%.1f %s", value, units[unitIndex])
	}
	return fmt.Sprintf("%.0f %s", value, units[unitIndex])
}

// Helper functions for history and logging

func addBandwidthHistory(totalRx, totalTx float64, interfacesRx, interfacesTx map[string]float64) {
	bandwidthHistoryLock.Lock()
	defer bandwidthHistoryLock.Unlock()

	entry := BandwidthHistoryEntry{
		Timestamp:    time.Now(),
		TotalRx:      totalRx,
		TotalTx:      totalTx,
		Interfaces:   interfacesRx,
		InterfacesTx: interfacesTx,
	}

	bandwidthHistory = append(bandwidthHistory, entry)

	// Keep only last maxHistorySize entries
	if len(bandwidthHistory) > maxHistorySize {
		bandwidthHistory = bandwidthHistory[len(bandwidthHistory)-maxHistorySize:]
	}
}

var networkEventID = 0

// Process bandwidth throttling
type ProcessThrottle struct {
	PID           int    `json:"pid"`
	Name          string `json:"name"`
	DownloadLimit int64  `json:"download_limit"` // bytes per second (0 = unlimited)
	UploadLimit   int64  `json:"upload_limit"`   // bytes per second (0 = unlimited)
	Interface     string `json:"interface"`
	ClassID       int    `json:"class_id"`
	CgroupPath    string `json:"cgroup_path"`
}

var (
	processThrottles     = make(map[int]*ProcessThrottle)
	processThrottlesLock sync.RWMutex
	nextClassID          = 100 // Start class IDs from 100
	throttleInitialized  = make(map[string]bool) // Track initialized interfaces
)

// SetProcessThrottleHandler sets bandwidth limits for a process
func SetProcessThrottleHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PID           int    `json:"pid"`
		DownloadLimit int64  `json:"download_limit"` // bytes per second
		UploadLimit   int64  `json:"upload_limit"`   // bytes per second
		Interface     string `json:"interface"`      // Optional, defaults to default route interface
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.PID <= 0 {
		json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"error":   "Invalid PID",
		})
		return
	}

	// Check if process exists
	if _, err := os.Stat(fmt.Sprintf("/proc/%d", req.PID)); os.IsNotExist(err) {
		json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"error":   "Process not found",
		})
		return
	}

	// Get process name
	procName := ""
	if data, err := os.ReadFile(fmt.Sprintf("/proc/%d/comm", req.PID)); err == nil {
		procName = strings.TrimSpace(string(data))
	}

	// Determine interface if not specified
	iface := req.Interface
	if iface == "" {
		iface = getDefaultInterface()
	}
	if iface == "" {
		json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"error":   "Could not determine network interface",
		})
		return
	}

	// Check if cgroups net_cls is available
	netClsPath := "/sys/fs/cgroup/net_cls"
	if _, err := os.Stat(netClsPath); os.IsNotExist(err) {
		// Try cgroup v1 legacy location
		netClsPath = "/sys/fs/cgroup/net_cls,net_prio"
		if _, err := os.Stat(netClsPath); os.IsNotExist(err) {
			json.NewEncoder(w).Encode(map[string]any{
				"success": false,
				"error":   "cgroups net_cls not available. You may need to mount it: mount -t cgroup -o net_cls net_cls /sys/fs/cgroup/net_cls",
			})
			return
		}
	}

	// Initialize tc on interface if needed
	if !throttleInitialized[iface] {
		if err := initializeThrottleInterface(iface); err != nil {
			json.NewEncoder(w).Encode(map[string]any{
				"success": false,
				"error":   fmt.Sprintf("Failed to initialize throttling on %s: %v", iface, err),
			})
			return
		}
		throttleInitialized[iface] = true
	}

	processThrottlesLock.Lock()
	defer processThrottlesLock.Unlock()

	// Check if already throttled
	existing, exists := processThrottles[req.PID]
	var classID int
	if exists {
		classID = existing.ClassID
	} else {
		classID = nextClassID
		nextClassID++
	}

	// Create cgroup for process
	cgroupPath := fmt.Sprintf("%s/throttle_%d", netClsPath, req.PID)
	if !exists {
		if err := os.MkdirAll(cgroupPath, 0755); err != nil {
			json.NewEncoder(w).Encode(map[string]any{
				"success": false,
				"error":   fmt.Sprintf("Failed to create cgroup: %v", err),
			})
			return
		}
	}

	// Move process to cgroup
	tasksFile := fmt.Sprintf("%s/tasks", cgroupPath)
	// Also try cgroup.procs for newer systems
	procsFile := fmt.Sprintf("%s/cgroup.procs", cgroupPath)

	written := false
	if err := os.WriteFile(tasksFile, []byte(strconv.Itoa(req.PID)), 0644); err == nil {
		written = true
	}
	if !written {
		if err := os.WriteFile(procsFile, []byte(strconv.Itoa(req.PID)), 0644); err != nil {
			json.NewEncoder(w).Encode(map[string]any{
				"success": false,
				"error":   fmt.Sprintf("Failed to add process to cgroup: %v", err),
			})
			return
		}
	}

	// Set class ID for cgroup (format: 0xAAAABBBB where AAAA is major, BBBB is minor)
	classIDHex := fmt.Sprintf("0x%04x%04x", 1, classID)
	classIDFile := fmt.Sprintf("%s/net_cls.classid", cgroupPath)
	if err := os.WriteFile(classIDFile, []byte(classIDHex), 0644); err != nil {
		json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"error":   fmt.Sprintf("Failed to set class ID: %v", err),
		})
		return
	}

	// Create/update tc class with bandwidth limits
	downloadRate := "1000mbit" // Default unlimited
	uploadRate := "1000mbit"

	if req.DownloadLimit > 0 {
		// Convert bytes/s to kbit/s (multiply by 8 for bits, divide by 1000 for kbit)
		kbits := (req.DownloadLimit * 8) / 1000
		if kbits < 1 {
			kbits = 1
		}
		downloadRate = fmt.Sprintf("%dkbit", kbits)
	}

	if req.UploadLimit > 0 {
		kbits := (req.UploadLimit * 8) / 1000
		if kbits < 1 {
			kbits = 1
		}
		uploadRate = fmt.Sprintf("%dkbit", kbits)
	}

	// Remove existing class if updating
	if exists {
		exec.Command("tc", "class", "del", "dev", iface, "classid", fmt.Sprintf("1:%d", classID)).Run()
	}

	// Add tc class for upload (egress)
	cmd := exec.Command("tc", "class", "add", "dev", iface, "parent", "1:", "classid", fmt.Sprintf("1:%d", classID), "htb", "rate", uploadRate, "ceil", uploadRate)
	if output, err := cmd.CombinedOutput(); err != nil {
		json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"error":   fmt.Sprintf("Failed to create tc class: %v - %s", err, string(output)),
		})
		return
	}

	// Add tc filter to match the cgroup class
	filterCmd := exec.Command("tc", "filter", "add", "dev", iface, "parent", "1:", "handle", fmt.Sprintf("%d:", classID), "cgroup")
	filterCmd.Run() // Ignore errors, filter might already exist

	// For ingress (download) limiting, we need IFB (Intermediate Functional Block)
	if req.DownloadLimit > 0 {
		setupIngressThrottle(iface, classID, downloadRate)
	}

	throttle := &ProcessThrottle{
		PID:           req.PID,
		Name:          procName,
		DownloadLimit: req.DownloadLimit,
		UploadLimit:   req.UploadLimit,
		Interface:     iface,
		ClassID:       classID,
		CgroupPath:    cgroupPath,
	}
	processThrottles[req.PID] = throttle

	logNetworkEvent("throttle_set", iface, fmt.Sprintf("Bandwidth limit set for %s (PID %d)", procName, req.PID),
		fmt.Sprintf("Download: %s, Upload: %s", downloadRate, uploadRate))

	json.NewEncoder(w).Encode(map[string]any{
		"success":  true,
		"message":  fmt.Sprintf("Bandwidth limits set for process %d", req.PID),
		"throttle": throttle,
	})
}

// RemoveProcessThrottleHandler removes bandwidth limits for a process
func RemoveProcessThrottleHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pidStr := vars["pid"]
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		http.Error(w, "Invalid PID", http.StatusBadRequest)
		return
	}

	processThrottlesLock.Lock()
	defer processThrottlesLock.Unlock()

	throttle, exists := processThrottles[pid]
	if !exists {
		json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"error":   "Process is not throttled",
		})
		return
	}

	// Remove tc class
	exec.Command("tc", "class", "del", "dev", throttle.Interface, "classid", fmt.Sprintf("1:%d", throttle.ClassID)).Run()

	// Remove cgroup (process will be moved to parent)
	os.Remove(throttle.CgroupPath)

	delete(processThrottles, pid)

	logNetworkEvent("throttle_removed", throttle.Interface, fmt.Sprintf("Bandwidth limit removed for %s (PID %d)", throttle.Name, pid), "")

	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"message": fmt.Sprintf("Bandwidth limits removed for process %d", pid),
	})
}

// GetProcessThrottlesHandler lists all throttled processes
func GetProcessThrottlesHandler(w http.ResponseWriter, r *http.Request) {
	processThrottlesLock.RLock()
	defer processThrottlesLock.RUnlock()

	throttles := make([]*ProcessThrottle, 0, len(processThrottles))
	for _, t := range processThrottles {
		// Check if process still exists
		if _, err := os.Stat(fmt.Sprintf("/proc/%d", t.PID)); os.IsNotExist(err) {
			continue // Skip dead processes
		}
		throttles = append(throttles, t)
	}

	json.NewEncoder(w).Encode(map[string]any{
		"success":   true,
		"throttles": throttles,
	})
}

// initializeThrottleInterface sets up tc qdisc on interface for throttling
func initializeThrottleInterface(iface string) error {
	// Remove existing qdisc (ignore errors)
	exec.Command("tc", "qdisc", "del", "dev", iface, "root").Run()

	// Create HTB qdisc
	cmd := exec.Command("tc", "qdisc", "add", "dev", iface, "root", "handle", "1:", "htb", "default", "99")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create qdisc: %v - %s", err, string(output))
	}

	// Create default class (unlimited)
	cmd = exec.Command("tc", "class", "add", "dev", iface, "parent", "1:", "classid", "1:99", "htb", "rate", "1000mbit")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create default class: %v - %s", err, string(output))
	}

	return nil
}

// setupIngressThrottle sets up ingress (download) throttling using IFB
func setupIngressThrottle(iface string, classID int, rate string) error {
	ifbDev := "ifb0"

	// Load ifb module
	exec.Command("modprobe", "ifb", "numifbs=1").Run()

	// Bring up ifb device
	exec.Command("ip", "link", "set", "dev", ifbDev, "up").Run()

	// Setup ingress qdisc on main interface
	exec.Command("tc", "qdisc", "add", "dev", iface, "handle", "ffff:", "ingress").Run()

	// Redirect ingress to ifb
	exec.Command("tc", "filter", "add", "dev", iface, "parent", "ffff:", "protocol", "ip", "u32", "match", "u32", "0", "0", "action", "mirred", "egress", "redirect", "dev", ifbDev).Run()

	// Setup HTB on ifb for ingress shaping
	exec.Command("tc", "qdisc", "del", "dev", ifbDev, "root").Run()
	exec.Command("tc", "qdisc", "add", "dev", ifbDev, "root", "handle", "1:", "htb", "default", "99").Run()
	exec.Command("tc", "class", "add", "dev", ifbDev, "parent", "1:", "classid", "1:99", "htb", "rate", "1000mbit").Run()

	// Add class for this process on ifb
	cmd := exec.Command("tc", "class", "add", "dev", ifbDev, "parent", "1:", "classid", fmt.Sprintf("1:%d", classID), "htb", "rate", rate, "ceil", rate)
	cmd.Run()

	// Add cgroup filter on ifb
	exec.Command("tc", "filter", "add", "dev", ifbDev, "parent", "1:", "handle", fmt.Sprintf("%d:", classID), "cgroup").Run()

	return nil
}

// getDefaultInterface returns the default route interface
func getDefaultInterface() string {
	cmd := exec.Command("ip", "route", "show", "default")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	// Parse "default via X.X.X.X dev INTERFACE"
	fields := strings.Fields(string(output))
	for i, f := range fields {
		if f == "dev" && i+1 < len(fields) {
			return fields[i+1]
		}
	}
	return ""
}

// CheckThrottleSupport checks if bandwidth throttling is supported
func CheckThrottleSupportHandler(w http.ResponseWriter, r *http.Request) {
	supported := true
	issues := []string{}

	// Check for tc command
	if _, err := exec.LookPath("tc"); err != nil {
		supported = false
		issues = append(issues, "tc command not found (install iproute2)")
	}

	// Check for cgroups net_cls
	netClsPath := "/sys/fs/cgroup/net_cls"
	if _, err := os.Stat(netClsPath); os.IsNotExist(err) {
		netClsPath = "/sys/fs/cgroup/net_cls,net_prio"
		if _, err := os.Stat(netClsPath); os.IsNotExist(err) {
			supported = false
			issues = append(issues, "cgroups net_cls not mounted. Run: mount -t cgroup -o net_cls net_cls /sys/fs/cgroup/net_cls")
		}
	}

	// Check for ifb module (for download limiting)
	ifbSupported := true
	if output, err := exec.Command("modprobe", "-n", "ifb").CombinedOutput(); err != nil {
		ifbSupported = false
		issues = append(issues, fmt.Sprintf("ifb module not available (download limiting may not work): %s", string(output)))
	}

	json.NewEncoder(w).Encode(map[string]any{
		"success":            true,
		"throttle_supported": supported,
		"ifb_supported":      ifbSupported,
		"issues":             issues,
		"net_cls_path":       netClsPath,
	})
}

func logNetworkEvent(eventType, iface, description, details string) {
	networkEventsLock.Lock()
	defer networkEventsLock.Unlock()

	networkEventID++
	event := NetworkEvent{
		ID:          networkEventID,
		Timestamp:   time.Now(),
		Type:        eventType,
		Interface:   iface,
		Description: description,
		Details:     details,
	}

	// Add to beginning (newest first)
	networkEvents = append([]NetworkEvent{event}, networkEvents...)

	// Keep only last maxNetworkEvents entries
	if len(networkEvents) > maxNetworkEvents {
		networkEvents = networkEvents[:maxNetworkEvents]
	}
}
