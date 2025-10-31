# System Monitoring Dashboard - Quick Start Guide

## 🚀 Getting Started

The system monitoring dashboard is automatically enabled on the main dashboard. Simply log in to your TSO instance and navigate to the Dashboard page.

## 📊 What You'll See

### Real-Time Monitoring Cards

The dashboard displays 4 main monitoring cards:

1. **⚙️ CPU Card**
   - Shows your processor model and specifications
   - Live CPU usage with color-coded progress bar
   - System load averages

2. **💾 Memory (RAM) Card**
   - Total installed memory
   - Current usage and available memory
   - Visual progress bar showing utilization

3. **💿 Swap Card**
   - Swap space configuration
   - Current swap usage
   - Helps identify memory pressure

4. **🖥️ System Info Card**
   - Motherboard information
   - System uptime
   - Hostname and current time

### Network Interfaces Section

Below the monitoring cards, you'll find detailed information about all network interfaces:

- **Connection Status**: Green badge for connected, red for disconnected
- **Interface Type**: Ethernet 🔌, Wireless 📡, or Virtual 🔗
- **Network Details**: IP address, MAC address, link speed
- **Traffic Statistics**: 
  - Total data transferred (download/upload)
  - Current transfer speeds (updated every 3 seconds)

## 🎨 Color Coding

The progress bars use an intelligent color system:

| Color | CPU Usage | RAM Usage | Swap Usage | Meaning |
|-------|-----------|-----------|------------|---------|
| 🟢 Green | 0-60% | 0-70% | 0-50% | Normal, healthy |
| 🟠 Orange | 60-80% | 70-85% | 50-75% | Warning, monitor |
| 🔴 Red | >80% | >85% | >75% | Critical, take action |

## ⚡ Update Frequency

- **System Stats**: Updates every 3 seconds
- **System Time**: Updates every second
- **Network Speeds**: Calculated from consecutive updates

## 🔧 Customization Options

### Change Update Frequency

Edit `/public/js/main.js` and modify line 55:

```javascript
// Update every 3 seconds (change 3000 to desired milliseconds)
setInterval(fetchSystemStats, 3000);
```

Example: For 5-second updates, use `5000`

### Modify Color Thresholds

In `/public/js/main.js`, adjust the threshold values:

**CPU (lines 90-93):**
```javascript
if (cpu.usage > 80) {  // Change 80 to your red threshold
    cpuUsageBar.classList.add('danger');
} else if (cpu.usage > 60) {  // Change 60 to your orange threshold
    cpuUsageBar.classList.add('warning');
}
```

**RAM (lines 109-113):**
```javascript
if (memory.usage_percent > 85) {  // Red threshold
    memUsageBar.classList.add('danger');
} else if (memory.usage_percent > 70) {  // Orange threshold
    memUsageBar.classList.add('warning');
}
```

**Swap (lines 128-132):**
```javascript
if (swap.usage_percent > 75) {  // Red threshold
    swapUsageBar.classList.add('danger');
} else if (swap.usage_percent > 50) {  // Orange threshold
    swapUsageBar.classList.add('warning');
}
```

### Hide Specific Network Interfaces

Edit `/public/api/system-stats.php` around line 140:

```php
foreach ($interfaceList as $interface) {
    if (empty($interface) || $interface === 'lo') {
        continue;
    }
    
    // Add interfaces to skip
    if (in_array($interface, ['docker0', 'br-*', 'veth*'])) {
        continue;
    }
    
    // ... rest of code
}
```

### Change Progress Bar Colors

Edit `/public/css/style.css` to customize the gradient colors:

```css
.progress-bar {
    background: linear-gradient(90deg, #ff8c00, #ffa500);  /* Normal: Orange gradient */
}

.progress-bar.warning {
    background: linear-gradient(90deg, #ff9800, #ffb347);  /* Warning: Light orange */
}

.progress-bar.danger {
    background: linear-gradient(90deg, #f44336, #ff6b6b);  /* Danger: Red gradient */
}
```

## 🐛 Troubleshooting

### Issue: Some Information Shows "Unknown" or "N/A"

**Possible Causes:**
1. Insufficient permissions to read system information
2. Running in a containerized environment (Docker, LXC)
3. Virtual machine with limited hardware exposure

**Solutions:**
- Ensure the web server has read access to `/proc`, `/sys` directories
- For motherboard info, try installing `dmidecode`: `sudo apt install dmidecode`
- Some information may not be available in virtualized environments

### Issue: Network Speed Shows "N/A"

**Cause:** Virtual or loopback interfaces often don't report speed

**Solution:** This is normal for virtual interfaces (docker, veth, etc.)

### Issue: High Swap Usage (Red)

**What it means:** Your system is using swap memory, indicating RAM pressure

**Recommendations:**
1. Check what's using memory (top/htop command)
2. Consider adding more RAM
3. Optimize applications to use less memory
4. Review which services are running

### Issue: CPU Usage Always Shows 0% or 100%

**Possible Causes:**
1. The `top` or `mpstat` command might not be installed
2. System is too fast/slow to measure accurately

**Solution:**
```bash
# Install sysstat package for better CPU monitoring
sudo apt install sysstat  # Ubuntu/Debian
sudo yum install sysstat  # CentOS/RHEL
```

### Issue: Dashboard Doesn't Update

**Check:**
1. Browser console for JavaScript errors (F12)
2. Network tab shows successful API calls to `/api/system-stats.php`
3. Verify you're logged in (API requires authentication)
4. Check web server error logs

## 📖 API Endpoint

The monitoring data is available via REST API:

**Endpoint:** `/api/system-stats.php`  
**Method:** GET  
**Authentication:** Required (must be logged in)  
**Response:** JSON

### Example Response:
```json
{
  "cpu": {
    "model": "Intel Core i7-9700K",
    "cores": 8,
    "architecture": "x86_64",
    "frequency": "3600.00 MHz",
    "usage": 45.3,
    "load_avg": {
      "1min": 1.23,
      "5min": 1.45,
      "15min": 1.67
    }
  },
  "memory": {
    "total": 17179869184,
    "total_formatted": "16.00 GB",
    "used": 9074934784,
    "used_formatted": "8.45 GB",
    "usage_percent": 70.2
  },
  "network": [
    {
      "name": "eth0",
      "status": "Connected",
      "is_up": true,
      "type": "Ethernet",
      "ip": "192.168.1.100",
      "speed": "1000 Mbps",
      "rx_bytes": 2518442496,
      "tx_bytes": 935854080
    }
  ]
}
```

## 🔒 Security Notes

- API endpoint requires authentication (logged-in user)
- All system commands use read-only operations
- No user input is passed to system commands
- Limited to information available to the web server user

## 💡 Tips & Best Practices

1. **Monitor Regularly**: Check the dashboard daily to understand your system's normal patterns
2. **Set Up Alerts**: If you see consistent orange/red indicators, investigate
3. **Network Traffic**: Watch for unexpected spikes that might indicate issues
4. **Uptime**: Regular monitoring helps track stability and planned maintenance
5. **Resource Planning**: Use historical observations to plan upgrades

## 📚 Additional Resources

- Full documentation: `/MONITORING.md`
- Implementation details: `/SYSTEM_MONITORING_SUMMARY.md`
- Test script: `php /tools/test-monitoring.php`

## 🆘 Support

If you encounter issues:
1. Check the troubleshooting section above
2. Review browser console (F12) for JavaScript errors
3. Check web server error logs
4. Verify system requirements (Linux with /proc and /sys filesystems)

---

**Enjoy your new system monitoring dashboard! 📊🚀**

