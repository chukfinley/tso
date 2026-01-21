package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

const (
	VMDir        = "/opt/serveros/vms"
	ISODir       = "/opt/serveros/storage/isos"
	VMLogDir     = "/opt/serveros/logs/vms"
	VMBackupDir  = "/opt/serveros/vms/backups"
	TemplateDir  = "/opt/serveros/vms/templates"
	QMPSocketDir = "/opt/serveros/run/qmp"
	OVMFPath     = "/usr/share/OVMF/OVMF_CODE.fd"
	OVMFVarsPath = "/usr/share/OVMF/OVMF_VARS.fd"
)

// ISO download progress tracking
var (
	isoDownloads     = make(map[int]*ISODownloadProgress)
	isoDownloadsLock sync.RWMutex
)

type ISODownloadProgress struct {
	ID       int    `json:"id"`
	URL      string `json:"url"`
	Filename string `json:"filename"`
	Total    int64  `json:"total"`
	Current  int64  `json:"current"`
	Percent  int    `json:"percent"`
	Status   string `json:"status"`
	Error    string `json:"error,omitempty"`
}

// VM field list for scanning - matches extended schema
var vmFields = `id, name, description, uuid, cpu_cores, ram_mb,
	COALESCE(cpu_type, 'host'), COALESCE(cpu_pinning, ''), COALESCE(numa_topology, ''),
	COALESCE(balloon_enabled, true), COALESCE(hugepages_enabled, false),
	COALESCE(disk_path, ''), COALESCE(disk_size_gb, 20), COALESCE(disk_format, 'qcow2'),
	COALESCE(cache_mode, 'writeback'), COALESCE(discard_enabled, true),
	COALESCE(boot_order, 'cd,hd'), COALESCE(iso_path, ''), COALESCE(boot_from_disk, false),
	COALESCE(physical_disk_device, ''), COALESCE(firmware_type, 'bios'),
	COALESCE(secure_boot, false), COALESCE(tpm_enabled, false),
	COALESCE(network_mode, 'nat'), COALESCE(network_bridge, ''), COALESCE(mac_address, ''),
	COALESCE(network_model, 'virtio'), vlan_id, bandwidth_limit_down, bandwidth_limit_up,
	COALESCE(display_type, 'spice'), COALESCE(spice_port, 0), COALESCE(vnc_port, 0),
	COALESCE(spice_password, ''), COALESCE(vnc_password, ''), COALESCE(qmp_socket_path, ''),
	COALESCE(status, 'stopped'), pid,
	COALESCE(autostart, false), COALESCE(autostart_delay, 0), COALESCE(tags, ''),
	COALESCE(os_type, ''), COALESCE(os_version, ''), template_id,
	created_by, created_at, updated_at, last_started_at`

func scanVM(row interface{ Scan(...interface{}) error }) (*VirtualMachine, error) {
	var vm VirtualMachine
	err := row.Scan(
		&vm.ID, &vm.Name, &vm.Description, &vm.UUID, &vm.CPUCores, &vm.RAMMB,
		&vm.CPUType, &vm.CPUPinning, &vm.NUMATopology,
		&vm.BalloonEnabled, &vm.HugepagesEnabled,
		&vm.DiskPath, &vm.DiskSizeGB, &vm.DiskFormat,
		&vm.CacheMode, &vm.DiscardEnabled,
		&vm.BootOrder, &vm.ISOPath, &vm.BootFromDisk,
		&vm.PhysicalDiskDevice, &vm.FirmwareType,
		&vm.SecureBoot, &vm.TPMEnabled,
		&vm.NetworkMode, &vm.NetworkBridge, &vm.MACAddress,
		&vm.NetworkModel, &vm.VLANID, &vm.BandwidthLimitDown, &vm.BandwidthLimitUp,
		&vm.DisplayType, &vm.SpicePort, &vm.VNCPort,
		&vm.SpicePassword, &vm.VNCPassword, &vm.QMPSocketPath,
		&vm.Status, &vm.PID,
		&vm.Autostart, &vm.AutostartDelay, &vm.Tags,
		&vm.OSType, &vm.OSVersion, &vm.TemplateID,
		&vm.CreatedBy, &vm.CreatedAt, &vm.UpdatedAt, &vm.LastStartedAt,
	)
	return &vm, err
}

func ListVMsHandler(w http.ResponseWriter, r *http.Request) {
	db, err := NewDatabase()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	rows, err := db.Query("SELECT " + vmFields + " FROM virtual_machines ORDER BY name")
	if err != nil {
		http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var vms []VirtualMachine
	for rows.Next() {
		vm, err := scanVM(rows)
		if err != nil {
			continue
		}
		vms = append(vms, *vm)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"vms":     vms,
	})
}

