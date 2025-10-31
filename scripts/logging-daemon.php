#!/usr/bin/env php
<?php
/**
 * TSO Background Logging Daemon
 * Runs continuously 24/7 to log data from pages, services, commands, PHP, and updates
 */

// Prevent running multiple instances
$lockFile = '/tmp/tso-logging-daemon.lock';
$lockFp = fopen($lockFile, 'w');
if (!flock($lockFp, LOCK_EX | LOCK_NB)) {
    error_log("Logging daemon is already running (lock file: $lockFile)");
    exit(1);
}

// Register shutdown function to remove lock
register_shutdown_function(function() use ($lockFile, $lockFp) {
    flock($lockFp, LOCK_UN);
    fclose($lockFp);
    @unlink($lockFile);
});

// Set up signal handlers for graceful shutdown (if pcntl available)
if (function_exists('pcntl_signal')) {
    pcntl_signal(SIGTERM, function() {
        global $running;
        $running = false;
    });
    pcntl_signal(SIGINT, function() {
        global $running;
        $running = false;
    });
}

// Determine installation directory
$installDir = getenv('TSO_INSTALL_DIR') ?: '/opt/serveros';
if (!is_dir($installDir)) {
    // Try to detect from script location
    $scriptDir = dirname(__DIR__);
    if (is_file("$scriptDir/config/config.php")) {
        $installDir = $scriptDir;
    }
}

if (!is_file("$installDir/config/config.php")) {
    error_log("ERROR: Cannot find config.php at $installDir/config/config.php");
    exit(1);
}

// Load configuration
require_once "$installDir/config/config.php";
require_once "$installDir/src/Database.php";
require_once "$installDir/src/Logger.php";

// Initialize logger
$logger = Logger::getInstance();
$db = Database::getInstance();

// Log daemon start
$logger->info("Logging daemon started", [
    'daemon' => 'tso-logging',
    'pid' => getmypid(),
    'install_dir' => $installDir
]);

// Configuration
$logDir = "$installDir/logs";
$monitorInterval = 5; // seconds
$commandHistoryFile = "$logDir/command-history.json";
$serviceStateFile = "$logDir/service-states.json";
$lastUpdateCheckFile = "$logDir/last-update-check.txt";

// Ensure log directory exists
if (!is_dir($logDir)) {
    mkdir($logDir, 0755, true);
}

// Initialize state files
if (!file_exists($commandHistoryFile)) {
    file_put_contents($commandHistoryFile, json_encode([]));
}
if (!file_exists($serviceStateFile)) {
    file_put_contents($serviceStateFile, json_encode([]));
}

// Main daemon loop
$running = true;
$iteration = 0;

while ($running) {
    $iteration++;
    
    // Process pending signals (if pcntl available)
    if (function_exists('pcntl_signal_dispatch')) {
        pcntl_signal_dispatch();
    }
    
    if (!$running) {
        break;
    }
    
    try {
        // 1. Monitor system services
        monitorServices($logger, $serviceStateFile);
        
        // 2. Monitor command executions (via history and process monitoring)
        monitorCommands($logger, $commandHistoryFile);
        
        // 3. Monitor system updates
        monitorUpdates($logger, $lastUpdateCheckFile);
        
        // 4. Monitor PHP error logs
        monitorPhpErrors($logger, $installDir);
        
        // 5. Monitor Apache/Apache2 access logs
        monitorApacheLogs($logger);
        
        // 6. Monitor system logs (syslog, auth.log)
        monitorSystemLogs($logger);
        
        // 7. Clean old logs periodically (every hour)
        if ($iteration % 720 === 0) { // 720 * 5 seconds = 1 hour
            cleanupOldLogs($logger);
        }
        
        // 8. Health check (log every 10 minutes)
        if ($iteration % 120 === 0) {
            $logger->debug("Logging daemon health check", [
                'iteration' => $iteration,
                'uptime' => $iteration * $monitorInterval,
                'memory_usage' => memory_get_usage(true),
                'memory_peak' => memory_get_peak_usage(true)
            ]);
        }
        
    } catch (Exception $e) {
        error_log("ERROR in logging daemon: " . $e->getMessage());
        // Try to log to database/file
        try {
            $logger->error("Logging daemon error", [
                'exception' => get_class($e),
                'message' => $e->getMessage(),
                'file' => $e->getFile(),
                'line' => $e->getLine()
            ]);
        } catch (Exception $e2) {
            error_log("CRITICAL: Cannot log to database: " . $e2->getMessage());
        }
    }
    
    // Sleep before next iteration
    sleep($monitorInterval);
}

