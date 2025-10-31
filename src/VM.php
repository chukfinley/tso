<?php
/**
 * Virtual Machine Management Class
 * Handles QEMU/KVM virtual machine operations
 */
class VM {
    private $db;
    private $vmDir = '/opt/serveros/vms';
    private $isoDir = '/opt/serveros/vms/iso';
    private $logDir = '/opt/serveros/logs/vms';

    public function __construct() {
        $this->db = Database::getInstance();

        // Ensure directories exist
        if (!is_dir($this->vmDir)) {
            mkdir($this->vmDir, 0755, true);
        }
        if (!is_dir($this->isoDir)) {
            mkdir($this->isoDir, 0755, true);
        }
        if (!is_dir($this->logDir)) {
            mkdir($this->logDir, 0755, true);
        }
    }

    /**
     * Get all VMs
     */
    public function getAll() {
        return $this->db->fetchAll("SELECT * FROM virtual_machines ORDER BY name");
    }

    /**
     * Get VM by ID
     */
    public function getById($id) {
        return $this->db->fetchOne("SELECT * FROM virtual_machines WHERE id = ?", [$id]);
    }

    /**
     * Get VM by UUID
     */
    public function getByUuid($uuid) {
        return $this->db->fetchOne("SELECT * FROM virtual_machines WHERE uuid = ?", [$uuid]);
    }

    /**
     * Create new VM
     */
    public function create($data) {
        $uuid = $this->generateUuid();
        $name = $data['name'];

        // Generate MAC address if not provided
        $macAddress = $data['mac_address'] ?? $this->generateMacAddress();

        // Set disk path
        $diskPath = $this->vmDir . '/' . $name . '.qcow2';

        // Create disk image
        if (!empty($data['disk_size_gb'])) {
            $this->createDiskImage($diskPath, $data['disk_size_gb'], $data['disk_format'] ?? 'qcow2');
        }

        // Allocate SPICE port
        $spicePort = $this->allocatePort(5900, 6000);

        $sql = "INSERT INTO virtual_machines (
            name, description, uuid, cpu_cores, ram_mb,
            disk_path, disk_size_gb, disk_format,
            boot_order, iso_path, boot_from_disk, physical_disk_device,
            network_mode, network_bridge, mac_address,
            display_type, spice_port, spice_password,
            created_by
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)";

        $params = [
            $name,
            $data['description'] ?? '',
            $uuid,
            $data['cpu_cores'] ?? 2,
            $data['ram_mb'] ?? 2048,
            $diskPath,
            $data['disk_size_gb'] ?? 20,
            $data['disk_format'] ?? 'qcow2',
            $data['boot_order'] ?? 'cd,hd',
            $data['iso_path'] ?? null,
            $data['boot_from_disk'] ?? false,
            $data['physical_disk_device'] ?? null,
            $data['network_mode'] ?? 'nat',
            $data['network_bridge'] ?? null,
            $macAddress,
            $data['display_type'] ?? 'spice',
            $spicePort,
            $this->generatePassword(),
            $_SESSION['user_id'] ?? null
        ];