func CreateVMHandler(w http.ResponseWriter, r *http.Request) {
	var req VirtualMachine
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

	// Generate defaults
	req.UUID = generateUUID()
	if req.MACAddress == "" {
		req.MACAddress = generateMACAddress()
	}
	if req.DiskPath == "" {
		req.DiskPath = filepath.Join(VMDir, req.Name+".qcow2")
	}
	if req.DiskFormat == "" {
		req.DiskFormat = "qcow2"
	}
	if req.CPUType == "" {
		req.CPUType = "host"
	}
	if req.NetworkModel == "" {
		req.NetworkModel = "virtio"
	}
	if req.CacheMode == "" {
		req.CacheMode = "writeback"
	}
	if req.FirmwareType == "" {
		req.FirmwareType = "bios"
	}
	if req.SpicePort == 0 {
		req.SpicePort = allocatePort(db, "spice")
	}
	if req.VNCPort == 0 {
		req.VNCPort = allocatePort(db, "vnc")
	}
	if req.SpicePassword == "" {
		req.SpicePassword = generatePassword()
	}
	if req.VNCPassword == "" {
		req.VNCPassword = generatePassword()
	}
	req.QMPSocketPath = filepath.Join(QMPSocketDir, req.UUID+".sock")

	user, _ := getCurrentUser(r)
	var createdBy *int
	if user != nil {
		createdBy = &user.ID
	}

	// Create directories
	os.MkdirAll(VMDir, 0755)
	os.MkdirAll(QMPSocketDir, 0755)
	os.MkdirAll(VMLogDir, 0755)

	// Create disk image if needed
	if req.DiskSizeGB > 0 {
		createDiskImage(req.DiskPath, req.DiskSizeGB, req.DiskFormat)
	}

	result, err := db.Exec(
		`INSERT INTO virtual_machines (name, description, uuid, cpu_cores, ram_mb,
		 cpu_type, cpu_pinning, numa_topology, balloon_enabled, hugepages_enabled,
		 disk_path, disk_size_gb, disk_format, cache_mode, discard_enabled,
		 boot_order, iso_path, boot_from_disk, physical_disk_device,
		 firmware_type, secure_boot, tpm_enabled,
		 network_mode, network_bridge, mac_address, network_model, vlan_id,
		 bandwidth_limit_down, bandwidth_limit_up,
		 display_type, spice_port, vnc_port, spice_password, vnc_password, qmp_socket_path,
		 autostart, autostart_delay, tags, os_type, os_version, template_id,
		 status, created_by)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'stopped', ?)`,
		req.Name, req.Description, req.UUID, req.CPUCores, req.RAMMB,
		req.CPUType, req.CPUPinning, req.NUMATopology, req.BalloonEnabled, req.HugepagesEnabled,
		req.DiskPath, req.DiskSizeGB, req.DiskFormat, req.CacheMode, req.DiscardEnabled,
		req.BootOrder, req.ISOPath, req.BootFromDisk, req.PhysicalDiskDevice,
		req.FirmwareType, req.SecureBoot, req.TPMEnabled,
		req.NetworkMode, req.NetworkBridge, req.MACAddress, req.NetworkModel, req.VLANID,
		req.BandwidthLimitDown, req.BandwidthLimitUp,
		req.DisplayType, req.SpicePort, req.VNCPort, req.SpicePassword, req.VNCPassword, req.QMPSocketPath,
		req.Autostart, req.AutostartDelay, req.Tags, req.OSType, req.OSVersion, req.TemplateID,
		createdBy,
	)
	if err != nil {
		http.Error(w, "Failed to create VM: "+err.Error(), http.StatusBadRequest)
		return
	}

	id, _ := result.LastInsertId()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"vm_id":   id,
	})
}

func GetVMHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	db, err := NewDatabase()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	vm, err := scanVM(db.QueryRow("SELECT "+vmFields+" FROM virtual_machines WHERE id = ?", id))
	if err != nil {
		http.Error(w, "VM not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"vm":      vm,
	})
}

