package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type SensorReading struct {
	Name        string  `json:"name"`
	Temperature float64 `json:"temperature"`
	Critical    float64 `json:"critical,omitempty"`
	High        float64 `json:"high,omitempty"`
	Unit        string  `json:"unit"`
}

type TemperatureInfo struct {
	CPU     []SensorReading `json:"cpu"`
	GPU     []SensorReading `json:"gpu"`
	Disks   []SensorReading `json:"disks"`
	Sensors []SensorReading `json:"sensors"`
}

func GetTemperatureHandler(w http.ResponseWriter, r *http.Request) {
	info := TemperatureInfo{
		CPU:     []SensorReading{},
		GPU:     []SensorReading{},
		Disks:   []SensorReading{},
		Sensors: []SensorReading{},
	}

	// Read from thermal zones
	info.CPU = append(info.CPU, readThermalZones()...)

	// Read from hwmon
	hwmonReadings := readHwmon()
	for _, reading := range hwmonReadings {
		nameLower := strings.ToLower(reading.Name)
		if strings.Contains(nameLower, "cpu") || strings.Contains(nameLower, "core") ||
			strings.Contains(nameLower, "package") || strings.Contains(nameLower, "tctl") {
			info.CPU = append(info.CPU, reading)
		} else if strings.Contains(nameLower, "gpu") || strings.Contains(nameLower, "nvidia") ||
			strings.Contains(nameLower, "amdgpu") || strings.Contains(nameLower, "radeon") {
			info.GPU = append(info.GPU, reading)
		} else {
			info.Sensors = append(info.Sensors, reading)
		}
	}

	// Read disk temperatures
	info.Disks = readDiskTemperatures()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"temperature": info,
	})
}

func readThermalZones() []SensorReading {
	var readings []SensorReading
	thermalPath := "/sys/class/thermal"

	entries, err := os.ReadDir(thermalPath)
	if err != nil {
		return readings
	}

	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), "thermal_zone") {
			continue
		}

		zonePath := filepath.Join(thermalPath, entry.Name())

		// Read type
		typeData, err := os.ReadFile(filepath.Join(zonePath, "type"))
		if err != nil {
			continue
		}
		zoneType := strings.TrimSpace(string(typeData))

		// Read temperature
		tempData, err := os.ReadFile(filepath.Join(zonePath, "temp"))
		if err != nil {
			continue
		}
		tempMilli, err := strconv.ParseInt(strings.TrimSpace(string(tempData)), 10, 64)
		if err != nil {
			continue
		}
		temp := float64(tempMilli) / 1000.0

		// Skip invalid readings
		if temp <= 0 || temp > 150 {
			continue
		}

		reading := SensorReading{
			Name:        zoneType,
			Temperature: temp,
			Unit:        "°C",
		}

		// Try to read trip points for critical/high thresholds
		tripFiles, _ := filepath.Glob(filepath.Join(zonePath, "trip_point_*_temp"))
		for _, tripFile := range tripFiles {
			tripData, err := os.ReadFile(tripFile)
			if err != nil {
				continue
			}
			tripTemp, err := strconv.ParseInt(strings.TrimSpace(string(tripData)), 10, 64)
			if err != nil {
				continue
			}
			tripVal := float64(tripTemp) / 1000.0

			// Read trip type
			tripTypeFile := strings.Replace(tripFile, "_temp", "_type", 1)
			tripTypeData, err := os.ReadFile(tripTypeFile)
			if err != nil {
				continue
			}
			tripType := strings.TrimSpace(string(tripTypeData))

			if tripType == "critical" && tripVal > 0 {
				reading.Critical = tripVal
			} else if tripType == "hot" || tripType == "high" {
				if tripVal > 0 {
					reading.High = tripVal
				}
			}
		}

		readings = append(readings, reading)
	}

	return readings
}

