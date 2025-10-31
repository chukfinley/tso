/**
 * ServerOS - Main JavaScript
 */

// Auto-hide alerts after 5 seconds
document.addEventListener('DOMContentLoaded', function() {
    const alerts = document.querySelectorAll('.alert');
    alerts.forEach(alert => {
        setTimeout(() => {
            alert.style.opacity = '0';
            alert.style.transition = 'opacity 0.5s ease';
            setTimeout(() => alert.remove(), 500);
        }, 5000);
    });

    // Initialize system monitoring if on dashboard
    if (document.getElementById('cpu-model')) {
        initSystemMonitoring();
    }
});

// Confirm delete actions
function confirmDelete(message) {
    return confirm(message || 'Are you sure you want to delete this item?');
}

// Form validation
function validateForm(formId) {
    const form = document.getElementById(formId);
    if (!form) return true;

    const inputs = form.querySelectorAll('input[required], select[required], textarea[required]');
    let valid = true;

    inputs.forEach(input => {
        if (!input.value.trim()) {
            input.style.borderColor = '#f44336';
            valid = false;
        } else {
            input.style.borderColor = '#333';
        }
    });

    return valid;
}

// System Monitoring Functions
let lastNetworkStats = {};

function initSystemMonitoring() {
    // Initial fetch
    fetchSystemStats();
    
    // Update every 3 seconds
    setInterval(fetchSystemStats, 3000);
    
    // Update system time every second
    setInterval(updateSystemTime, 1000);
}

function fetchSystemStats() {
    fetch('/api/system-stats.php')
        .then(response => response.json())
        .then(data => {
            updateCpuInfo(data.cpu);
            updateMemoryInfo(data.memory);
            updateSwapInfo(data.swap);
            updateSystemInfo(data.uptime, data.motherboard);
            updateNetworkInfo(data.network);
        })
        .catch(error => {
            console.error('Error fetching system stats:', error);
        });
}

function updateCpuInfo(cpu) {
    document.getElementById('cpu-model').textContent = cpu.model;
    document.getElementById('cpu-cores').textContent = `${cpu.cores} Cores / ${cpu.architecture}`;
    document.getElementById('cpu-freq').textContent = cpu.frequency;
    document.getElementById('cpu-load').textContent = `${cpu.load_avg['1min']} / ${cpu.load_avg['5min']} / ${cpu.load_avg['15min']}`;
    
    // Update CPU usage with color coding
    const cpuUsageBar = document.getElementById('cpu-usage-bar');
    const cpuUsageText = document.getElementById('cpu-usage-text');
    cpuUsageBar.style.width = cpu.usage + '%';
    cpuUsageText.textContent = cpu.usage + '%';
    
    // Color coding
    cpuUsageBar.className = 'progress-bar';
    if (cpu.usage > 80) {
        cpuUsageBar.classList.add('danger');
    } else if (cpu.usage > 60) {
        cpuUsageBar.classList.add('warning');
    }
}

function updateMemoryInfo(memory) {
    document.getElementById('mem-total').textContent = memory.total_formatted;
    document.getElementById('mem-usage').textContent = `${memory.used_formatted} / ${memory.available_formatted}`;
    
    // Update memory usage with color coding
    const memUsageBar = document.getElementById('mem-usage-bar');
    const memUsageText = document.getElementById('mem-usage-text');
    memUsageBar.style.width = memory.usage_percent + '%';
    memUsageText.textContent = memory.usage_percent + '%';
    
    // Color coding
    memUsageBar.className = 'progress-bar';
    if (memory.usage_percent > 85) {
        memUsageBar.classList.add('danger');
    } else if (memory.usage_percent > 70) {
        memUsageBar.classList.add('warning');
    }
}

function updateSwapInfo(swap) {
    document.getElementById('swap-total').textContent = swap.total_formatted;
    document.getElementById('swap-usage').textContent = `${swap.used_formatted} / ${swap.free_formatted}`;
    
    // Update swap usage with color coding
    const swapUsageBar = document.getElementById('swap-usage-bar');
    const swapUsageText = document.getElementById('swap-usage-text');
    swapUsageBar.style.width = swap.usage_percent + '%';
    swapUsageText.textContent = swap.usage_percent + '%';
    
    // Color coding
    swapUsageBar.className = 'progress-bar';
    if (swap.usage_percent > 75) {
        swapUsageBar.classList.add('danger');
    } else if (swap.usage_percent > 50) {
        swapUsageBar.classList.add('warning');
    }
}