func UpdateVMHandler(w http.ResponseWriter, r *http.Request) {
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

	// Check if VM is running
	var status string
	err = db.QueryRow("SELECT status FROM virtual_machines WHERE id = ?", id).Scan(&status)
	if err != nil {
		http.Error(w, "VM not found", http.StatusNotFound)
		return
	}
	if status == "running" {
		http.Error(w, "Cannot update VM while it is running", http.StatusBadRequest)
		return
	}

	updates := []string{}
	values := []interface{}{}

	allowedFields := map[string]bool{
		"name": true, "description": true, "cpu_cores": true, "ram_mb": true,
		"cpu_type": true, "cpu_pinning": true, "numa_topology": true,
		"balloon_enabled": true, "hugepages_enabled": true,
		"cache_mode": true, "discard_enabled": true,
		"boot_order": true, "iso_path": true, "boot_from_disk": true,
		"physical_disk_device": true, "firmware_type": true,
		"secure_boot": true, "tpm_enabled": true,
		"network_mode": true, "network_bridge": true, "network_model": true,
		"vlan_id": true, "bandwidth_limit_down": true, "bandwidth_limit_up": true,
		"display_type": true, "autostart": true, "autostart_delay": true,
		"tags": true, "os_type": true, "os_version": true,
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
	query := "UPDATE virtual_machines SET " + strings.Join(updates, ", ") + " WHERE id = ?"
	_, err = db.Exec(query, values...)
	if err != nil {
		http.Error(w, "Failed to update VM", http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func DeleteVMHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	db, err := NewDatabase()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	vm, err := scanVM(db.QueryRow("SELECT "+vmFields+" FROM virtual_machines WHERE id = ?", id))
	if err != nil {
		http.Error(w, "VM not found", http.StatusNotFound)
		return
	}

	// Stop VM if running
	if vm.Status == "running" {
		stopVM(id, true)
	}

	// Delete disk image
	if vm.DiskPath != "" {
		exec.Command("rm", "-f", vm.DiskPath).Run()
	}

	// Delete QMP socket
	if vm.QMPSocketPath != "" {
		os.Remove(vm.QMPSocketPath)
	}

	_, err = db.Exec("DELETE FROM virtual_machines WHERE id = ?", id)
	if err != nil {
		http.Error(w, "Failed to delete VM", http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func StartVMHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	db, err := NewDatabase()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	vm, err := scanVM(db.QueryRow("SELECT "+vmFields+" FROM virtual_machines WHERE id = ?", id))
	if err != nil {
		http.Error(w, "VM not found", http.StatusNotFound)
		return
	}

	if vm.Status == "running" {
		http.Error(w, "VM is already running", http.StatusBadRequest)
		return
	}

	// Build and execute QEMU command
	qemuCmd := buildQEMUCommand(*vm)
	logFile := filepath.Join(VMLogDir, vm.Name+".log")

	// Ensure log dir exists
	os.MkdirAll(VMLogDir, 0755)

	// Execute QEMU in background
	execCmd := exec.Command("bash", "-c", fmt.Sprintf("nohup %s > %s 2>&1 & echo $!", qemuCmd, logFile))
	output, err := execCmd.Output()
	if err != nil {
		http.Error(w, "Failed to start VM: "+err.Error(), http.StatusInternalServerError)
		return
	}

	pidStr := strings.TrimSpace(string(output))
	pid, _ := strconv.Atoi(pidStr)

	db.Exec("UPDATE virtual_machines SET status = 'running', pid = ?, last_started_at = NOW() WHERE id = ?", pid, id)

	// Apply bandwidth limiting if configured
	if vm.BandwidthLimitDown != nil || vm.BandwidthLimitUp != nil {
		applyVMBandwidthLimit(pid, vm.BandwidthLimitDown, vm.BandwidthLimitUp)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"pid":     pid,
	})
}

func StopVMHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	var req struct {
		Force bool `json:"force"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	stopVM(id, req.Force)

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func RestartVMHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	db, err := NewDatabase()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Get VM info first
	vm, err := scanVM(db.QueryRow("SELECT "+vmFields+" FROM virtual_machines WHERE id = ?", id))
	if err != nil {
		http.Error(w, "VM not found", http.StatusNotFound)
		return
	}

	// Stop VM if running
	if vm.Status == "running" {
		stopVM(id, false)
		// Wait for VM to stop
		time.Sleep(2 * time.Second)
	}

	// Start VM again
	qemuCmd := buildQEMUCommand(*vm)
	logFile := filepath.Join(VMLogDir, vm.Name+".log")

	execCmd := exec.Command("bash", "-c", fmt.Sprintf("nohup %s > %s 2>&1 & echo $!", qemuCmd, logFile))
	output, err := execCmd.Output()
	if err != nil {
		http.Error(w, "Failed to restart VM: "+err.Error(), http.StatusInternalServerError)
		return
	}

	pidStr := strings.TrimSpace(string(output))
	pid, _ := strconv.Atoi(pidStr)

	db.Exec("UPDATE virtual_machines SET status = 'running', pid = ?, last_started_at = NOW() WHERE id = ?", pid, id)

	// Apply bandwidth limiting if configured
	if vm.BandwidthLimitDown != nil || vm.BandwidthLimitUp != nil {
		applyVMBandwidthLimit(pid, vm.BandwidthLimitDown, vm.BandwidthLimitUp)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"pid":     pid,
	})
}

func GetVMStatusHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	db, err := NewDatabase()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	var status string
	var pid sql.NullInt64
	err = db.QueryRow("SELECT status, pid FROM virtual_machines WHERE id = ?", id).Scan(&status, &pid)
	if err != nil {
		http.Error(w, "VM not found", http.StatusNotFound)
		return
	}

	// Check if process is still running
	if status == "running" && pid.Valid {
		cmd := exec.Command("ps", "-p", strconv.FormatInt(pid.Int64, 10))
		if cmd.Run() != nil {
			// Process died
			db.Exec("UPDATE virtual_machines SET status = 'stopped', pid = NULL WHERE id = ?", id)
			status = "stopped"
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"status":  status,
	})
}

func GetVMLogsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])
	lines := r.URL.Query().Get("lines")
	if lines == "" {
		lines = "100"
	}

	db, err := NewDatabase()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	var vmName string
	err = db.QueryRow("SELECT name FROM virtual_machines WHERE id = ?", id).Scan(&vmName)
	if err != nil {
		http.Error(w, "VM not found", http.StatusNotFound)
		return
	}

	logFile := filepath.Join(VMLogDir, vmName+".log")
	cmd := exec.Command("tail", "-n", lines, logFile)
	output, _ := cmd.Output()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"logs":    string(output),
	})
}

func GetVMSpiceHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	db, err := NewDatabase()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	vm, err := scanVM(db.QueryRow("SELECT "+vmFields+" FROM virtual_machines WHERE id = ?", id))
	if err != nil {
		http.Error(w, "VM not found", http.StatusNotFound)
		return
	}

	spiceFile := fmt.Sprintf(`[virt-viewer]
type=spice
host=localhost
port=%d
password=%s
title=%s
delete-this-file=1
fullscreen=0
`, vm.SpicePort, vm.SpicePassword, vm.Name)

	w.Header().Set("Content-Type", "application/x-virt-viewer")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.vv", vm.Name))
	w.Write([]byte(spiceFile))
}

// ISO Management Handlers

func ListISOsHandler(w http.ResponseWriter, r *http.Request) {
	db, err := NewDatabase()
	if err != nil {
		// Fallback to filesystem scan
		listISOsFromFilesystem(w)
		return
	}
	defer db.Close()

	rows, err := db.Query(`SELECT id, name, filename, file_path, file_size, checksum_sha256,
		os_type, os_version, description, download_url, download_status, download_progress,
		download_error, is_predefined, is_verified, created_by, created_at, updated_at
		FROM iso_library ORDER BY name`)
	if err != nil {
		listISOsFromFilesystem(w)
		return
	}
	defer rows.Close()

	var isos []ISOLibrary
	for rows.Next() {
		var iso ISOLibrary
		rows.Scan(&iso.ID, &iso.Name, &iso.Filename, &iso.FilePath, &iso.FileSize,
			&iso.ChecksumSHA256, &iso.OSType, &iso.OSVersion, &iso.Description,
			&iso.DownloadURL, &iso.DownloadStatus, &iso.DownloadProgress,
			&iso.DownloadError, &iso.IsPredefined, &iso.IsVerified,
			&iso.CreatedBy, &iso.CreatedAt, &iso.UpdatedAt)
		isos = append(isos, iso)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"isos":    isos,
	})
}

func listISOsFromFilesystem(w http.ResponseWriter) {
	cmd := exec.Command("find", ISODir, "-name", "*.iso", "-type", "f")
	output, _ := cmd.Output()

	var isos []map[string]interface{}
	for _, line := range strings.Split(string(output), "\n") {
		if line != "" {
			info, _ := os.Stat(line)
			var size int64
			if info != nil {
				size = info.Size()
			}
			isos = append(isos, map[string]interface{}{
				"name":      filepath.Base(line),
				"filename":  filepath.Base(line),
				"file_path": line,
				"file_size": size,
			})
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"isos":    isos,
	})
}

func UploadISOHandler(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form (max 10GB)
	err := r.ParseMultipartForm(10 << 30)
	if err != nil {
		http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "No file provided", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Validate file extension
	if !strings.HasSuffix(strings.ToLower(header.Filename), ".iso") {
		http.Error(w, "Only ISO files are allowed", http.StatusBadRequest)
		return
	}

	// Create ISO directory if not exists
	os.MkdirAll(ISODir, 0755)

	// Create destination file
	destPath := filepath.Join(ISODir, header.Filename)
	destFile, err := os.Create(destPath)
	if err != nil {
		http.Error(w, "Failed to create file: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer destFile.Close()

	// Copy file
	written, err := io.Copy(destFile, file)
	if err != nil {
		os.Remove(destPath)
		http.Error(w, "Failed to write file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Save to database
	db, _ := NewDatabase()
	if db != nil {
		defer db.Close()
		user, _ := getCurrentUser(r)
		var createdBy *int
		if user != nil {
			createdBy = &user.ID
		}
		db.Exec(`INSERT INTO iso_library (name, filename, file_path, file_size, download_status, created_by)
			VALUES (?, ?, ?, ?, 'completed', ?)`,
			strings.TrimSuffix(header.Filename, ".iso"), header.Filename, destPath, written, createdBy)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"filename":  header.Filename,
		"file_path": destPath,
		"file_size": written,
	})
}

func DownloadISOFromURLHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL      string `json:"url"`
		Filename string `json:"filename"`
		OSType   string `json:"os_type"`
		OSVersion string `json:"os_version"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	// Determine filename
	filename := req.Filename
	if filename == "" {
		parts := strings.Split(req.URL, "/")
		filename = parts[len(parts)-1]
		if !strings.HasSuffix(filename, ".iso") {
			filename += ".iso"
		}
	}

	os.MkdirAll(ISODir, 0755)
	destPath := filepath.Join(ISODir, filename)

	db, _ := NewDatabase()
	var isoID int64
	if db != nil {
		defer db.Close()
		user, _ := getCurrentUser(r)
		var createdBy *int
		if user != nil {
			createdBy = &user.ID
		}
		result, _ := db.Exec(`INSERT INTO iso_library (name, filename, file_path, download_url, os_type, os_version, download_status, created_by)
			VALUES (?, ?, ?, ?, ?, ?, 'downloading', ?)`,
			strings.TrimSuffix(filename, ".iso"), filename, destPath, req.URL, req.OSType, req.OSVersion, createdBy)
		isoID, _ = result.LastInsertId()
	}

	// Start download in background
	go downloadISO(int(isoID), req.URL, destPath)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"iso_id":   isoID,
		"filename": filename,
		"status":   "downloading",
	})
}

func downloadISO(isoID int, url, destPath string) {
	isoDownloadsLock.Lock()
	progress := &ISODownloadProgress{
		ID:       isoID,
		URL:      url,
		Filename: filepath.Base(destPath),
		Status:   "downloading",
	}
	isoDownloads[isoID] = progress
	isoDownloadsLock.Unlock()

	db, _ := NewDatabase()

	// Use wget for reliable downloads with progress
	cmd := exec.Command("wget", "-O", destPath, url)
	err := cmd.Run()

	if err != nil {
		progress.Status = "failed"
		progress.Error = err.Error()
		if db != nil {
			db.Exec("UPDATE iso_library SET download_status = 'failed', download_error = ? WHERE id = ?", err.Error(), isoID)
			db.Close()
		}
		return
	}

	// Get file size
	info, _ := os.Stat(destPath)
	var size int64
	if info != nil {
		size = info.Size()
	}

	progress.Status = "completed"
	progress.Percent = 100
	progress.Total = size
	progress.Current = size

	if db != nil {
		db.Exec("UPDATE iso_library SET download_status = 'completed', download_progress = 100, file_size = ? WHERE id = ?", size, isoID)
		db.Close()
	}
}

func GetISODownloadProgressHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	isoDownloadsLock.RLock()
	progress, exists := isoDownloads[id]
	isoDownloadsLock.RUnlock()

	if !exists {
		// Check database
		db, _ := NewDatabase()
		if db != nil {
			defer db.Close()
			var status string
			var prog int
			err := db.QueryRow("SELECT download_status, download_progress FROM iso_library WHERE id = ?", id).Scan(&status, &prog)
			if err == nil {
				json.NewEncoder(w).Encode(map[string]interface{}{
					"success":  true,
					"status":   status,
					"progress": prog,
				})
				return
			}
		}
		http.Error(w, "Download not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"progress": progress,
	})
}

func DeleteISOHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	db, err := NewDatabase()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	var filePath string
	err = db.QueryRow("SELECT file_path FROM iso_library WHERE id = ?", id).Scan(&filePath)
	if err != nil {
		http.Error(w, "ISO not found", http.StatusNotFound)
		return
	}

	// Delete file
	os.Remove(filePath)

	// Delete from database
	db.Exec("DELETE FROM iso_library WHERE id = ?", id)

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func GetPredefinedISOsHandler(w http.ResponseWriter, r *http.Request) {
	predefined := []map[string]interface{}{
		{
			"name":       "Ubuntu 24.04 LTS",
			"url":        "https://releases.ubuntu.com/24.04/ubuntu-24.04-live-server-amd64.iso",
			"os_type":    "linux",
			"os_version": "ubuntu-24.04",
		},
		{
			"name":       "Ubuntu 22.04 LTS",
			"url":        "https://releases.ubuntu.com/22.04/ubuntu-22.04.4-live-server-amd64.iso",
			"os_type":    "linux",
			"os_version": "ubuntu-22.04",
		},
		{
			"name":       "Debian 12",
			"url":        "https://cdimage.debian.org/debian-cd/current/amd64/iso-cd/debian-12.5.0-amd64-netinst.iso",
			"os_type":    "linux",
			"os_version": "debian-12",
		},
		{
			"name":       "Fedora 40 Server",
			"url":        "https://download.fedoraproject.org/pub/fedora/linux/releases/40/Server/x86_64/iso/Fedora-Server-netinst-x86_64-40-1.14.iso",
			"os_type":    "linux",
			"os_version": "fedora-40",
		},
		{
			"name":       "Rocky Linux 9",
			"url":        "https://download.rockylinux.org/pub/rocky/9/isos/x86_64/Rocky-9.3-x86_64-minimal.iso",
			"os_type":    "linux",
			"os_version": "rocky-9",
		},
		{
			"name":       "Arch Linux",
			"url":        "https://geo.mirror.pkgbuild.com/iso/latest/archlinux-x86_64.iso",
			"os_type":    "linux",
			"os_version": "arch-rolling",
		},
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"isos":    predefined,
	})
}

