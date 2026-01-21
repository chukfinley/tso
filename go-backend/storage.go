package main

import (
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
)

type DiskInfo struct {
	Name       string `json:"name"`
	Model      string `json:"model"`
	Size       int64  `json:"size"`
	SizeFormat string `json:"size_formatted"`
	Type       string `json:"type"` // ssd, hdd, nvme
	Serial     string `json:"serial,omitempty"`
	Vendor     string `json:"vendor,omitempty"`
}

type PartitionInfo struct {
	Device       string  `json:"device"`
	MountPoint   string  `json:"mount_point"`
	Filesystem   string  `json:"filesystem"`
	Total        int64   `json:"total"`
	Used         int64   `json:"used"`
	Available    int64   `json:"available"`
	UsagePercent float64 `json:"usage_percent"`
	TotalFmt     string  `json:"total_formatted"`
	UsedFmt      string  `json:"used_formatted"`
	AvailableFmt string  `json:"available_formatted"`
}

func GetStorageDisksHandler(w http.ResponseWriter, r *http.Request) {
	disks := getPhysicalDisks()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"disks":   disks,
	})
}

func GetStoragePartitionsHandler(w http.ResponseWriter, r *http.Request) {
	partitions := getMountedPartitions()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"success":    true,
		"partitions": partitions,
	})
}

func getPhysicalDisks() []DiskInfo {
	var disks []DiskInfo

	// Get list of block devices using lsblk
	cmd := exec.Command("lsblk", "-d", "-n", "-o", "NAME,SIZE,TYPE,MODEL,SERIAL,VENDOR", "-b")
	output, err := cmd.Output()
	if err != nil {
		// Fallback to reading from /sys/block
		return getDisksFromSysBlock()
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		name := fields[0]
		// Skip loop devices, ram disks, etc.
		if strings.HasPrefix(name, "loop") || strings.HasPrefix(name, "ram") ||
			strings.HasPrefix(name, "dm-") || strings.HasPrefix(name, "sr") {
			continue
		}

		diskType := fields[2]
		if diskType != "disk" {
			continue
		}

		size, _ := strconv.ParseInt(fields[1], 10, 64)
		model := ""
		serial := ""
		vendor := ""

		if len(fields) > 3 {
			// Model might have spaces, join remaining fields before SERIAL
			remaining := fields[3:]
			if len(remaining) >= 1 {
				model = strings.TrimSpace(remaining[0])
			}
			if len(remaining) >= 2 {
				serial = strings.TrimSpace(remaining[1])
			}
			if len(remaining) >= 3 {
				vendor = strings.TrimSpace(remaining[2])
			}
		}

		// Determine disk type (ssd, hdd, nvme)
		diskKind := detectDiskType(name)

		disks = append(disks, DiskInfo{
			Name:       name,
			Model:      model,
			Size:       size,
			SizeFormat: formatBytes(size),
			Type:       diskKind,
			Serial:     serial,
			Vendor:     vendor,
		})
	}

	return disks
}

func getDisksFromSysBlock() []DiskInfo {
	var disks []DiskInfo

	entries, err := os.ReadDir("/sys/block")
	if err != nil {
		return disks
	}

	for _, entry := range entries {
		name := entry.Name()

		// Skip loop devices, ram disks, etc.
		if strings.HasPrefix(name, "loop") || strings.HasPrefix(name, "ram") ||
			strings.HasPrefix(name, "dm-") || strings.HasPrefix(name, "sr") {
			continue
		}

		devicePath := filepath.Join("/sys/block", name, "device")
		if _, err := os.Stat(devicePath); os.IsNotExist(err) {
			continue
		}

		// Read size
		sizeData, _ := os.ReadFile(filepath.Join("/sys/block", name, "size"))
		sectors, _ := strconv.ParseInt(strings.TrimSpace(string(sizeData)), 10, 64)
		size := sectors * 512

		// Read model
		modelData, _ := os.ReadFile(filepath.Join(devicePath, "model"))
		model := strings.TrimSpace(string(modelData))

		// Read vendor
		vendorData, _ := os.ReadFile(filepath.Join(devicePath, "vendor"))
		vendor := strings.TrimSpace(string(vendorData))

		diskType := detectDiskType(name)

		disks = append(disks, DiskInfo{
			Name:       name,
			Model:      model,
			Size:       size,
			SizeFormat: formatBytes(size),
			Type:       diskType,
			Vendor:     vendor,
		})
	}

	return disks
}

