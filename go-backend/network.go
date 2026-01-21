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