func ListPhysicalDisksHandler(w http.ResponseWriter, r *http.Request) {
	cmd := exec.Command("lsblk", "-ndo", "NAME,SIZE,TYPE,MODEL")
	output, _ := cmd.Output()

	var disks []map[string]interface{}
	for _, line := range strings.Split(string(output), "\n") {
		if strings.Contains(line, "disk") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				disk := map[string]interface{}{
					"device": "/dev/" + parts[0],
					"size":   parts[1],
				}
				if len(parts) >= 4 {
					disk["model"] = strings.Join(parts[3:], " ")
				}
				disks = append(disks, disk)
			}
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"disks":   disks,
	})
}

func ListNetworkBridgesHandler(w http.ResponseWriter, r *http.Request) {
	cmd := exec.Command("ip", "link", "show", "type", "bridge")
	output, _ := cmd.Output()

	var bridges []string
	for _, line := range strings.Split(string(output), "\n") {
		if strings.Contains(line, ":") && !strings.HasPrefix(line, " ") {
			parts := strings.Fields(line)
			if len(parts) > 1 {
				name := strings.TrimSuffix(parts[1], ":")
				bridges = append(bridges, name)
			}
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"bridges": bridges,
	})
}

// Backup Handlers

func ListVMBackupsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vmID, _ := strconv.Atoi(vars["id"])

	db, err := NewDatabase()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	rows, err := db.Query(`SELECT id, vm_id, vm_name, backup_name, backup_path, backup_size,
		compressed, compression_type, status, created_by, created_at, completed_at, notes
		FROM vm_backups WHERE vm_id = ? ORDER BY created_at DESC`, vmID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var backups []VMBackup
	for rows.Next() {
		var backup VMBackup
		rows.Scan(&backup.ID, &backup.VMID, &backup.VMName, &backup.BackupName,
			&backup.BackupPath, &backup.BackupSize, &backup.Compressed,
			&backup.CompressionType, &backup.Status, &backup.CreatedBy,
			&backup.CreatedAt, &backup.CompletedAt, &backup.Notes)
		backups = append(backups, backup)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"backups": backups,
	})
}

func CreateVMBackupHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vmID, _ := strconv.Atoi(vars["id"])

	var req struct {
		Notes string `json:"notes"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	db, err := NewDatabase()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	vm, err := scanVM(db.QueryRow("SELECT "+vmFields+" FROM virtual_machines WHERE id = ?", vmID))
	if err != nil {
		http.Error(w, "VM not found", http.StatusNotFound)
		return
	}

	os.MkdirAll(VMBackupDir, 0755)

	backupName := fmt.Sprintf("%s_%s", vm.Name, time.Now().Format("2006-01-02_15-04-05"))
	backupPath := filepath.Join(VMBackupDir, backupName+".qcow2.gz")

	user, _ := getCurrentUser(r)
	var createdBy *int
	if user != nil {
		createdBy = &user.ID
	}

	result, err := db.Exec(
		`INSERT INTO vm_backups (vm_id, vm_name, backup_name, backup_path, compressed, compression_type, status, created_by, notes)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		vmID, vm.Name, backupName, backupPath, true, "gzip", "creating", createdBy, req.Notes,
	)
	if err != nil {
		http.Error(w, "Failed to create backup record", http.StatusInternalServerError)
		return
	}

	backupID, _ := result.LastInsertId()

	// Start backup in background - FIX: properly pipe gzip output to file
	go func() {
		db2, _ := NewDatabase()
		if db2 == nil {
			return
		}
		defer db2.Close()

		// Create backup using gzip compression
		cmd := exec.Command("bash", "-c", fmt.Sprintf("gzip -c '%s' > '%s'", vm.DiskPath, backupPath))
		err := cmd.Run()

		if err != nil {
			db2.Exec("UPDATE vm_backups SET status = 'failed' WHERE id = ?", backupID)
			return
		}

		// Get backup size
		info, _ := os.Stat(backupPath)
		var size int64
		if info != nil {
			size = info.Size()
		}

		db2.Exec("UPDATE vm_backups SET status = 'completed', backup_size = ?, completed_at = NOW() WHERE id = ?", size, backupID)
	}()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"backup_id": backupID,
	})
}

func CheckBackupStatusHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	backupID, _ := strconv.Atoi(vars["backupId"])

	db, err := NewDatabase()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	var status string
	var size sql.NullInt64
	err = db.QueryRow("SELECT status, backup_size FROM vm_backups WHERE id = ?", backupID).Scan(&status, &size)
	if err != nil {
		http.Error(w, "Backup not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"status":  status,
		"size":    size.Int64,
	})
}

func RestoreBackupHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	backupID, _ := strconv.Atoi(vars["backupId"])

	db, err := NewDatabase()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	var backup VMBackup
	err = db.QueryRow(`SELECT id, vm_id, vm_name, backup_name, backup_path, backup_size,
		compressed, compression_type, status, created_by, created_at, completed_at, notes
		FROM vm_backups WHERE id = ?`, backupID).Scan(
		&backup.ID, &backup.VMID, &backup.VMName, &backup.BackupName, &backup.BackupPath,
		&backup.BackupSize, &backup.Compressed, &backup.CompressionType, &backup.Status,
		&backup.CreatedBy, &backup.CreatedAt, &backup.CompletedAt, &backup.Notes)
	if err != nil {
		http.Error(w, "Backup not found", http.StatusNotFound)
		return
	}

	// Get VM disk path
	var diskPath string
	err = db.QueryRow("SELECT disk_path FROM virtual_machines WHERE id = ?", backup.VMID).Scan(&diskPath)
	if err != nil {
		http.Error(w, "VM not found", http.StatusNotFound)
		return
	}

	// Check if VM is running
	var status string
	db.QueryRow("SELECT status FROM virtual_machines WHERE id = ?", backup.VMID).Scan(&status)
	if status == "running" {
		http.Error(w, "Cannot restore while VM is running", http.StatusBadRequest)
		return
	}

	db.Exec("UPDATE vm_backups SET status = 'restoring' WHERE id = ?", backupID)

	// Restore in background
	go func() {
		db2, _ := NewDatabase()
		if db2 == nil {
			return
		}
		defer db2.Close()

		var cmd *exec.Cmd
		if backup.Compressed {
			cmd = exec.Command("bash", "-c", fmt.Sprintf("gunzip -c '%s' > '%s'", backup.BackupPath, diskPath))
		} else {
			cmd = exec.Command("cp", backup.BackupPath, diskPath)
		}

		err := cmd.Run()
		if err != nil {
			db2.Exec("UPDATE vm_backups SET status = 'failed' WHERE id = ?", backupID)
			return
		}

		db2.Exec("UPDATE vm_backups SET status = 'completed' WHERE id = ?", backupID)
	}()

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func DeleteBackupHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	backupID, _ := strconv.Atoi(vars["backupId"])

	db, err := NewDatabase()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	var backupPath string
	err = db.QueryRow("SELECT backup_path FROM vm_backups WHERE id = ?", backupID).Scan(&backupPath)
	if err != nil {
		http.Error(w, "Backup not found", http.StatusNotFound)
		return
	}

	exec.Command("rm", "-f", backupPath).Run()
	db.Exec("DELETE FROM vm_backups WHERE id = ?", backupID)

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// Helper functions

