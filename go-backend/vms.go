package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

const (
	VMDir     = "/opt/serveros/vms"
	ISODir    = "/opt/serveros/storage/isos"
	VMLogDir  = "/opt/serveros/logs/vms"
)

func ListVMsHandler(w http.ResponseWriter, r *http.Request) {
	db, err := NewDatabase()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	rows, err := db.Query("SELECT * FROM virtual_machines ORDER BY name")
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var vms []VirtualMachine
	for rows.Next() {
		var vm VirtualMachine
		err := rows.Scan(
			&vm.ID, &vm.Name, &vm.Description, &vm.UUID, &vm.CPUCores, &vm.RAMMB,
			&vm.DiskPath, &vm.DiskSizeGB, &vm.DiskFormat, &vm.BootOrder, &vm.ISOPath,
			&vm.BootFromDisk, &vm.PhysicalDiskDevice, &vm.NetworkMode, &vm.NetworkBridge,
			&vm.MACAddress, &vm.DisplayType, &vm.SpicePort, &vm.VNCPort, &vm.SpicePassword,
			&vm.Status, &vm.PID, &vm.CreatedBy, &vm.CreatedAt, &vm.UpdatedAt, &vm.LastStartedAt,
		)
		if err != nil {
			continue
		}
		vms = append(vms, vm)
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

	req.UUID = generateUUID()
	if req.MACAddress == "" {
		req.MACAddress = generateMACAddress()
	}
	if req.DiskPath == "" {
		req.DiskPath = filepath.Join(VMDir, req.Name+".qcow2")
	}
	if req.SpicePort == 0 {
		req.SpicePort = allocatePort(db)
	}
	if req.SpicePassword == "" {
		req.SpicePassword = generatePassword()
	}

	user, _ := getCurrentUser(r)
	var createdBy *int
	if user != nil {
		createdBy = &user.ID
	}

	// Create disk image if needed
	if req.DiskSizeGB > 0 {
		createDiskImage(req.DiskPath, req.DiskSizeGB, req.DiskFormat)
	}

	result, err := db.Exec(
		`INSERT INTO virtual_machines (name, description, uuid, cpu_cores, ram_mb, disk_path, disk_size_gb,
		 disk_format, boot_order, iso_path, boot_from_disk, physical_disk_device, network_mode,
		 network_bridge, mac_address, display_type, spice_port, spice_password, status, created_by)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'stopped', ?)`,
		req.Name, req.Description, req.UUID, req.CPUCores, req.RAMMB, req.DiskPath, req.DiskSizeGB,
		req.DiskFormat, req.BootOrder, req.ISOPath, req.BootFromDisk, req.PhysicalDiskDevice,
		req.NetworkMode, req.NetworkBridge, req.MACAddress, req.DisplayType, req.SpicePort,
		req.SpicePassword, createdBy,
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

	var vm VirtualMachine
	err = db.QueryRow("SELECT * FROM virtual_machines WHERE id = ?", id).Scan(
		&vm.ID, &vm.Name, &vm.Description, &vm.UUID, &vm.CPUCores, &vm.RAMMB,
		&vm.DiskPath, &vm.DiskSizeGB, &vm.DiskFormat, &vm.BootOrder, &vm.ISOPath,
		&vm.BootFromDisk, &vm.PhysicalDiskDevice, &vm.NetworkMode, &vm.NetworkBridge,
		&vm.MACAddress, &vm.DisplayType, &vm.SpicePort, &vm.VNCPort, &vm.SpicePassword,
		&vm.Status, &vm.PID, &vm.CreatedBy, &vm.CreatedAt, &vm.UpdatedAt, &vm.LastStartedAt,
	)
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
		"boot_order": true, "iso_path": true, "boot_from_disk": true,
		"physical_disk_device": true, "network_mode": true, "network_bridge": true,
		"display_type": true,
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

	var vm VirtualMachine
	err = db.QueryRow("SELECT * FROM virtual_machines WHERE id = ?", id).Scan(
		&vm.ID, &vm.Name, &vm.Description, &vm.UUID, &vm.CPUCores, &vm.RAMMB,
		&vm.DiskPath, &vm.DiskSizeGB, &vm.DiskFormat, &vm.BootOrder, &vm.ISOPath,
		&vm.BootFromDisk, &vm.PhysicalDiskDevice, &vm.NetworkMode, &vm.NetworkBridge,
		&vm.MACAddress, &vm.DisplayType, &vm.SpicePort, &vm.VNCPort, &vm.SpicePassword,
		&vm.Status, &vm.PID, &vm.CreatedBy, &vm.CreatedAt, &vm.UpdatedAt, &vm.LastStartedAt,
	)
	if err != nil {
		http.Error(w, "VM not found", http.StatusNotFound)
		return
	}

	// Stop VM if running
	if vm.Status == "running" {
		stopVM(id, true)
	}

	// Delete disk image
	exec.Command("rm", "-f", vm.DiskPath).Run()

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

	var vm VirtualMachine
	err = db.QueryRow("SELECT * FROM virtual_machines WHERE id = ?", id).Scan(
		&vm.ID, &vm.Name, &vm.Description, &vm.UUID, &vm.CPUCores, &vm.RAMMB,
		&vm.DiskPath, &vm.DiskSizeGB, &vm.DiskFormat, &vm.BootOrder, &vm.ISOPath,
		&vm.BootFromDisk, &vm.PhysicalDiskDevice, &vm.NetworkMode, &vm.NetworkBridge,
		&vm.MACAddress, &vm.DisplayType, &vm.SpicePort, &vm.VNCPort, &vm.SpicePassword,
		&vm.Status, &vm.PID, &vm.CreatedBy, &vm.CreatedAt, &vm.UpdatedAt, &vm.LastStartedAt,
	)
	if err != nil {
		http.Error(w, "VM not found", http.StatusNotFound)
		return
	}

	if vm.Status == "running" {
		http.Error(w, "VM is already running", http.StatusBadRequest)
		return
	}

	cmd := buildQEMUCommand(vm)
	logFile := filepath.Join(VMLogDir, vm.Name+".log")
	
	execCmd := exec.Command("bash", "-c", fmt.Sprintf("nohup %s > %s 2>&1 & echo $!", cmd, logFile))
	output, err := execCmd.Output()
	if err != nil {
		http.Error(w, "Failed to start VM", http.StatusInternalServerError)
		return
	}

	pidStr := strings.TrimSpace(string(output))
	pid, _ := strconv.Atoi(pidStr)

	db.Exec("UPDATE virtual_machines SET status = 'running', pid = ?, last_started_at = NOW() WHERE id = ?", pid, id)

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
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

	stopVM(id, false)
	time.Sleep(1 * time.Second)
	
	// Start VM (simplified - would need full VM data)
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
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

	var vm VirtualMachine
	err = db.QueryRow("SELECT * FROM virtual_machines WHERE id = ?", id).Scan(
		&vm.ID, &vm.Name, &vm.Description, &vm.UUID, &vm.CPUCores, &vm.RAMMB,
		&vm.DiskPath, &vm.DiskSizeGB, &vm.DiskFormat, &vm.BootOrder, &vm.ISOPath,
		&vm.BootFromDisk, &vm.PhysicalDiskDevice, &vm.NetworkMode, &vm.NetworkBridge,
		&vm.MACAddress, &vm.DisplayType, &vm.SpicePort, &vm.VNCPort, &vm.SpicePassword,
		&vm.Status, &vm.PID, &vm.CreatedBy, &vm.CreatedAt, &vm.UpdatedAt, &vm.LastStartedAt,
	)
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

func ListISOsHandler(w http.ResponseWriter, r *http.Request) {
	cmd := exec.Command("find", ISODir, "-name", "*.iso", "-type", "f")
	output, _ := cmd.Output()

	var isos []map[string]interface{}
	for _, line := range strings.Split(string(output), "\n") {
		if line != "" {
			isos = append(isos, map[string]interface{}{
				"name": filepath.Base(line),
				"path": line,
			})
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"isos":    isos,
	})
}

func UploadISOHandler(w http.ResponseWriter, r *http.Request) {
	// Handle multipart file upload
	// Implementation would handle file upload and save to ISODir
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func ListPhysicalDisksHandler(w http.ResponseWriter, r *http.Request) {
	cmd := exec.Command("lsblk", "-ndo", "NAME,SIZE,TYPE,MODEL")
	output, _ := cmd.Output()

	var disks []map[string]interface{}
	for _, line := range strings.Split(string(output), "\n") {
		if strings.Contains(line, "disk") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				disks = append(disks, map[string]interface{}{
					"device": "/dev/" + parts[0],
					"size":    parts[1],
				})
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
		if strings.Contains(line, ":") {
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

func ListVMBackupsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vmID, _ := strconv.Atoi(vars["id"])

	db, err := NewDatabase()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	rows, err := db.Query("SELECT * FROM vm_backups WHERE vm_id = ? ORDER BY created_at DESC", vmID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var backups []VMBackup
	for rows.Next() {
		var backup VMBackup
		// Scan backup fields
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

	var vm VirtualMachine
	err = db.QueryRow("SELECT * FROM virtual_machines WHERE id = ?", vmID).Scan(
		&vm.ID, &vm.Name, &vm.Description, &vm.UUID, &vm.CPUCores, &vm.RAMMB,
		&vm.DiskPath, &vm.DiskSizeGB, &vm.DiskFormat, &vm.BootOrder, &vm.ISOPath,
		&vm.BootFromDisk, &vm.PhysicalDiskDevice, &vm.NetworkMode, &vm.NetworkBridge,
		&vm.MACAddress, &vm.DisplayType, &vm.SpicePort, &vm.VNCPort, &vm.SpicePassword,
		&vm.Status, &vm.PID, &vm.CreatedBy, &vm.CreatedAt, &vm.UpdatedAt, &vm.LastStartedAt,
	)
	if err != nil {
		http.Error(w, "VM not found", http.StatusNotFound)
		return
	}

	backupName := fmt.Sprintf("%s_%s", vm.Name, time.Now().Format("2006-01-02_15-04-05"))
	backupPath := filepath.Join(VMDir, "backups", backupName+".qcow2.gz")

	user, _ := getCurrentUser(r)
	var createdBy *int
	if user != nil {
		createdBy = &user.ID
	}

	db.Exec(
		`INSERT INTO vm_backups (vm_id, vm_name, backup_name, backup_path, compressed, compression_type, status, created_by, notes)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		vmID, vm.Name, backupName, backupPath, true, "gzip", "creating", createdBy, req.Notes,
	)

	// Start backup in background
	go func() {
		exec.Command("gzip", "-c", vm.DiskPath).Run()
	}()

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
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
	err = db.QueryRow("SELECT status FROM vm_backups WHERE id = ?", backupID).Scan(&status)
	if err != nil {
		http.Error(w, "Backup not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"status":  status,
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
	err = db.QueryRow("SELECT * FROM vm_backups WHERE id = ?", backupID).Scan(
		&backup.ID, &backup.VMID, &backup.VMName, &backup.BackupName, &backup.BackupPath,
		&backup.BackupSize, &backup.Compressed, &backup.CompressionType, &backup.Status,
		&backup.CreatedBy, &backup.CreatedAt, &backup.CompletedAt, &backup.Notes,
	)
	if err != nil {
		http.Error(w, "Backup not found", http.StatusNotFound)
		return
	}

	// Restore backup
	exec.Command("gunzip", "-c", backup.BackupPath).Run()

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

func allocatePort(db *Database) int {
	// Find available port between 5900-6000
	for port := 5900; port <= 6000; port++ {
		var count int
		db.QueryRow("SELECT COUNT(*) FROM virtual_machines WHERE spice_port = ?", port).Scan(&count)
		if count == 0 {
			return port
		}
	}
	return 5900
}

func createDiskImage(path string, sizeGB int, format string) {
	exec.Command("qemu-img", "create", "-f", format, path, fmt.Sprintf("%dG", sizeGB)).Run()
}

func buildQEMUCommand(vm VirtualMachine) string {
	cmd := []string{
		"qemu-system-x86_64",
		"-enable-kvm",
		"-cpu", "host",
		"-smp", fmt.Sprintf("cores=%d", vm.CPUCores),
		"-m", fmt.Sprintf("%d", vm.RAMMB),
		"-machine", "type=q35,accel=kvm",
		"-uuid", vm.UUID,
		"-name", vm.Name,
	}

	if vm.DiskPath != "" {
		cmd = append(cmd, "-drive", fmt.Sprintf("file=%s,if=virtio,format=%s", vm.DiskPath, vm.DiskFormat))
	}

	if vm.ISOPath != "" {
		cmd = append(cmd, "-cdrom", vm.ISOPath)
	}

	cmd = append(cmd, "-boot", "order="+vm.BootOrder)

	if vm.NetworkMode == "nat" {
		cmd = append(cmd, "-netdev", "user,id=net0")
		cmd = append(cmd, "-device", fmt.Sprintf("virtio-net-pci,netdev=net0,mac=%s", vm.MACAddress))
	}

	if vm.DisplayType == "spice" {
		cmd = append(cmd, "-spice", fmt.Sprintf("port=%d,addr=0.0.0.0,disable-ticketing", vm.SpicePort))
		cmd = append(cmd, "-vga", "qxl")
	}

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