function updateSystemInfo(uptime, motherboard) {
    const mbInfo = `${motherboard.vendor} ${motherboard.name}`;
    document.getElementById('mb-info').textContent = mbInfo !== 'Unknown Unknown' ? mbInfo : 'Unknown';
    document.getElementById('system-uptime').textContent = uptime.formatted;
}

function updateSystemTime() {
    const timeElement = document.getElementById('system-time');
    if (timeElement) {
        const now = new Date();
        const formatted = now.getFullYear() + '-' + 
                         String(now.getMonth() + 1).padStart(2, '0') + '-' + 
                         String(now.getDate()).padStart(2, '0') + ' ' + 
                         String(now.getHours()).padStart(2, '0') + ':' + 
                         String(now.getMinutes()).padStart(2, '0') + ':' + 
                         String(now.getSeconds()).padStart(2, '0');
        timeElement.textContent = formatted;
    }
}

function updateNetworkInfo(interfaces) {
    const container = document.getElementById('network-interfaces');
    if (!container) return;
    
    if (interfaces.length === 0) {
        container.innerHTML = '<p style="color: #666; text-align: center;">No network interfaces found.</p>';
        return;
    }
    
    let html = '<div class="network-grid">';
    
    interfaces.forEach(iface => {
        const statusClass = iface.is_up ? 'connected' : 'disconnected';
        const icon = iface.type === 'Wireless' ? '📡' : 
                     iface.type === 'Ethernet' ? '🔌' : 
                     iface.type === 'Virtual' ? '🔗' : '🌐';
        
        // Calculate speed if we have previous data
        let rxSpeed = 'N/A';
        let txSpeed = 'N/A';
        
        if (lastNetworkStats[iface.name]) {
            const timeDiff = 3; // seconds between updates
            const rxDiff = iface.rx_bytes - lastNetworkStats[iface.name].rx_bytes;
            const txDiff = iface.tx_bytes - lastNetworkStats[iface.name].tx_bytes;
            
            rxSpeed = formatSpeed(rxDiff / timeDiff);
            txSpeed = formatSpeed(txDiff / timeDiff);
        }
        
        // Store current stats for next update
        lastNetworkStats[iface.name] = {
            rx_bytes: iface.rx_bytes,
            tx_bytes: iface.tx_bytes
        };
        
        html += `
            <div class="network-interface-card">
                <div class="network-interface-header">
                    <div class="network-interface-name">${icon} ${iface.name}</div>
                    <div class="network-status ${statusClass}">${iface.status}</div>
                </div>
                <div class="network-detail">
                    <span class="network-detail-label">Type:</span>
                    <span class="network-detail-value">${iface.type}</span>
                </div>
                <div class="network-detail">
                    <span class="network-detail-label">IP Address:</span>
                    <span class="network-detail-value">${iface.ip}</span>
                </div>
                <div class="network-detail">
                    <span class="network-detail-label">MAC Address:</span>
                    <span class="network-detail-value">${iface.mac}</span>
                </div>
                <div class="network-detail">
                    <span class="network-detail-label">Speed:</span>
                    <span class="network-detail-value">${iface.speed}</span>
                </div>
                <div class="network-traffic">
                    <div class="network-traffic-item">
                        <div class="network-traffic-label">⬇ Download</div>
                        <div class="network-traffic-value">${iface.rx_formatted}</div>
                        ${rxSpeed !== 'N/A' ? `<div style="color: #4caf50; font-size: 12px; margin-top: 4px;">${rxSpeed}/s</div>` : ''}
                    </div>
                    <div class="network-traffic-item">
                        <div class="network-traffic-label">⬆ Upload</div>
                        <div class="network-traffic-value">${iface.tx_formatted}</div>
                        ${txSpeed !== 'N/A' ? `<div style="color: #2196f3; font-size: 12px; margin-top: 4px;">${txSpeed}/s</div>` : ''}
                    </div>
                </div>
            </div>
        `;
    });
    
    html += '</div>';
    container.innerHTML = html;
}

function formatSpeed(bytesPerSecond) {
    const units = ['B', 'KB', 'MB', 'GB'];
    let value = bytesPerSecond;
    let unitIndex = 0;
    
    while (value >= 1024 && unitIndex < units.length - 1) {
        value /= 1024;
        unitIndex++;
    }
    
    return value.toFixed(2) + ' ' + units[unitIndex];
}