func generateUUID() string {
	return fmt.Sprintf("%04x%04x-%04x-%04x-%04x-%04x%04x%04x",
		rand.Intn(0xffff), rand.Intn(0xffff), rand.Intn(0xffff),
		rand.Intn(0x0fff)|0x4000, rand.Intn(0x3fff)|0x8000,
		rand.Intn(0xffff), rand.Intn(0xffff), rand.Intn(0xffff))
}

func generateMACAddress() string {
	return fmt.Sprintf("52:54:00:%02x:%02x:%02x", rand.Intn(0xff), rand.Intn(0xff), rand.Intn(0xff))
}

func generatePassword() string {
	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	password := make([]byte, 12)
	for i := range password {
		password[i] = chars[rand.Intn(len(chars))]
	}
	return string(password)
}

func allocatePort(db *Database, portType string) int {
	startPort := 5900
	if portType == "vnc" {
		startPort = 5950
	}

	for port := startPort; port <= startPort+100; port++ {
		var count int
		column := "spice_port"
		if portType == "vnc" {
			column = "vnc_port"
		}
		db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM virtual_machines WHERE %s = ?", column), port).Scan(&count)
		if count == 0 {
			return port
		}
	}
	return startPort
}

func createDiskImage(path string, sizeGB int, format string) {
	if format == "" {
		format = "qcow2"
	}
	os.MkdirAll(filepath.Dir(path), 0755)
	exec.Command("qemu-img", "create", "-f", format, path, fmt.Sprintf("%dG", sizeGB)).Run()
}

