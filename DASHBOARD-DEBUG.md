# Dashboard Auto-Update Debugging Guide

## Changes Made

I've enhanced the dashboard auto-update functionality with extensive debugging to identify why the data isn't displaying properly even though fetch requests are happening.

### What Was Added:

1. **Comprehensive Console Logging** - Every update function now logs:
   - When it's called
   - What data it receives
   - What elements it's trying to update
   - Any errors that occur

2. **Better Error Handling** - Each update function is wrapped in try-catch blocks

3. **Validation Checks** - Before updating, the code verifies:
   - DOM elements exist
   - Data fields are present
   - Values are valid

## How to Debug

### Step 1: Open Browser Developer Console

1. Open your dashboard in the browser
2. Press `F12` or right-click and select "Inspect"
3. Go to the "Console" tab

### Step 2: Check the Console Output

You should see logs like this every second:

```
Dashboard loaded, initializing auto-refresh...
Starting auto-refresh - updates every 1 second
Received data: {cpu: {...}, memory: {...}, swap: {...}, ...}
Updating CPU: {usage: 45.2, load_avg: {...}}
updateCpuStats called with: {usage: 45.2, load_avg: {...}}
Updating Memory: {used_formatted: "4.2 GB", available_formatted: "11.8 GB", ...}
updateMemoryStats called with: {...}
...
```

### Step 3: Look for Warnings or Errors

**Common issues you might see:**

#### Issue 1: Elements Not Found
```
⚠ CPU usage elements not found or data missing
```
**Solution**: This means the HTML element IDs don't match. Check that the page has elements with IDs like:
- `cpu-usage-bar`
- `cpu-usage-text`
- `mem-usage-bar`
- etc.

#### Issue 2: Data Missing
```
⚠ Memory usage text element not found or data missing
  memUsage: true
  used: undefined
  available: undefined
```
**Solution**: The API isn't returning the expected data format. Run the test script (see below).

#### Issue 3: API Errors
```
✗ Error fetching system stats: 401 Unauthorized
```
**Solution**: Authentication issue. Make sure you're logged in.

### Step 4: Check Network Tab

1. Go to the "Network" tab in Developer Tools
2. Look for requests to `/api/system-stats.php`
3. Click on one to see:
   - **Status**: Should be `200 OK`
   - **Response**: Should show JSON data
   - **Preview**: Shows formatted data

### Step 5: Test the API Directly

Run the included test script:

```bash
cd /home/user/git/tso
php test-api.php
```

This will show you exactly what data the API is returning.

## Expected Behavior

When working correctly, you should see:

1. ✅ Console logs every second showing data updates
2. ✅ Network requests every second to `/api/system-stats.php`
3. ✅ Green pulsing indicator in bottom-right corner
4. ✅ Progress bars and numbers updating in real-time
5. ✅ "Last update" shows "Just now"

## Common Problems and Solutions

### Problem: Console shows "Page not visible, skipping update"
**Cause**: Tab is in background
**Solution**: Make sure the dashboard tab is active and visible

### Problem: No console logs at all
**Cause**: JavaScript not loading or browser cache
**Solution**: 
- Hard refresh: `Ctrl+F5` (Windows/Linux) or `Cmd+Shift+R` (Mac)
- Clear browser cache
- Check browser console for JavaScript errors on page load

### Problem: API returns error
**Cause**: Authentication or permission issues
**Solution**: 
- Make sure you're logged in
- Check PHP error logs: `sudo tail -f /var/log/apache2/error.log` or similar
- Verify file permissions on `/public/api/system-stats.php`

### Problem: Data is received but not displaying
**Cause**: Element IDs don't match or CSS hiding elements
**Solution**: Check console warnings - they'll tell you exactly which elements are missing

### Problem: Updates very slow or inconsistent
**Cause**: Server performance issues
**Solution**: The API might be taking too long. Check server resources.

## Quick Verification Checklist

Run through this checklist:

- [ ] Dashboard page loads without errors
- [ ] Green indicator visible in bottom-right
- [ ] Browser console shows "Dashboard loaded, initializing auto-refresh..."
- [ ] Console shows "Received data:" every second
- [ ] Network tab shows requests to `/api/system-stats.php` with 200 status
- [ ] API response contains valid JSON with cpu, memory, swap, uptime fields
- [ ] No console errors or warnings
- [ ] Progress bars visible on page
- [ ] Numbers updating when watching closely

## Additional Tools

### View Raw API Response

Open this URL in your browser (while logged in):
```
http://your-server/api/system-stats.php
```

You should see JSON like:
```json
{
    "cpu": {
        "usage": 45.2,
        "load_avg": {
            "1min": 1.23,
            "5min": 1.45,
            "15min": 1.67
        }
    },
    "memory": {
        "usage_percent": 35.4,
        "used_formatted": "4.2 GB",
        "available_formatted": "11.8 GB"
    },
    ...
}
```

### Monitor Network Traffic

In browser console, run:
```javascript
// This will log every successful update
window.updateCount = 0;
const originalFlash = flashRefreshIndicator;
flashRefreshIndicator = function() {
    window.updateCount++;
    console.log('✓ Update #' + window.updateCount);
    originalFlash();
};
```

## Getting Help

If you're still having issues, provide:

1. All console output (especially errors and warnings)
2. Network tab showing the API request/response
3. Output from `php test-api.php`
4. Browser and version
5. Any PHP error logs

## Files Modified

- `/home/user/git/tso/public/dashboard.php` - Enhanced with debugging
- `/home/user/git/tso/test-api.php` - New test script
- `/home/user/git/tso/DASHBOARD-DEBUG.md` - This file

