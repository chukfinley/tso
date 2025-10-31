# System Monitoring Dashboard

This document describes the comprehensive system monitoring features added to the dashboard.

## Features Overview

The dashboard now includes real-time monitoring of the following system metrics:

### 1. **CPU Monitoring**
- **Model**: Displays the CPU model name and brand
- **Cores & Architecture**: Shows the number of CPU cores and system architecture (x86_64, ARM, etc.)
- **Frequency**: Current CPU frequency in MHz
- **Usage**: Real-time CPU usage percentage with color-coded progress bar
  - Green (0-60%): Normal usage
  - Orange (60-80%): Warning - moderate usage
  - Red (>80%): Danger - high usage
- **Load Average**: System load average for 1, 5, and 15 minutes

### 2. **Memory (RAM) Monitoring**
- **Total Memory**: Total installed RAM
- **Used/Available**: Shows memory currently in use vs. available memory
- **Usage Percentage**: Color-coded progress bar showing memory utilization
  - Green (0-70%): Normal usage
  - Orange (70-85%): Warning - high usage
  - Red (>85%): Danger - critical usage

### 3. **Swap Monitoring**
- **Total Swap**: Total swap space configured
- **Used/Free**: Current swap usage vs. free swap space
- **Usage Percentage**: Color-coded progress bar
  - Green (0-50%): Normal usage
  - Orange (50-75%): Warning - moderate swap usage
  - Red (>75%): Danger - heavy swap usage (may indicate insufficient RAM)

### 4. **System Information**
- **Motherboard**: Displays motherboard vendor and model
- **System Uptime**: Shows how long the system has been running
- **Hostname**: System hostname
- **System Time**: Current system time (updates every second)

### 5. **Network Monitoring**
- **Network Interfaces**: Displays all network interfaces (Ethernet, Wireless, Virtual)
- **Connection Status**: Shows if each interface is connected or disconnected
- **Interface Type**: Identifies Ethernet, Wireless, or Virtual interfaces
- **IP Address**: Current IP address assigned to each interface
- **MAC Address**: Hardware MAC address
- **Link Speed**: Maximum supported speed in Mbps
- **Traffic Statistics**:
  - **Download**: Total bytes received and current download speed
  - **Upload**: Total bytes transmitted and current upload speed
  - Real-time speed calculations (B/s, KB/s, MB/s, GB/s)

## Technical Implementation

### Backend API
**File**: `/public/api/system-stats.php`

This API endpoint collects system information using native Linux commands and the `/proc` filesystem:
- CPU info from `/proc/cpuinfo` and `lscpu`
- Memory info from `/proc/meminfo`
- Network stats from `/sys/class/net/`
- Motherboard info from `/sys/class/dmi/id/`
- System uptime from `/proc/uptime`

The API returns JSON data with all system metrics.

### Frontend Updates
**Files**: 
- `/public/dashboard.php` - Dashboard HTML structure
- `/public/css/style.css` - Styling for monitoring cards and progress bars
- `/public/js/main.js` - JavaScript for fetching and updating data

### Update Frequency
- **System stats**: Updated every 3 seconds
- **System time**: Updated every second
- **Network speeds**: Calculated from the difference between consecutive API calls

### Visual Features
1. **Color-coded progress bars**: Provide instant visual feedback on resource usage
2. **Smooth animations**: Progress bars animate smoothly when values change
3. **Icon indicators**: Each card has a relevant icon for quick identification
4. **Network interface cards**: Grid layout with hover effects
5. **Real-time speed indicators**: Show current network throughput for each interface
6. **Status badges**: Visual indicators for network connection status

## Browser Compatibility
The monitoring dashboard uses modern JavaScript features and is compatible with:
- Chrome/Chromium 60+
- Firefox 55+
- Safari 12+
- Edge 79+

## Performance Considerations
- API calls are made every 3 seconds to balance real-time updates with server load
- Network speed calculations are done client-side to minimize server processing
- Data is fetched asynchronously to prevent UI blocking
- Failed API calls are logged to console but don't break the UI

## Security
- All API endpoints require user authentication
- System commands are executed securely using PHP's `shell_exec()` with sanitized inputs
- No user input is directly executed in system commands

## Troubleshooting

### Missing Information
If some information shows as "Unknown" or "N/A":
1. **Motherboard info**: May require running the web server with higher privileges or installing `dmidecode`
2. **CPU frequency**: Some virtual machines may not expose this information
3. **Network speed**: Virtual interfaces often report "-1" or unavailable

### Performance Issues
If the dashboard is slow:
1. Check server CPU usage - consider increasing the update interval from 3 seconds
2. Reduce the number of network interfaces being monitored
3. Check PHP's `max_execution_time` setting

## Future Enhancements
Potential additions for future versions:
- Disk I/O monitoring
- Process list and management
- Temperature sensors (CPU, GPU, drives)
- Historical data graphs
- Alert notifications for critical thresholds
- Customizable refresh intervals
- Export data to CSV/JSON

