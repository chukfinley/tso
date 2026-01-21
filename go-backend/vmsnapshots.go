package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// ListVMSnapshotsHandler returns all snapshots for a VM
func ListVMSnapshotsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vmID, _ := strconv.Atoi(vars["id"])

	db, err := NewDatabase()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	rows, err := db.Query(`SELECT id, vm_id, name, description, snapshot_type, parent_id,
		size_bytes, status, created_by, created_at, completed_at
		FROM vm_snapshots WHERE vm_id = ? ORDER BY created_at DESC`, vmID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var snapshots []VMSnapshot
	for rows.Next() {
		var snap VMSnapshot
		rows.Scan(&snap.ID, &snap.VMID, &snap.Name, &snap.Description,
			&snap.SnapshotType, &snap.ParentID, &snap.SizeBytes,
			&snap.Status, &snap.CreatedBy, &snap.CreatedAt, &snap.CompletedAt)
		snapshots = append(snapshots, snap)
	}

	// Also get qemu-img snapshot list for qcow2 images
	var diskPath string
	db.QueryRow("SELECT disk_path FROM virtual_machines WHERE id = ?", vmID).Scan(&diskPath)

	qemuSnapshots := listQemuSnapshots(diskPath)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":        true,
		"snapshots":      snapshots,
		"qemu_snapshots": qemuSnapshots,
	})
}

// CreateVMSnapshotHandler creates a new snapshot
func CreateVMSnapshotHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vmID, _ := strconv.Atoi(vars["id"])

	var req struct {
		Name         string `json:"name"`
		Description  string `json:"description"`
		SnapshotType string `json:"snapshot_type"` // disk, memory, full
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		req.Name = fmt.Sprintf("snapshot_%s", time.Now().Format("2006-01-02_15-04-05"))
	}
	if req.SnapshotType == "" {
		req.SnapshotType = "disk"
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

	user, _ := getCurrentUser(r)
	var createdBy *int
	if user != nil {
		createdBy = &user.ID
	}

	result, err := db.Exec(
		`INSERT INTO vm_snapshots (vm_id, name, description, snapshot_type, status, created_by)
		 VALUES (?, ?, ?, ?, 'creating', ?)`,
		vmID, req.Name, req.Description, req.SnapshotType, createdBy,
	)
	if err != nil {
		http.Error(w, "Failed to create snapshot record: "+err.Error(), http.StatusBadRequest)
		return
	}

	snapshotID, _ := result.LastInsertId()

	// Create snapshot in background
	go func() {
		db2, _ := NewDatabase()
		if db2 == nil {
			return
		}
		defer db2.Close()

		var err error
		var size int64

		switch req.SnapshotType {
		case "disk":
			// Use qemu-img snapshot for disk-only snapshot (works for stopped VMs)
			if vm.Status == "running" {
				// For running VMs, use QMP to create snapshot
				err = createQMPSnapshot(vm.QMPSocketPath, req.Name)
			} else {
				// For stopped VMs, use qemu-img
				err = createQemuImgSnapshot(vm.DiskPath, req.Name)
			}
		case "memory", "full":
			// Memory snapshots require QMP and a running VM
			if vm.Status != "running" {
				db2.Exec("UPDATE vm_snapshots SET status = 'failed' WHERE id = ?", snapshotID)
				return
			}
			err = createQMPSnapshot(vm.QMPSocketPath, req.Name)
		}

		if err != nil {
			db2.Exec("UPDATE vm_snapshots SET status = 'failed' WHERE id = ?", snapshotID)
			return
		}

		// Get snapshot size
		size = getSnapshotSize(vm.DiskPath, req.Name)

		db2.Exec("UPDATE vm_snapshots SET status = 'completed', size_bytes = ?, completed_at = NOW() WHERE id = ?", size, snapshotID)
	}()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"snapshot_id": snapshotID,
	})
}