func detectDiskType(name string) string {
	// NVMe devices
	if strings.HasPrefix(name, "nvme") {
		return "nvme"
	}

	// Check rotational status
	rotPath := filepath.Join("/sys/block", name, "queue", "rotational")
	rotData, err := os.ReadFile(rotPath)
	if err == nil {
		rot := strings.TrimSpace(string(rotData))
		if rot == "0" {
			return "ssd"
		}
		return "hdd"
	}

	// Default to hdd for unknown
	return "hdd"
}

func getMountedPartitions() []PartitionInfo {
	var partitions []PartitionInfo

	// Parse /proc/mounts to get mounted filesystems
	mountData, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return partitions
	}

	// Keep track of seen mount points to avoid duplicates
	seenMounts := make(map[string]bool)

	lines := strings.Split(string(mountData), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		device := fields[0]
		mountPoint := fields[1]
		fsType := fields[2]

		// Filter out pseudo filesystems
		if !shouldIncludeMount(device, mountPoint, fsType) {
			continue
		}

		// Skip duplicate mount points
		if seenMounts[mountPoint] {
			continue
		}
		seenMounts[mountPoint] = true

		// Get disk usage using statfs
		var stat syscall.Statfs_t
		if err := syscall.Statfs(mountPoint, &stat); err != nil {
			continue
		}

		total := int64(stat.Blocks) * int64(stat.Bsize)
		available := int64(stat.Bavail) * int64(stat.Bsize)
		used := total - int64(stat.Bfree)*int64(stat.Bsize)

		usagePercent := 0.0
		if total > 0 {
			usagePercent = float64(used) / float64(total) * 100
		}

		// Decode escape sequences in mount point (like \040 for space)
		mountPoint = decodeMountPath(mountPoint)

		partitions = append(partitions, PartitionInfo{
			Device:       device,
			MountPoint:   mountPoint,
			Filesystem:   fsType,
			Total:        total,
			Used:         used,
			Available:    available,
			UsagePercent: usagePercent,
			TotalFmt:     formatBytes(total),
			UsedFmt:      formatBytes(used),
			AvailableFmt: formatBytes(available),
		})
	}

	return partitions
}

func shouldIncludeMount(device, mountPoint, fsType string) bool {
	// Include real block devices
	if strings.HasPrefix(device, "/dev/") {
		// Skip certain pseudo-devices
		if strings.HasPrefix(device, "/dev/loop") {
			return false
		}
		return true
	}

	// Include certain tmpfs mounts that users might care about
	if fsType == "tmpfs" {
		// Only include /tmp and /dev/shm for tmpfs
		if mountPoint == "/tmp" || mountPoint == "/dev/shm" {
			return true
		}
		return false
	}

	// Skip other virtual filesystems
	pseudoFS := map[string]bool{
		"proc": true, "sysfs": true, "devtmpfs": true, "devpts": true,
		"cgroup": true, "cgroup2": true, "securityfs": true, "debugfs": true,
		"tracefs": true, "fusectl": true, "configfs": true, "hugetlbfs": true,
		"mqueue": true, "bpf": true, "pstore": true, "efivarfs": true,
		"autofs": true, "rpc_pipefs": true, "overlay": true, "squashfs": true,
		"nsfs": true, "fuse.snapfuse": true,
	}

	return !pseudoFS[fsType]
}

func decodeMountPath(path string) string {
	// Decode octal escape sequences (e.g., \040 for space)
	re := regexp.MustCompile(`\\([0-7]{3})`)
	return re.ReplaceAllStringFunc(path, func(match string) string {
		code, _ := strconv.ParseInt(match[1:], 8, 32)
		return string(rune(code))
	})
}

// formatBytes is defined in system.go
