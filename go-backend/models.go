package main

import (
	"time"
)

type User struct {
	ID        int       `json:"id" db:"id"`
	Username  string    `json:"username" db:"username"`
	Email     string    `json:"email" db:"email"`
	Password  string    `json:"-" db:"password"`
	FullName  string    `json:"full_name" db:"full_name"`
	Role      string    `json:"role" db:"role"`
	IsActive  bool      `json:"is_active" db:"is_active"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
	LastLogin *time.Time `json:"last_login" db:"last_login"`
}

type Share struct {
	ID               int       `json:"id" db:"id"`
	ShareName        string    `json:"share_name" db:"share_name"`
	DisplayName      string    `json:"display_name" db:"display_name"`
	Path             string    `json:"path" db:"path"`
	Comment          string    `json:"comment" db:"comment"`
	Browseable       bool      `json:"browseable" db:"browseable"`
	Readonly         bool      `json:"readonly" db:"readonly"`
	GuestOk          bool      `json:"guest_ok" db:"guest_ok"`
	CaseSensitive    string    `json:"case_sensitive" db:"case_sensitive"`
	PreserveCase     bool      `json:"preserve_case" db:"preserve_case"`
	ShortPreserveCase bool     `json:"short_preserve_case" db:"short_preserve_case"`
	ValidUsers       string    `json:"valid_users" db:"valid_users"`
	WriteList        string    `json:"write_list" db:"write_list"`
	ReadList         string    `json:"read_list" db:"read_list"`
	AdminUsers       string    `json:"admin_users" db:"admin_users"`
	CreateMask       string    `json:"create_mask" db:"create_mask"`
	DirectoryMask    string    `json:"directory_mask" db:"directory_mask"`
	ForceUser        string    `json:"force_user" db:"force_user"`
	ForceGroup       string    `json:"force_group" db:"force_group"`
	IsActive         bool      `json:"is_active" db:"is_active"`
	CreatedBy        *int       `json:"created_by" db:"created_by"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

type ShareUser struct {
	ID          int       `json:"id" db:"id"`
	Username    string    `json:"username" db:"username"`
	FullName    string    `json:"full_name" db:"full_name"`
	PasswordHash string   `json:"-" db:"password_hash"`
	IsActive    bool      `json:"is_active" db:"is_active"`
	CreatedBy   *int       `json:"created_by" db:"created_by"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type SharePermission struct {
	ID              int       `json:"id" db:"id"`
	ShareID         int       `json:"share_id" db:"share_id"`
	ShareUserID     int       `json:"share_user_id" db:"share_user_id"`
	PermissionLevel string    `json:"permission_level" db:"permission_level"`
	CreatedBy       *int       `json:"created_by" db:"created_by"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

type VirtualMachine struct {
	ID                 int       `json:"id" db:"id"`
	Name               string    `json:"name" db:"name"`
	Description        string    `json:"description" db:"description"`
	UUID               string    `json:"uuid" db:"uuid"`
	CPUCores           int       `json:"cpu_cores" db:"cpu_cores"`
	RAMMB              int       `json:"ram_mb" db:"ram_mb"`
	DiskPath           string    `json:"disk_path" db:"disk_path"`
	DiskSizeGB         int       `json:"disk_size_gb" db:"disk_size_gb"`
	DiskFormat         string    `json:"disk_format" db:"disk_format"`
	BootOrder          string    `json:"boot_order" db:"boot_order"`
	ISOPath            string    `json:"iso_path" db:"iso_path"`
	BootFromDisk       bool      `json:"boot_from_disk" db:"boot_from_disk"`
	PhysicalDiskDevice string    `json:"physical_disk_device" db:"physical_disk_device"`
	NetworkMode        string    `json:"network_mode" db:"network_mode"`
	NetworkBridge      string    `json:"network_bridge" db:"network_bridge"`
	MACAddress         string    `json:"mac_address" db:"mac_address"`
	DisplayType        string    `json:"display_type" db:"display_type"`
	SpicePort          int       `json:"spice_port" db:"spice_port"`
	VNCPort            int       `json:"vnc_port" db:"vnc_port"`
	SpicePassword      string    `json:"spice_password" db:"spice_password"`
	Status             string    `json:"status" db:"status"`
	PID                *int       `json:"pid" db:"pid"`
	CreatedBy          *int       `json:"created_by" db:"created_by"`
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time `json:"updated_at" db:"updated_at"`
	LastStartedAt      *time.Time `json:"last_started_at" db:"last_started_at"`
}

type VMBackup struct {
	ID             int       `json:"id" db:"id"`
	VMID           int       `json:"vm_id" db:"vm_id"`
	VMName         string    `json:"vm_name" db:"vm_name"`
	BackupName     string    `json:"backup_name" db:"backup_name"`
	BackupPath     string    `json:"backup_path" db:"backup_path"`
	BackupSize     *int64    `json:"backup_size" db:"backup_size"`
	Compressed     bool      `json:"compressed" db:"compressed"`
	CompressionType string   `json:"compression_type" db:"compression_type"`
	Status         string    `json:"status" db:"status"`
	CreatedBy      *int       `json:"created_by" db:"created_by"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	CompletedAt    *time.Time `json:"completed_at" db:"completed_at"`
	Notes           string    `json:"notes" db:"notes"`
}

type ActivityLog struct {
	ID          int       `json:"id" db:"id"`
	UserID      *int       `json:"user_id" db:"user_id"`
	Action      string    `json:"action" db:"action"`
	Description string    `json:"description" db:"description"`
	IPAddress   string    `json:"ip_address" db:"ip_address"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	Username    string    `json:"username,omitempty"`
}

type SystemLog struct {
	ID        int       `json:"id" db:"id"`
	Level     string    `json:"level" db:"level"`
	Message   string    `json:"message" db:"message"`
	Context   string    `json:"context" db:"context"`
	UserID    *int       `json:"user_id" db:"user_id"`
	IPAddress string    `json:"ip_address" db:"ip_address"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	Username  string    `json:"username,omitempty"`
}

