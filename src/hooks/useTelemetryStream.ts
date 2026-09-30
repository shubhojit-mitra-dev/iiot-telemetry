import { useState, useEffect } from 'react';

export interface TelemetryPayload {
  device_id: string;
  timestamp: number;
  temperature: number;
  vibration: number;
  rpm: number;
}

export const useTelemetryStream = (isMock = true) => {
  const [data, setData] = useState<TelemetryPayload[]>([]);
  const [latestByDevice, setLatestByDevice] = useState<Record<string, TelemetryPayload>>({});
  const [anomaly, setAnomaly] = useState<TelemetryPayload | null>(null);

  useEffect(() => {
    if (!isMock) return;

    const interval = setInterval(() => {
      const now = Date.now();
      const deviceId = `Device-0${Math.floor(Math.random() * 5) + 1}`;
      
      const isAnomaly = Math.random() < 0.02;
      const temperature = isAnomaly ? 120 + Math.random() * 10 : 80 + Math.random() * 20;
      
      const payload: TelemetryPayload = {
        device_id: deviceId,
        timestamp: now,
        temperature,
        vibration: 5 + Math.random() * 5,
        rpm: 3000 + Math.random() * 500,
      };

      if (temperature > 120) {
        setAnomaly(payload);
      }

      setData((prev) => {
        const newData = [...prev, payload];
        return newData.slice(-50); // Sliding window
      });

      setLatestByDevice((prev) => ({
        ...prev,
        [deviceId]: payload,
      }));

    }, 1000); // 1 tick per second for smooth chart rendering

    return () => clearInterval(interval);
  }, [isMock]);

  return { data, latestByDevice, anomaly, clearAnomaly: () => setAnomaly(null) };
};