        $this->db->query($sql, $params);
        return $this->db->lastInsertId();
    }

    /**
     * Update VM configuration
     */
    public function update($id, $data) {
        $vm = $this->getById($id);
        if (!$vm) {
            return false;
        }

        // Don't allow updates while VM is running
        if ($vm['status'] === 'running') {
            throw new Exception("Cannot update VM while it is running");
        }

        $updates = [];
        $params = [];

        $allowedFields = [
            'name', 'description', 'cpu_cores', 'ram_mb',
            'boot_order', 'iso_path', 'boot_from_disk', 'physical_disk_device',
            'network_mode', 'network_bridge', 'display_type'
        ];

        foreach ($allowedFields as $field) {
            if (isset($data[$field])) {
                $updates[] = "$field = ?";
                $params[] = $data[$field];
            }
        }

        if (empty($updates)) {
            return true;
        }

        $params[] = $id;
        $sql = "UPDATE virtual_machines SET " . implode(', ', $updates) . " WHERE id = ?";
        $this->db->query($sql, $params);

        return true;
    }

    /**
     * Delete VM
     */
    public function delete($id) {
        $vm = $this->getById($id);
        if (!$vm) {
            return false;
        }

        // Stop VM if running
        if ($vm['status'] === 'running') {
            $this->stop($id, true);
        }

        // Delete disk image
        if (!empty($vm['disk_path']) && file_exists($vm['disk_path'])) {
            unlink($vm['disk_path']);
        }

        // Delete from database
        $this->db->query("DELETE FROM virtual_machines WHERE id = ?", [$id]);

        return true;
    }

    /**
     * Start VM
     */
    public function start($id) {
        $vm = $this->getById($id);
        if (!$vm) {
            throw new Exception("VM not found");
        }

        if ($vm['status'] === 'running') {
            throw new Exception("VM is already running");
        }

        // Build QEMU command
        $cmd = $this->buildQemuCommand($vm);

        // Log file
        $logFile = $this->logDir . '/' . $vm['name'] . '.log';

        // Start VM in background
        $fullCmd = sprintf(
            'nohup %s > %s 2>&1 & echo $!',
            $cmd,
            escapeshellarg($logFile)
        );

        $pid = trim(shell_exec($fullCmd));

        if (empty($pid) || !is_numeric($pid)) {
            throw new Exception("Failed to start VM");
        }

        // Update VM status
        $this->db->query(
            "UPDATE virtual_machines SET status = 'running', pid = ?, last_started_at = NOW() WHERE id = ?",
            [$pid, $id]
        );

        return true;
    }

    /**
     * Stop VM
     */
    public function stop($id, $force = false) {
        $vm = $this->getById($id);
        if (!$vm) {
            throw new Exception("VM not found");
        }

        if ($vm['status'] !== 'running' || empty($vm['pid'])) {
            throw new Exception("VM is not running");
        }

        // Stop VM process
        $signal = $force ? 'SIGKILL' : 'SIGTERM';
        shell_exec("kill -s $signal " . (int)$vm['pid'] . " 2>/dev/null");

        // Wait a moment for graceful shutdown
        if (!$force) {
            sleep(2);
        }

        // Update VM status
        $this->db->query(
            "UPDATE virtual_machines SET status = 'stopped', pid = NULL WHERE id = ?",
            [$id]
        );

        return true;
    }

    /**
     * Restart VM
     */
    public function restart($id) {
        $this->stop($id, false);
        sleep(1);
        return $this->start($id);
    }

    /**
     * Get VM status (check if process is actually running)
     */
    public function getStatus($id) {
        $vm = $this->getById($id);
        if (!$vm) {
            return null;
        }

        // Check if PID is still running
        if ($vm['status'] === 'running' && !empty($vm['pid'])) {
            $running = shell_exec("ps -p " . (int)$vm['pid'] . " > /dev/null 2>&1 && echo 'yes' || echo 'no'");
            if (trim($running) === 'no') {
                // Process died, update status
                $this->db->query("UPDATE virtual_machines SET status = 'stopped', pid = NULL WHERE id = ?", [$id]);
                return 'stopped';
            }
        }

        return $vm['status'];
    }

    /**
     * Get VM logs
     */
    public function getLogs($id, $lines = 100) {
        $vm = $this->getById($id);
        if (!$vm) {
            return null;
        }

        $logFile = $this->logDir . '/' . $vm['name'] . '.log';

        if (!file_exists($logFile)) {
            return '';
        }

        return shell_exec("tail -n " . (int)$lines . " " . escapeshellarg($logFile));
    }

    /**
     * Generate SPICE connection file
     */
    public function generateSpiceFile($id) {
        $vm = $this->getById($id);
        if (!$vm) {
            return null;
        }

        if ($vm['display_type'] !== 'spice') {
            throw new Exception("VM does not use SPICE display");
        }

        $content = "[virt-viewer]\n";
        $content .= "type=spice\n";
        $content .= "host=localhost\n";
        $content .= "port=" . $vm['spice_port'] . "\n";
        $content .= "password=" . $vm['spice_password'] . "\n";
        $content .= "title=" . $vm['name'] . "\n";
        $content .= "delete-this-file=1\n";
        $content .= "fullscreen=0\n";

        return $content;
    }

    /**
     * List available ISO files
     */
    public function listIsos() {
        $isos = [];
        $files = glob($this->isoDir . '/*.iso');

        foreach ($files as $file) {
            $isos[] = [
                'name' => basename($file),
                'path' => $file,
                'size' => filesize($file),
                'size_formatted' => $this->formatBytes(filesize($file))
            ];
        }

        return $isos;
    }

    /**
     * List available physical disks
     */
    public function listPhysicalDisks() {
        $disks = [];
        $output = shell_exec("lsblk -ndo NAME,SIZE,TYPE,MODEL 2>/dev/null | grep disk");

        if (!empty($output)) {
            $lines = explode("\n", trim($output));
            foreach ($lines as $line) {
                if (empty($line)) continue;

                $parts = preg_split('/\s+/', $line, 4);
                $disks[] = [
                    'device' => '/dev/' . $parts[0],
                    'size' => $parts[1] ?? 'Unknown',
                    'model' => $parts[3] ?? 'Unknown'
                ];
            }
        }

        return $disks;
    }

    /**
     * Get available network bridges
     */
    public function listNetworkBridges() {
        $bridges = [];
        $output = shell_exec("ip link show type bridge 2>/dev/null | grep '^[0-9]' | awk '{print $2}' | sed 's/:$//'");

        if (!empty($output)) {
            $bridges = array_filter(explode("\n", trim($output)));
        }

        // Add default bridge
        if (!in_array('virbr0', $bridges)) {
            $bridges[] = 'virbr0';
        }

        return $bridges;
    }

    // ============ Private Helper Methods ============

    /**
     * Build QEMU command line
     */
    private function buildQemuCommand($vm) {
        $cmd = ['qemu-system-x86_64'];

        // Enable KVM if available
        $cmd[] = '-enable-kvm';

        // CPU and RAM
        $cmd[] = '-cpu host';
        $cmd[] = '-smp cores=' . $vm['cpu_cores'];
        $cmd[] = '-m ' . $vm['ram_mb'];

        // Machine type
        $cmd[] = '-machine type=q35,accel=kvm';

        // UUID
        $cmd[] = '-uuid ' . $vm['uuid'];
        $cmd[] = '-name ' . escapeshellarg($vm['name']);

        // Disk
        if (!empty($vm['disk_path']) && file_exists($vm['disk_path'])) {
            $cmd[] = '-drive file=' . escapeshellarg($vm['disk_path']) . ',if=virtio,format=' . $vm['disk_format'];
        }

        // Physical disk
        if (!empty($vm['physical_disk_device']) && $vm['boot_from_disk']) {
            $cmd[] = '-drive file=' . $vm['physical_disk_device'] . ',format=raw,if=virtio';
        }

        // ISO/CDROM
        if (!empty($vm['iso_path']) && file_exists($vm['iso_path'])) {
            $cmd[] = '-cdrom ' . escapeshellarg($vm['iso_path']);
        }

        // Boot order
        $cmd[] = '-boot order=' . $vm['boot_order'];

        // Network
        $cmd[] = $this->buildNetworkConfig($vm);

        // Display
        if ($vm['display_type'] === 'spice') {
            $cmd[] = '-spice port=' . $vm['spice_port'] . ',addr=0.0.0.0,disable-ticketing';
            $cmd[] = '-vga qxl';
            $cmd[] = '-device virtio-serial';
            $cmd[] = '-chardev spicevmc,id=vdagent,name=vdagent';
            $cmd[] = '-device virtserialport,chardev=vdagent,name=com.redhat.spice.0';
        } elseif ($vm['display_type'] === 'vnc') {
            $cmd[] = '-vnc :' . ($vm['vnc_port'] - 5900);
        } else {
            $cmd[] = '-display none';
        }

        // Other options
        $cmd[] = '-daemonize';
        $cmd[] = '-pidfile /var/run/vm-' . $vm['name'] . '.pid';

        return implode(' ', $cmd);
    }

    /**
     * Build network configuration
     */
    private function buildNetworkConfig($vm) {
        $config = [];

        if ($vm['network_mode'] === 'nat') {
            $config[] = '-netdev user,id=net0';
            $config[] = '-device virtio-net-pci,netdev=net0,mac=' . $vm['mac_address'];
        } elseif ($vm['network_mode'] === 'bridge' && !empty($vm['network_bridge'])) {
            $config[] = '-netdev bridge,id=net0,br=' . $vm['network_bridge'];
            $config[] = '-device virtio-net-pci,netdev=net0,mac=' . $vm['mac_address'];
        } elseif ($vm['network_mode'] === 'user') {
            $config[] = '-net user';
            $config[] = '-net nic,model=virtio,macaddr=' . $vm['mac_address'];
        }

        return implode(' ', $config);
    }

    /**
     * Create disk image
     */
    private function createDiskImage($path, $sizeGb, $format = 'qcow2') {
        $cmd = sprintf(
            'qemu-img create -f %s %s %dG',
            escapeshellarg($format),
            escapeshellarg($path),
            (int)$sizeGb
        );

        exec($cmd, $output, $returnCode);

        if ($returnCode !== 0) {
            throw new Exception("Failed to create disk image");
        }

        return true;
    }

    /**
     * Generate UUID
     */
    private function generateUuid() {
        return sprintf(
            '%04x%04x-%04x-%04x-%04x-%04x%04x%04x',
            mt_rand(0, 0xffff), mt_rand(0, 0xffff),
            mt_rand(0, 0xffff),
            mt_rand(0, 0x0fff) | 0x4000,
            mt_rand(0, 0x3fff) | 0x8000,
            mt_rand(0, 0xffff), mt_rand(0, 0xffff), mt_rand(0, 0xffff)
        );
    }

    /**
     * Generate MAC address
     */
    private function generateMacAddress() {
        return sprintf(
            '52:54:00:%02x:%02x:%02x',
            mt_rand(0, 0xff),
            mt_rand(0, 0xff),
            mt_rand(0, 0xff)
        );
    }

    /**
     * Generate random password
     */
    private function generatePassword($length = 12) {
        $chars = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789';
        $password = '';
        for ($i = 0; $i < $length; $i++) {
            $password .= $chars[mt_rand(0, strlen($chars) - 1)];
        }
        return $password;
    }

    /**
     * Allocate available port
     */
    private function allocatePort($min, $max) {
        $usedPorts = $this->db->fetchAll(
            "SELECT spice_port FROM virtual_machines WHERE spice_port IS NOT NULL"
        );

        $used = array_column($usedPorts, 'spice_port');

        for ($port = $min; $port <= $max; $port++) {
            if (!in_array($port, $used)) {
                return $port;
            }
        }

        throw new Exception("No available ports");
    }

    /**
     * Format bytes to human readable
     */
    private function formatBytes($bytes, $precision = 2) {
        $units = ['B', 'KB', 'MB', 'GB', 'TB'];
        $bytes = max($bytes, 0);
        $pow = floor(($bytes ? log($bytes) : 0) / log(1024));
        $pow = min($pow, count($units) - 1);
        $bytes /= pow(1024, $pow);
        return round($bytes, $precision) . ' ' . $units[$pow];
    }
}