func buildQEMUCommand(vm VirtualMachine) string {
	cmd := []string{
		"qemu-system-x86_64",
		"-enable-kvm",
		"-machine", "type=q35,accel=kvm",
		"-uuid", vm.UUID,
		"-name", vm.Name,
	}

	// CPU configuration
	cpuType := vm.CPUType
	if cpuType == "" {
		cpuType = "host"
	}
	cmd = append(cmd, "-cpu", cpuType)
	cmd = append(cmd, "-smp", fmt.Sprintf("cores=%d", vm.CPUCores))

	// Memory configuration
	cmd = append(cmd, "-m", fmt.Sprintf("%d", vm.RAMMB))

	// Memory balloon
	if vm.BalloonEnabled {
		cmd = append(cmd, "-device", "virtio-balloon-pci,id=balloon0")
	}

	// Hugepages
	if vm.HugepagesEnabled {
		cmd = append(cmd, "-mem-path", "/dev/hugepages")
		cmd = append(cmd, "-mem-prealloc")
	}

	// UEFI/BIOS configuration
	if vm.FirmwareType == "uefi" {
		// Check if OVMF is available
		if _, err := os.Stat(OVMFPath); err == nil {
			vmVarsPath := filepath.Join(VMDir, vm.Name+"_VARS.fd")
			// Copy OVMF_VARS if not exists
			if _, err := os.Stat(vmVarsPath); os.IsNotExist(err) {
				exec.Command("cp", OVMFVarsPath, vmVarsPath).Run()
			}
			cmd = append(cmd, "-drive", fmt.Sprintf("if=pflash,format=raw,readonly=on,file=%s", OVMFPath))
			cmd = append(cmd, "-drive", fmt.Sprintf("if=pflash,format=raw,file=%s", vmVarsPath))
		}
	}

	// TPM support
	if vm.TPMEnabled {
		tpmSocketPath := filepath.Join(QMPSocketDir, vm.UUID+"-tpm.sock")
		// Start swtpm if not running
		cmd = append(cmd, "-chardev", fmt.Sprintf("socket,id=chrtpm,path=%s", tpmSocketPath))
		cmd = append(cmd, "-tpmdev", "emulator,id=tpm0,chardev=chrtpm")
		cmd = append(cmd, "-device", "tpm-tis,tpmdev=tpm0")
	}

	// Disk configuration
	if vm.DiskPath != "" {
		diskFormat := vm.DiskFormat
		if diskFormat == "" {
			diskFormat = "qcow2"
		}
		cacheMode := vm.CacheMode
		if cacheMode == "" {
			cacheMode = "writeback"
		}

		diskOpts := fmt.Sprintf("file=%s,if=virtio,format=%s,cache=%s", vm.DiskPath, diskFormat, cacheMode)
		if vm.DiscardEnabled {
			diskOpts += ",discard=unmap"
		}
		cmd = append(cmd, "-drive", diskOpts)
	}

	// Physical disk passthrough
	if vm.PhysicalDiskDevice != "" {
		cmd = append(cmd, "-drive", fmt.Sprintf("file=%s,if=virtio,format=raw", vm.PhysicalDiskDevice))
	}

	// ISO/CDROM
	if vm.ISOPath != "" {
		cmd = append(cmd, "-cdrom", vm.ISOPath)
	}

	// Boot order
	bootOrder := vm.BootOrder
	if bootOrder == "" {
		bootOrder = "cd,hd"
	}
	cmd = append(cmd, "-boot", "order="+bootOrder)

	// Network configuration
	switch vm.NetworkMode {
	case "nat":
		cmd = append(cmd, "-netdev", "user,id=net0")
		cmd = append(cmd, "-device", fmt.Sprintf("%s-net-pci,netdev=net0,mac=%s", vm.NetworkModel, vm.MACAddress))
	case "bridge":
		if vm.NetworkBridge != "" {
			tapName := fmt.Sprintf("tap_%s", vm.Name[:min(10, len(vm.Name))])
			cmd = append(cmd, "-netdev", fmt.Sprintf("tap,id=net0,ifname=%s,script=no,downscript=no", tapName))
			cmd = append(cmd, "-device", fmt.Sprintf("%s-net-pci,netdev=net0,mac=%s", vm.NetworkModel, vm.MACAddress))
		}
	case "user":
		cmd = append(cmd, "-netdev", "user,id=net0,hostfwd=tcp::2222-:22")
		cmd = append(cmd, "-device", fmt.Sprintf("%s-net-pci,netdev=net0,mac=%s", vm.NetworkModel, vm.MACAddress))
	}

	// Display configuration
	switch vm.DisplayType {
	case "spice":
		spiceOpts := fmt.Sprintf("port=%d,addr=0.0.0.0", vm.SpicePort)
		if vm.SpicePassword != "" {
			spiceOpts += ",password=" + vm.SpicePassword
		} else {
			spiceOpts += ",disable-ticketing=on"
		}
		cmd = append(cmd, "-spice", spiceOpts)
		cmd = append(cmd, "-vga", "qxl")
		cmd = append(cmd, "-device", "virtio-serial-pci")
		cmd = append(cmd, "-chardev", "spicevmc,id=vdagent,name=vdagent")
		cmd = append(cmd, "-device", "virtserialport,chardev=vdagent,name=com.redhat.spice.0")
	case "vnc":
		vncDisplay := vm.VNCPort - 5900
		if vncDisplay < 0 {
			vncDisplay = 0
		}
		vncOpts := fmt.Sprintf("0.0.0.0:%d", vncDisplay)
		if vm.VNCPassword != "" {
			vncOpts += ",password=on"
		}
		cmd = append(cmd, "-vnc", vncOpts)
		cmd = append(cmd, "-vga", "std")
	case "none":
		cmd = append(cmd, "-nographic")
	default:
		// Default to both SPICE and VNC
		cmd = append(cmd, "-spice", fmt.Sprintf("port=%d,addr=0.0.0.0,disable-ticketing=on", vm.SpicePort))
		cmd = append(cmd, "-vga", "qxl")
		vncDisplay := vm.VNCPort - 5900
		cmd = append(cmd, "-vnc", fmt.Sprintf("0.0.0.0:%d", vncDisplay))
	}

	// QMP socket for management
	if vm.QMPSocketPath != "" {
		os.MkdirAll(filepath.Dir(vm.QMPSocketPath), 0755)
		cmd = append(cmd, "-qmp", fmt.Sprintf("unix:%s,server,nowait", vm.QMPSocketPath))
	}

	// USB controller
	cmd = append(cmd, "-usb")
	cmd = append(cmd, "-device", "usb-tablet")

	// Audio (optional, PulseAudio)
	// cmd = append(cmd, "-audiodev", "pa,id=snd0")
	// cmd = append(cmd, "-device", "intel-hda")
	// cmd = append(cmd, "-device", "hda-output,audiodev=snd0")

	// Daemonize
	cmd = append(cmd, "-daemonize")

	return strings.Join(cmd, " ")
}

func stopVM(id int, force bool) {
	db, _ := NewDatabase()
	if db == nil {
		return
	}
	defer db.Close()

	var pid sql.NullInt64
	db.QueryRow("SELECT pid FROM virtual_machines WHERE id = ?", id).Scan(&pid)

	if pid.Valid {
		signal := "SIGTERM"
		if force {
			signal = "SIGKILL"
		}
		exec.Command("kill", "-s", signal, strconv.FormatInt(pid.Int64, 10)).Run()
		if !force {
			time.Sleep(2 * time.Second)
		}
	}

	db.Exec("UPDATE virtual_machines SET status = 'stopped', pid = NULL WHERE id = ?", id)
}

func applyVMBandwidthLimit(pid int, downloadLimit, uploadLimit *int) {
	// Use the same throttling mechanism as network.go
	if downloadLimit == nil && uploadLimit == nil {
		return
	}

	// This would integrate with the process throttling system from network.go
	// For now, we'll use tc directly on the VM's tap interface
	// This requires the VM to be using bridge networking
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