// RestoreVMSnapshotHandler restores a VM to a snapshot
func RestoreVMSnapshotHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vmID, _ := strconv.Atoi(vars["id"])
	snapshotID, _ := strconv.Atoi(vars["snapshotId"])

	db, err := NewDatabase()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Get VM info
	vm, err := scanVM(db.QueryRow("SELECT "+vmFields+" FROM virtual_machines WHERE id = ?", vmID))
	if err != nil {
		http.Error(w, "VM not found", http.StatusNotFound)
		return
	}

	// Get snapshot info
	var snapshot VMSnapshot
	err = db.QueryRow(`SELECT id, vm_id, name, description, snapshot_type, parent_id,
		size_bytes, status, created_by, created_at, completed_at
		FROM vm_snapshots WHERE id = ? AND vm_id = ?`, snapshotID, vmID).Scan(
		&snapshot.ID, &snapshot.VMID, &snapshot.Name, &snapshot.Description,
		&snapshot.SnapshotType, &snapshot.ParentID, &snapshot.SizeBytes,
		&snapshot.Status, &snapshot.CreatedBy, &snapshot.CreatedAt, &snapshot.CompletedAt)
	if err != nil {
		http.Error(w, "Snapshot not found", http.StatusNotFound)
		return
	}

	// Check if VM is running
	if vm.Status == "running" {
		// Try to restore via QMP if possible
		if snapshot.SnapshotType == "memory" || snapshot.SnapshotType == "full" {
			err = restoreQMPSnapshot(vm.QMPSocketPath, snapshot.Name)
			if err != nil {
				http.Error(w, "Failed to restore snapshot: "+err.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			http.Error(w, "Cannot restore disk-only snapshot while VM is running. Stop the VM first.", http.StatusBadRequest)
			return
		}
	} else {
		// VM is stopped, use qemu-img
		err = restoreQemuImgSnapshot(vm.DiskPath, snapshot.Name)
		if err != nil {
			http.Error(w, "Failed to restore snapshot: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// DeleteVMSnapshotHandler deletes a snapshot
func DeleteVMSnapshotHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vmID, _ := strconv.Atoi(vars["id"])
	snapshotID, _ := strconv.Atoi(vars["snapshotId"])

	db, err := NewDatabase()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Get VM info
	vm, err := scanVM(db.QueryRow("SELECT "+vmFields+" FROM virtual_machines WHERE id = ?", vmID))
	if err != nil {
		http.Error(w, "VM not found", http.StatusNotFound)
		return
	}

	// Get snapshot info
	var snapshotName string
	err = db.QueryRow("SELECT name FROM vm_snapshots WHERE id = ? AND vm_id = ?", snapshotID, vmID).Scan(&snapshotName)
	if err != nil {
		http.Error(w, "Snapshot not found", http.StatusNotFound)
		return
	}

	// Delete snapshot from qcow2 image
	if vm.Status == "running" {
		// Use QMP
		deleteQMPSnapshot(vm.QMPSocketPath, snapshotName)
	} else {
		// Use qemu-img
		deleteQemuImgSnapshot(vm.DiskPath, snapshotName)
	}

	// Delete from database
	db.Exec("DELETE FROM vm_snapshots WHERE id = ?", snapshotID)

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// Helper functions for snapshot management

func listQemuSnapshots(diskPath string) []map[string]interface{} {
	if diskPath == "" {
		return nil
	}

	cmd := exec.Command("qemu-img", "snapshot", "-l", diskPath)
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	var snapshots []map[string]interface{}
	lines := strings.Split(string(output), "\n")
	for i, line := range lines {
		if i < 2 || line == "" { // Skip header lines
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 4 {
			snapshots = append(snapshots, map[string]interface{}{
				"id":        fields[0],
				"tag":       fields[1],
				"vm_size":   fields[2],
				"date":      fields[3],
				"vm_clock":  "",
			})
		}
	}

	return snapshots
}

func createQemuImgSnapshot(diskPath, name string) error {
	cmd := exec.Command("qemu-img", "snapshot", "-c", name, diskPath)
	return cmd.Run()
}

func restoreQemuImgSnapshot(diskPath, name string) error {
	cmd := exec.Command("qemu-img", "snapshot", "-a", name, diskPath)
	return cmd.Run()
}

func deleteQemuImgSnapshot(diskPath, name string) error {
	cmd := exec.Command("qemu-img", "snapshot", "-d", name, diskPath)
	return cmd.Run()
}

func getSnapshotSize(diskPath, name string) int64 {
	// This is an approximation - qemu-img doesn't easily report individual snapshot sizes
	// We could parse qemu-img info output for more accuracy
	info, err := os.Stat(diskPath)
	if err != nil {
		return 0
	}
	return info.Size()
}

// QMP commands for live snapshot management

func createQMPSnapshot(socketPath, name string) error {
	if socketPath == "" {
		return fmt.Errorf("QMP socket not configured")
	}

	// Use QMP to create a snapshot
	qmpCmd := fmt.Sprintf(`{"execute": "snapshot-save", "arguments": {"job-id": "snap-%s", "tag": "%s", "vmstate": "vmstate0", "devices": ["drive0"]}}`, name, name)
	return sendQMPCommand(socketPath, qmpCmd)
}

func restoreQMPSnapshot(socketPath, name string) error {
	if socketPath == "" {
		return fmt.Errorf("QMP socket not configured")
	}

	qmpCmd := fmt.Sprintf(`{"execute": "snapshot-load", "arguments": {"job-id": "load-%s", "tag": "%s", "vmstate": "vmstate0", "devices": ["drive0"]}}`, name, name)
	return sendQMPCommand(socketPath, qmpCmd)
}

func deleteQMPSnapshot(socketPath, name string) error {
	if socketPath == "" {
		return fmt.Errorf("QMP socket not configured")
	}

	qmpCmd := fmt.Sprintf(`{"execute": "snapshot-delete", "arguments": {"job-id": "del-%s", "tag": "%s", "devices": ["drive0"]}}`, name, name)
	return sendQMPCommand(socketPath, qmpCmd)
}

func sendQMPCommand(socketPath, command string) error {
	// Use socat or nc to send QMP command
	// First, send qmp_capabilities to enable command mode
	initCmd := `{"execute": "qmp_capabilities"}`

	cmd := exec.Command("bash", "-c", fmt.Sprintf(
		`echo -e '%s\n%s' | socat - UNIX-CONNECT:%s`,
		initCmd, command, socketPath,
	))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("QMP command failed: %v - %s", err, string(output))
	}

	// Check if response contains error
	if strings.Contains(string(output), `"error"`) {
		return fmt.Errorf("QMP error: %s", string(output))
	}

	return nil
}
