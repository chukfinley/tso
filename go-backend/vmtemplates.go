package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// ListVMTemplatesHandler returns all VM templates
func ListVMTemplatesHandler(w http.ResponseWriter, r *http.Request) {
	db, err := NewDatabase()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	rows, err := db.Query(`SELECT id, name, description, cpu_cores, ram_mb, cpu_type,
		disk_size_gb, disk_format, network_mode, display_type, firmware_type,
		os_type, os_version, disk_path, disk_size_actual,
		cloud_init_enabled, COALESCE(cloud_init_user_data, ''),
		COALESCE(cloud_init_meta_data, ''), COALESCE(cloud_init_network_config, ''),
		is_public, download_count, created_by, created_at, updated_at
		FROM vm_templates ORDER BY name`)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var templates []VMTemplate
	for rows.Next() {
		var t VMTemplate
		rows.Scan(&t.ID, &t.Name, &t.Description, &t.CPUCores, &t.RAMMB, &t.CPUType,
			&t.DiskSizeGB, &t.DiskFormat, &t.NetworkMode, &t.DisplayType, &t.FirmwareType,
			&t.OSType, &t.OSVersion, &t.DiskPath, &t.DiskSizeActual,
			&t.CloudInitEnabled, &t.CloudInitUserData,
			&t.CloudInitMetaData, &t.CloudInitNetworkConfig,
			&t.IsPublic, &t.DownloadCount, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt)
		templates = append(templates, t)
	}

	json.NewEncoder(w).Encode(map[string]any{
		"success":   true,
		"templates": templates,
	})
}

// GetVMTemplateHandler returns a single template
func GetVMTemplateHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	db, err := NewDatabase()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	var t VMTemplate
	err = db.QueryRow(`SELECT id, name, description, cpu_cores, ram_mb, cpu_type,
		disk_size_gb, disk_format, network_mode, display_type, firmware_type,
		os_type, os_version, disk_path, disk_size_actual,
		cloud_init_enabled, COALESCE(cloud_init_user_data, ''),
		COALESCE(cloud_init_meta_data, ''), COALESCE(cloud_init_network_config, ''),
		is_public, download_count, created_by, created_at, updated_at
		FROM vm_templates WHERE id = ?`, id).Scan(
		&t.ID, &t.Name, &t.Description, &t.CPUCores, &t.RAMMB, &t.CPUType,
		&t.DiskSizeGB, &t.DiskFormat, &t.NetworkMode, &t.DisplayType, &t.FirmwareType,
		&t.OSType, &t.OSVersion, &t.DiskPath, &t.DiskSizeActual,
		&t.CloudInitEnabled, &t.CloudInitUserData,
		&t.CloudInitMetaData, &t.CloudInitNetworkConfig,
		&t.IsPublic, &t.DownloadCount, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		http.Error(w, "Template not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(map[string]any{
		"success":  true,
		"template": t,
	})
}