// Log daemon shutdown
try {
    $logger->info("Logging daemon stopped", [
        'daemon' => 'tso-logging',
        'pid' => getmypid(),
        'iterations' => $iteration
    ]);
} catch (Exception $e) {
    error_log("Cannot log daemon stop: " . $e->getMessage());
}

exit(0);

/**
 * Monitor system services
 */
function monitorServices($logger, $stateFile) {
    $services = [
        'apache2', 'mariadb', 'smbd', 'nmbd', 'libvirtd'
    ];
    
    $previousStates = [];
    if (file_exists($stateFile)) {
        $previousStates = json_decode(file_get_contents($stateFile), true) ?: [];
    }
    
    $currentStates = [];
    
    foreach ($services as $service) {
        $isActive = false;
        $status = 'unknown';
        
        // Check service status
        $output = [];
        $returnCode = 0;
        @exec("systemctl is-active $service 2>/dev/null", $output, $returnCode);
        $isActive = ($returnCode === 0);
        
        // Get more detailed status
        @exec("systemctl is-enabled $service 2>/dev/null", $output, $enabledReturn);
        $isEnabled = ($enabledReturn === 0);
        
        $currentStates[$service] = [
            'active' => $isActive,
            'enabled' => $isEnabled,
            'timestamp' => time()
        ];
        
        // Check if state changed
        if (isset($previousStates[$service])) {
            $prevState = $previousStates[$service];
            if ($prevState['active'] !== $isActive) {
                $action = $isActive ? 'started' : 'stopped';
                $logger->info("Service $action: $service", [
                    'service' => $service,
                    'action' => $action,
                    'enabled' => $isEnabled,
                    'previous_state' => $prevState['active'] ? 'active' : 'inactive'
                ]);
            }
        } else {
            // First time seeing this service
            $logger->debug("Monitoring service: $service", [
                'service' => $service,
                'active' => $isActive,
                'enabled' => $isEnabled
            ]);
        }
    }
    
    // Save current states
    file_put_contents($stateFile, json_encode($currentStates));
}

/**
 * Monitor command executions
 */
function monitorCommands($logger, $historyFile) {
    // Monitor bash history for www-data user (if accessible)
    $historyFiles = [
        '/root/.bash_history',
        '/home/www-data/.bash_history',
        '/var/www/.bash_history'
    ];
    
    $previousCommands = [];
    if (file_exists($historyFile)) {
        $previousCommands = json_decode(file_get_contents($historyFile), true) ?: [];
    }
    
    $newCommands = [];
    
    foreach ($historyFiles as $histFile) {
        if (!is_readable($histFile)) {
            continue;
        }
        
        $lines = file($histFile);
        if (!$lines) {
            continue;
        }
        
        // Get recent commands (last 10)
        $recentLines = array_slice($lines, -10);
        
        foreach ($recentLines as $line) {
            $line = trim($line);
            if (empty($line) || $line[0] === '#') {
                continue;
            }
            
            $hash = md5($line);
            if (!in_array($hash, $previousCommands)) {
                $newCommands[] = $hash;
                
                // Log significant commands
                if (preg_match('/\b(sudo|systemctl|apt|yum|service|kill|reboot|shutdown)\b/i', $line)) {
                    $logger->info("Command executed", [
                        'command' => substr($line, 0, 200),
                        'source' => basename($histFile),
                        'type' => 'command_execution'
                    ]);
                }
            }
        }
    }
    
    // Update history file
    if (!empty($newCommands)) {
        $allCommands = array_merge($previousCommands, $newCommands);
        // Keep last 1000 command hashes
        $allCommands = array_slice($allCommands, -1000);
        file_put_contents($historyFile, json_encode($allCommands));
    }
}

/**
 * Monitor system updates
 */
