import { useEffect, useState } from 'react';
import { temperatureAPI, TemperatureInfo, SensorReading } from '../../api';
import './Widgets.css';

function TemperatureWidget() {
  const [temps, setTemps] = useState<TemperatureInfo | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchTemperatures();
    const interval = setInterval(fetchTemperatures, 5000);
    return () => clearInterval(interval);
  }, []);

  const fetchTemperatures = async () => {
    try {
      const data = await temperatureAPI.get();
      setTemps(data);
      setLoading(false);
    } catch (error) {
      console.error('Failed to fetch temperatures:', error);
      setLoading(false);
    }
  };

  const getTemperatureClass = (sensor: SensorReading): string => {
    if (sensor.critical && sensor.temperature >= sensor.critical) return 'temp-critical';
    if (sensor.high && sensor.temperature >= sensor.high) return 'temp-high';
    if (sensor.temperature >= 80) return 'temp-high';
    if (sensor.temperature >= 60) return 'temp-warning';
    return 'temp-normal';
  };

  const renderSensorList = (sensors: SensorReading[], title: string) => {
    if (!sensors || sensors.length === 0) return null;

    return (
      <div className="temp-category">
        <h4>{title}</h4>
        <div className="temp-sensors">
          {sensors.map((sensor, index) => (
            <div key={index} className={`temp-sensor ${getTemperatureClass(sensor)}`}>
              <span className="sensor-name">{sensor.name}</span>
              <span className="sensor-temp">{sensor.temperature.toFixed(1)}{sensor.unit}</span>
            </div>
          ))}
        </div>
      </div>
    );
  };

  if (loading) {
    return (
      <div className="widget widget-temperature">
        <div className="widget-header">Temperature</div>
        <div className="widget-body loading">Loading...</div>
      </div>
    );
  }

  if (!temps || (temps.cpu.length === 0 && temps.gpu.length === 0 && temps.disks.length === 0 && temps.sensors.length === 0)) {
    return (
      <div className="widget widget-temperature">
        <div className="widget-header">Temperature</div>
        <div className="widget-body no-data">No temperature sensors available</div>
      </div>
    );
  }

  return (
    <div className="widget widget-temperature">
      <div className="widget-header">Temperature</div>
      <div className="widget-body">
        {renderSensorList(temps.cpu, 'CPU')}
        {renderSensorList(temps.gpu, 'GPU')}
        {renderSensorList(temps.disks, 'Disks')}
        {renderSensorList(temps.sensors, 'Other Sensors')}
      </div>
    </div>
  );
}

export default TemperatureWidget;