// CreateVMTemplateHandler creates a new template (empty or from a VM)
func CreateVMTemplateHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name                   string `json:"name"`
		Description            string `json:"description"`
		CPUCores               int    `json:"cpu_cores"`
		RAMMB                  int    `json:"ram_mb"`
		CPUType                string `json:"cpu_type"`
		DiskSizeGB             int    `json:"disk_size_gb"`
		DiskFormat             string `json:"disk_format"`
		NetworkMode            string `json:"network_mode"`
		DisplayType            string `json:"display_type"`
		FirmwareType           string `json:"firmware_type"`
		OSType                 string `json:"os_type"`
		OSVersion              string `json:"os_version"`
		CloudInitEnabled       bool   `json:"cloud_init_enabled"`
		CloudInitUserData      string `json:"cloud_init_user_data"`
		CloudInitMetaData      string `json:"cloud_init_meta_data"`
		CloudInitNetworkConfig string `json:"cloud_init_network_config"`
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

	result, err := db.Exec(
		`INSERT INTO vm_templates (name, description, cpu_cores, ram_mb, cpu_type,
		 disk_size_gb, disk_format, network_mode, display_type, firmware_type,
		 os_type, os_version, cloud_init_enabled, cloud_init_user_data,
		 cloud_init_meta_data, cloud_init_network_config, created_by)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		req.Name, req.Description, req.CPUCores, req.RAMMB, req.CPUType,
		req.DiskSizeGB, req.DiskFormat, req.NetworkMode, req.DisplayType, req.FirmwareType,
		req.OSType, req.OSVersion, req.CloudInitEnabled, req.CloudInitUserData,
		req.CloudInitMetaData, req.CloudInitNetworkConfig, createdBy,
	)
	if err != nil {
		http.Error(w, "Failed to create template: "+err.Error(), http.StatusBadRequest)
		return
	}

	id, _ := result.LastInsertId()
	json.NewEncoder(w).Encode(map[string]any{
		"success":     true,
		"template_id": id,
	})
}

// SaveVMAsTemplateHandler saves an existing VM as a template
func SaveVMAsTemplateHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vmID, _ := strconv.Atoi(vars["id"])

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		IncludeDisk bool   `json:"include_disk"`
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

	vm, err := scanVM(db.QueryRow("SELECT "+vmFields+" FROM virtual_machines WHERE id = ?", vmID))
	if err != nil {
		http.Error(w, "VM not found", http.StatusNotFound)
		return
	}

	if vm.Status == "running" {
		http.Error(w, "Cannot save template from running VM", http.StatusBadRequest)
		return
	}

	templateName := req.Name
	if templateName == "" {
		templateName = vm.Name + "_template"
	}

	user, _ := getCurrentUser(r)
	var createdBy *int
	if user != nil {
		createdBy = &user.ID
	}

	// Create template directory
	os.MkdirAll(TemplateDir, 0755)

	var diskPath string
	var diskSizeActual int64

	// Copy disk if requested
	if req.IncludeDisk && vm.DiskPath != "" {
		diskPath = filepath.Join(TemplateDir, templateName+".qcow2")

		// Copy disk image
		cmd := exec.Command("cp", vm.DiskPath, diskPath)
		if err := cmd.Run(); err != nil {
			http.Error(w, "Failed to copy disk: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Get disk size
		if info, err := os.Stat(diskPath); err == nil {
			diskSizeActual = info.Size()
		}
	}

	result, err := db.Exec(
		`INSERT INTO vm_templates (name, description, cpu_cores, ram_mb, cpu_type,
		 disk_size_gb, disk_format, network_mode, display_type, firmware_type,
		 os_type, os_version, disk_path, disk_size_actual, created_by)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		templateName, req.Description, vm.CPUCores, vm.RAMMB, vm.CPUType,
		vm.DiskSizeGB, vm.DiskFormat, vm.NetworkMode, vm.DisplayType, vm.FirmwareType,
		vm.OSType, vm.OSVersion, diskPath, diskSizeActual, createdBy,
	)
	if err != nil {
		http.Error(w, "Failed to create template: "+err.Error(), http.StatusBadRequest)
		return
	}

	id, _ := result.LastInsertId()
	json.NewEncoder(w).Encode(map[string]any{
		"success":     true,
		"template_id": id,
	})
}