function monitorUpdates($logger, $checkFile) {
    $lastCheck = 0;
    if (file_exists($checkFile)) {
        $lastCheck = (int)trim(file_get_contents($checkFile));
    }
    
    $currentTime = time();
    
    // Check every 6 hours
    if (($currentTime - $lastCheck) < 21600) {
        return;
    }
    
    file_put_contents($checkFile, $currentTime);
    
    // Check for package updates
    $output = [];
    @exec("apt list --upgradable 2>/dev/null | grep -v 'Listing...' | wc -l", $output, $returnCode);
    
    if ($returnCode === 0 && !empty($output)) {
        $updatesAvailable = (int)trim($output[0]);
        if ($updatesAvailable > 0) {
            $logger->info("System updates available", [
                'updates_count' => $updatesAvailable,
                'type' => 'system_update_check'
            ]);
        }
    }
    
    // Check TSO update script execution
    $updateLogs = glob('/opt/serveros/logs/update-*.log');
    if ($updateLogs) {
        foreach ($updateLogs as $updateLog) {
            $mtime = filemtime($updateLog);
            if ($mtime > $lastCheck) {
                $logger->info("TSO update executed", [
                    'log_file' => basename($updateLog),
                    'timestamp' => date('Y-m-d H:i:s', $mtime),
                    'type' => 'tso_update'
                ]);
            }
        }
    }
}

/**
 * Monitor PHP error logs
 */
function monitorPhpErrors($logger, $installDir) {
    $errorLog = ini_get('error_log');
    if (empty($errorLog)) {
        $errorLog = '/var/log/php*.log';
    }
    
    // Check Apache PHP error log
    $apacheErrorLog = '/var/log/apache2/error.log';
    if (is_readable($apacheErrorLog)) {
        monitorLogFile($logger, $apacheErrorLog, 'php_apache_errors', 100);
    }
    
    // Check application logs
    $appLogs = glob("$installDir/logs/app-*.log");
    foreach ($appLogs as $logFile) {
        monitorLogFile($logger, $logFile, 'php_app_logs', 50);
    }
}

/**
 * Monitor Apache access logs
 */
function monitorApacheLogs($logger) {
    $accessLogs = [
        '/var/log/apache2/access.log',
        '/var/log/apache2/serveros_access.log'
    ];
    
    foreach ($accessLogs as $logFile) {
        if (is_readable($logFile)) {
            monitorLogFile($logger, $logFile, 'apache_access', 100);
        }
    }
}

/**
 * Monitor system logs
 */
function monitorSystemLogs($logger) {
    $systemLogs = [
        '/var/log/auth.log' => 'auth',
        '/var/log/syslog' => 'syslog'
    ];
    
    foreach ($systemLogs as $logFile => $type) {
        if (is_readable($logFile)) {
            monitorLogFile($logger, $logFile, "system_$type", 50);
        }
    }
}

/**
 * Monitor a log file for new entries
 */
function monitorLogFile($logger, $logFile, $type, $lines = 100) {
    static $filePositions = [];
    
    $key = md5($logFile);
    $position = $filePositions[$key] ?? 0;
    
    if (!file_exists($logFile)) {
        return;
    }
    
    $currentSize = filesize($logFile);
    
    if ($currentSize <= $position) {
        return; // No new content
    }
    
    // Read new content
    $fp = fopen($logFile, 'r');
    if (!$fp) {
        return;
    }
    
    fseek($fp, $position);
    $newContent = fread($fp, $currentSize - $position);
    fclose($fp);
    
    $filePositions[$key] = $currentSize;
    
    if (empty($newContent)) {
        return;
    }
    
    // Process new lines
    $linesArray = explode("\n", trim($newContent));
    foreach ($linesArray as $line) {
        $line = trim($line);
        if (empty($line)) {
            continue;
        }
        
        // Log significant entries
        $shouldLog = false;
        $level = 'info';
        
        // Check for errors
        if (preg_match('/\b(error|ERROR|fatal|FATAL|exception|EXCEPTION|critical|CRITICAL)\b/i', $line)) {
            $shouldLog = true;
            $level = 'error';
        }
        // Check for warnings
        elseif (preg_match('/\b(warning|WARNING|warn|WARN)\b/i', $line)) {
            $shouldLog = true;
            $level = 'warning';
        }
        // Check for security events
        elseif (preg_match('/\b(failed|FAILED|denied|DENIED|unauthorized|UNAUTHORIZED|authentication|AUTH)\b/i', $line)) {
            $shouldLog = true;
            $level = 'warning';
        }
        
        if ($shouldLog) {
            $logger->log($level, "Log entry from $type", [
                'source' => basename($logFile),
                'type' => $type,
                'entry' => substr($line, 0, 500)
            ]);
        }
    }
}

/**
 * Clean up old logs
 */
function cleanupOldLogs($logger) {
    try {
        // Use logger's cleanup method
        $logger->clearOldLogs(30); // Keep last 30 days
        
        $logger->debug("Log cleanup completed", [
            'type' => 'log_cleanup',
            'retention_days' => 30
        ]);
    } catch (Exception $e) {
        error_log("Log cleanup error: " . $e->getMessage());
    }
}