func readHwmon() []SensorReading {
	var readings []SensorReading
	hwmonPath := "/sys/class/hwmon"

	entries, err := os.ReadDir(hwmonPath)
	if err != nil {
		return readings
	}

	for _, entry := range entries {
		devicePath := filepath.Join(hwmonPath, entry.Name())

		// Get device name
		nameData, err := os.ReadFile(filepath.Join(devicePath, "name"))
		deviceName := "unknown"
		if err == nil {
			deviceName = strings.TrimSpace(string(nameData))
		}

		// Find all temperature inputs
		tempFiles, _ := filepath.Glob(filepath.Join(devicePath, "temp*_input"))
		for _, tempFile := range tempFiles {
			tempData, err := os.ReadFile(tempFile)
			if err != nil {
				continue
			}
			tempMilli, err := strconv.ParseInt(strings.TrimSpace(string(tempData)), 10, 64)
			if err != nil {
				continue
			}
			temp := float64(tempMilli) / 1000.0

			// Skip invalid readings
			if temp <= 0 || temp > 150 {
				continue
			}

			// Get sensor number
			base := filepath.Base(tempFile)
			numMatch := regexp.MustCompile(`temp(\d+)_input`).FindStringSubmatch(base)
			if len(numMatch) < 2 {
				continue
			}
			sensorNum := numMatch[1]

			// Try to read label
			labelFile := strings.Replace(tempFile, "_input", "_label", 1)
			labelData, err := os.ReadFile(labelFile)
			sensorLabel := ""
			if err == nil {
				sensorLabel = strings.TrimSpace(string(labelData))
			}

			name := deviceName
			if sensorLabel != "" {
				name = sensorLabel
			} else if deviceName != "unknown" {
				name = fmt.Sprintf("%s temp%s", deviceName, sensorNum)
			}

			reading := SensorReading{
				Name:        name,
				Temperature: temp,
				Unit:        "°C",
			}

			// Read critical threshold
			critFile := strings.Replace(tempFile, "_input", "_crit", 1)
			critData, err := os.ReadFile(critFile)
			if err == nil {
				critMilli, err := strconv.ParseInt(strings.TrimSpace(string(critData)), 10, 64)
				if err == nil && critMilli > 0 {
					reading.Critical = float64(critMilli) / 1000.0
				}
			}

			// Read max/high threshold
			maxFile := strings.Replace(tempFile, "_input", "_max", 1)
			maxData, err := os.ReadFile(maxFile)
			if err == nil {
				maxMilli, err := strconv.ParseInt(strings.TrimSpace(string(maxData)), 10, 64)
				if err == nil && maxMilli > 0 {
					reading.High = float64(maxMilli) / 1000.0
				}
			}

			readings = append(readings, reading)
		}
	}

	return readings
}

func readDiskTemperatures() []SensorReading {
	var readings []SensorReading

	// Find all block devices
	blockDevices, err := filepath.Glob("/sys/block/sd*")
	if err != nil {
		return readings
	}

	// Also check NVMe devices
	nvmeDevices, _ := filepath.Glob("/sys/block/nvme*")
	blockDevices = append(blockDevices, nvmeDevices...)

	for _, devicePath := range blockDevices {
		deviceName := filepath.Base(devicePath)

		// For NVMe, check hwmon directly
		if strings.HasPrefix(deviceName, "nvme") {
			hwmonPath := filepath.Join(devicePath, "device", "hwmon")
			hwmonEntries, err := os.ReadDir(hwmonPath)
			if err == nil && len(hwmonEntries) > 0 {
				tempFile := filepath.Join(hwmonPath, hwmonEntries[0].Name(), "temp1_input")
				tempData, err := os.ReadFile(tempFile)
				if err == nil {
					tempMilli, err := strconv.ParseInt(strings.TrimSpace(string(tempData)), 10, 64)
					if err == nil && tempMilli > 0 {
						temp := float64(tempMilli) / 1000.0
						if temp > 0 && temp < 150 {
							readings = append(readings, SensorReading{
								Name:        deviceName,
								Temperature: temp,
								Unit:        "°C",
							})
						}
					}
				}
			}
			continue
		}

		// For SATA/SAS, try smartctl
		cmd := exec.Command("smartctl", "-A", "-d", "auto", "/dev/"+deviceName)
		output, err := cmd.Output()
		if err != nil {
			continue
		}

		// Parse temperature from SMART data
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			// Look for temperature attribute (ID 194 or 190)
			if strings.Contains(line, "Temperature_Celsius") ||
				strings.Contains(line, "Airflow_Temperature") {
				fields := strings.Fields(line)
				if len(fields) >= 10 {
					// Raw value is usually in the last column
					tempStr := fields[len(fields)-1]
					// Sometimes it's in format "temp Min/Max/Current"
					tempParts := strings.Split(tempStr, " ")
					tempStr = tempParts[0]

					temp, err := strconv.ParseFloat(tempStr, 64)
					if err == nil && temp > 0 && temp < 150 {
						readings = append(readings, SensorReading{
							Name:        deviceName,
							Temperature: temp,
							Unit:        "°C",
						})
						break
					}
				}
			}
		}
	}

	return readings
}