// CreateVMFromTemplateHandler creates a new VM from a template
func CreateVMFromTemplateHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	templateID, _ := strconv.Atoi(vars["id"])

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		// Optional overrides
		CPUCores   int    `json:"cpu_cores,omitempty"`
		RAMMB      int    `json:"ram_mb,omitempty"`
		DiskSizeGB int    `json:"disk_size_gb,omitempty"`
		ISOPath    string `json:"iso_path,omitempty"`
		// Cloud-init variables
		CloudInitHostname  string `json:"cloud_init_hostname"`
		CloudInitUsername  string `json:"cloud_init_username"`
		CloudInitPassword  string `json:"cloud_init_password"`
		CloudInitSSHKeys   string `json:"cloud_init_ssh_keys"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "VM name is required", http.StatusBadRequest)
		return
	}

	db, err := NewDatabase()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Get template
	var t VMTemplate
	err = db.QueryRow(`SELECT id, name, description, cpu_cores, ram_mb, cpu_type,
		disk_size_gb, disk_format, network_mode, display_type, firmware_type,
		os_type, os_version, disk_path, disk_size_actual,
		cloud_init_enabled, COALESCE(cloud_init_user_data, ''),
		COALESCE(cloud_init_meta_data, ''), COALESCE(cloud_init_network_config, ''),
		is_public, download_count, created_by, created_at, updated_at
		FROM vm_templates WHERE id = ?`, templateID).Scan(
		&t.ID, &t.Name, &t.Description, &t.CPUCores, &t.RAMMB, &t.CPUType,
		&t.DiskSizeGB, &t.DiskFormat, &t.NetworkMode, &t.DisplayType, &t.FirmwareType,
		&t.OSType, &t.OSVersion, &t.DiskPath, &t.DiskSizeActual,
		&t.CloudInitEnabled, &t.CloudInitUserData,
		&t.CloudInitMetaData, &t.CloudInitNetworkConfig,
		&t.IsPublic, &t.DownloadCount, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		http.Error(w, "Template not found", http.StatusNotFound)
		return
	}

	// Apply overrides
	cpuCores := t.CPUCores
	if req.CPUCores > 0 {
		cpuCores = req.CPUCores
	}
	ramMB := t.RAMMB
	if req.RAMMB > 0 {
		ramMB = req.RAMMB
	}
	diskSizeGB := t.DiskSizeGB
	if req.DiskSizeGB > 0 {
		diskSizeGB = req.DiskSizeGB
	}

	// Generate VM parameters
	uuid := generateUUID()
	macAddress := generateMACAddress()
	diskPath := filepath.Join(VMDir, req.Name+".qcow2")
	spicePort := allocatePort(db, "spice")
	vncPort := allocatePort(db, "vnc")
	spicePassword := generatePassword()
	vncPassword := generatePassword()
	qmpSocketPath := filepath.Join(QMPSocketDir, uuid+".sock")

	user, _ := getCurrentUser(r)
	var createdBy *int
	if user != nil {
		createdBy = &user.ID
	}

	// Create disk
	os.MkdirAll(VMDir, 0755)
	os.MkdirAll(QMPSocketDir, 0755)

	if t.DiskPath != "" {
		// Copy template disk as base
		cmd := exec.Command("qemu-img", "create", "-f", "qcow2", "-b", t.DiskPath, "-F", "qcow2", diskPath)
		if err := cmd.Run(); err != nil {
			// Fallback to regular copy
			exec.Command("cp", t.DiskPath, diskPath).Run()
		}
	} else {
		// Create new disk
		createDiskImage(diskPath, diskSizeGB, t.DiskFormat)
	}

	// Create cloud-init ISO if enabled
	var cloudInitISO string
	if t.CloudInitEnabled {
		cloudInitISO = createCloudInitISO(req.Name, req.CloudInitHostname,
			req.CloudInitUsername, req.CloudInitPassword, req.CloudInitSSHKeys,
			t.CloudInitUserData, t.CloudInitMetaData, t.CloudInitNetworkConfig)
	}

	// Use cloud-init ISO or provided ISO
	isoPath := req.ISOPath
	if cloudInitISO != "" && isoPath == "" {
		isoPath = cloudInitISO
	}

	result, err := db.Exec(
		`INSERT INTO virtual_machines (name, description, uuid, cpu_cores, ram_mb,
		 cpu_type, disk_path, disk_size_gb, disk_format, boot_order, iso_path,
		 network_mode, mac_address, network_model, display_type, spice_port, vnc_port,
		 spice_password, vnc_password, qmp_socket_path, firmware_type,
		 os_type, os_version, template_id, status, created_by)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'cd,hd', ?, ?, ?, 'virtio', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'stopped', ?)`,
		req.Name, req.Description, uuid, cpuCores, ramMB,
		t.CPUType, diskPath, diskSizeGB, t.DiskFormat, isoPath,
		t.NetworkMode, macAddress, t.DisplayType, spicePort, vncPort,
		spicePassword, vncPassword, qmpSocketPath, t.FirmwareType,
		t.OSType, t.OSVersion, templateID, createdBy,
	)
	if err != nil {
		http.Error(w, "Failed to create VM: "+err.Error(), http.StatusBadRequest)
		return
	}

	vmID, _ := result.LastInsertId()

	// Update template download count
	db.Exec("UPDATE vm_templates SET download_count = download_count + 1 WHERE id = ?", templateID)

	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"vm_id":   vmID,
	})
}

// DeleteVMTemplateHandler deletes a template
func DeleteVMTemplateHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	db, err := NewDatabase()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	var diskPath string
	err = db.QueryRow("SELECT disk_path FROM vm_templates WHERE id = ?", id).Scan(&diskPath)
	if err != nil {
		http.Error(w, "Template not found", http.StatusNotFound)
		return
	}

	// Delete disk file
	if diskPath != "" {
		os.Remove(diskPath)
	}

	// Delete from database
	db.Exec("DELETE FROM vm_templates WHERE id = ?", id)

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// createCloudInitISO creates a cloud-init ISO for VM configuration
func createCloudInitISO(vmName, hostname, username, password, sshKeys,
	userDataTemplate, metaDataTemplate, networkConfigTemplate string) string {

	// Create temp directory for cloud-init files
	tmpDir := filepath.Join("/tmp", "cloud-init-"+vmName)
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	// Generate user-data
	userData := userDataTemplate
	if userData == "" {
		userData = generateDefaultUserData(hostname, username, password, sshKeys)
	} else {
		// Replace placeholders
		userData = strings.ReplaceAll(userData, "{{hostname}}", hostname)
		userData = strings.ReplaceAll(userData, "{{username}}", username)
		userData = strings.ReplaceAll(userData, "{{password}}", password)
		userData = strings.ReplaceAll(userData, "{{ssh_keys}}", sshKeys)
	}

	// Generate meta-data
	metaData := metaDataTemplate
	if metaData == "" {
		if hostname == "" {
			hostname = vmName
		}
		metaData = fmt.Sprintf("instance-id: %s\nlocal-hostname: %s\n", vmName, hostname)
	}

	// Write files
	os.WriteFile(filepath.Join(tmpDir, "user-data"), []byte(userData), 0644)
	os.WriteFile(filepath.Join(tmpDir, "meta-data"), []byte(metaData), 0644)

	if networkConfigTemplate != "" {
		os.WriteFile(filepath.Join(tmpDir, "network-config"), []byte(networkConfigTemplate), 0644)
	}

	// Create ISO
	isoPath := filepath.Join(VMDir, vmName+"-cloud-init.iso")
	cmd := exec.Command("genisoimage", "-output", isoPath, "-volid", "cidata", "-joliet", "-rock",
		filepath.Join(tmpDir, "user-data"),
		filepath.Join(tmpDir, "meta-data"))

	if networkConfigTemplate != "" {
		cmd.Args = append(cmd.Args, filepath.Join(tmpDir, "network-config"))
	}

	if err := cmd.Run(); err != nil {
		// Try cloud-localds as fallback
		exec.Command("cloud-localds", isoPath,
			filepath.Join(tmpDir, "user-data"),
			filepath.Join(tmpDir, "meta-data")).Run()
	}

	if _, err := os.Stat(isoPath); os.IsNotExist(err) {
		return ""
	}

	return isoPath
}

func generateDefaultUserData(hostname, username, password, sshKeys string) string {
	userData := "#cloud-config\n"

	if hostname != "" {
		userData += fmt.Sprintf("hostname: %s\n", hostname)
	}

	if username != "" || password != "" || sshKeys != "" {
		userData += "users:\n"
		if username == "" {
			username = "user"
		}
		userData += fmt.Sprintf("  - name: %s\n", username)
		userData += "    sudo: ALL=(ALL) NOPASSWD:ALL\n"
		userData += "    shell: /bin/bash\n"

		if password != "" {
			// Hash password
			cmd := exec.Command("openssl", "passwd", "-6", password)
			hashedPass, err := cmd.Output()
			if err == nil {
				userData += fmt.Sprintf("    passwd: %s\n", strings.TrimSpace(string(hashedPass)))
			}
		}

		if sshKeys != "" {
			userData += "    ssh_authorized_keys:\n"
			for _, key := range strings.Split(sshKeys, "\n") {
				key = strings.TrimSpace(key)
				if key != "" {
					userData += fmt.Sprintf("      - %s\n", key)
				}
			}
		}
	}

	userData += "package_update: true\n"
	userData += "package_upgrade: true\n"

	return userData
}

// GetPredefinedTemplatesHandler returns predefined template configurations
func GetPredefinedTemplatesHandler(w http.ResponseWriter, r *http.Request) {
	predefined := []map[string]any{
		{
			"name":          "Ubuntu Server 24.04",
			"description":   "Ubuntu 24.04 LTS Server with minimal installation",
			"cpu_cores":     2,
			"ram_mb":        2048,
			"disk_size_gb":  20,
			"firmware_type": "bios",
			"os_type":       "linux",
			"os_version":    "ubuntu-24.04",
		},
		{
			"name":          "Debian 12 Server",
			"description":   "Debian 12 Bookworm Server",
			"cpu_cores":     2,
			"ram_mb":        2048,
			"disk_size_gb":  20,
			"firmware_type": "bios",
			"os_type":       "linux",
			"os_version":    "debian-12",
		},
		{
			"name":          "Windows 11",
			"description":   "Windows 11 Pro (requires UEFI and TPM)",
			"cpu_cores":     4,
			"ram_mb":        8192,
			"disk_size_gb":  64,
			"firmware_type": "uefi",
			"os_type":       "windows",
			"os_version":    "windows-11",
		},
		{
			"name":          "Windows Server 2022",
			"description":   "Windows Server 2022 Standard",
			"cpu_cores":     4,
			"ram_mb":        4096,
			"disk_size_gb":  40,
			"firmware_type": "uefi",
			"os_type":       "windows",
			"os_version":    "windows-server-2022",
		},
		{
			"name":          "Alpine Linux",
			"description":   "Lightweight Alpine Linux",
			"cpu_cores":     1,
			"ram_mb":        512,
			"disk_size_gb":  4,
			"firmware_type": "bios",
			"os_type":       "linux",
			"os_version":    "alpine",
		},
		{
			"name":          "FreeBSD 14",
			"description":   "FreeBSD 14.0-RELEASE",
			"cpu_cores":     2,
			"ram_mb":        2048,
			"disk_size_gb":  20,
			"firmware_type": "bios",
			"os_type":       "freebsd",
			"os_version":    "freebsd-14",
		},
	}

	json.NewEncoder(w).Encode(map[string]any{
		"success":   true,
		"templates": predefined,
	})
}
