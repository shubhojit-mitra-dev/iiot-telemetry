import { useState, useEffect, useRef } from 'react';

// --- Device Configuration ---
export interface DeviceConfig {
  id: string;
  name: string;
  type: string;
  baseTemp: number;
  baseVib: number;
  baseRpm: number;
}

export const DEVICES: DeviceConfig[] = [
  { id: 'TURB-001', name: 'Gas Turbine A', type: 'Turbine', baseTemp: 82, baseVib: 6.5, baseRpm: 3200 },
  { id: 'TURB-002', name: 'Gas Turbine B', type: 'Turbine', baseTemp: 85, baseVib: 7.0, baseRpm: 3150 },
  { id: 'COMP-001', name: 'Air Compressor', type: 'Compressor', baseTemp: 78, baseVib: 5.8, baseRpm: 2800 },
  { id: 'PUMP-001', name: 'Coolant Pump A', type: 'Pump', baseTemp: 65, baseVib: 4.2, baseRpm: 1800 },
  { id: 'PUMP-002', name: 'Coolant Pump B', type: 'Pump', baseTemp: 67, baseVib: 4.5, baseRpm: 1820 },
  { id: 'CONV-001', name: 'Main Conveyor', type: 'Conveyor', baseTemp: 55, baseVib: 3.8, baseRpm: 900 },
  { id: 'WELD-001', name: 'Robotic Welder 1', type: 'Welder', baseTemp: 95, baseVib: 8.2, baseRpm: 2400 },
  { id: 'WELD-002', name: 'Robotic Welder 2', type: 'Welder', baseTemp: 92, baseVib: 7.9, baseRpm: 2380 },
  { id: 'CNC-001', name: 'CNC Lathe Alpha', type: 'CNC', baseTemp: 72, baseVib: 5.5, baseRpm: 4500 },
  { id: 'CNC-002', name: 'CNC Mill Beta', type: 'CNC', baseTemp: 74, baseVib: 5.8, baseRpm: 4200 },
];

export const DEVICE_COLORS: Record<string, string> = {
  'TURB-001': '#5794F2', 'TURB-002': '#73BF69', 'COMP-001': '#FADE2A',
  'PUMP-001': '#FF9830', 'PUMP-002': '#F2495C', 'CONV-001': '#B877D9',
  'WELD-001': '#FF6EB4', 'WELD-002': '#8AB8FF', 'CNC-001': '#00C9FF',
  'CNC-002': '#C8F2C2',
};

// --- Data Types ---
export interface TelemetryPayload {
  device_id: string;
  timestamp: number;
  temperature: number;
  vibration: number;
  rpm: number;
}

export interface Incident {
  id: string;
  device_id: string;
  device_name: string;
  timestamp: number;
  temperature: number;
  vibration: number;
  severity: 'warning' | 'critical';
  message: string;
}

export interface TelemetryStats {
  activeDevices: number;
  totalDevices: number;
  messagesIngested: number;
  avgTemperature: number;
  peakTemperature: number;
  avgVibration: number;
  avgRpm: number;
  anomalyCount: number;
}

export type ChartDataPoint = { timestamp: number; [deviceId: string]: number };

// --- AI Diagnostics ---
const AI_DIAGNOSTICS = [
  'Potential bearing degradation detected. Synchronous temperature rise with vibration spike suggests inner race wear. Recommend immediate lubrication system inspection.',
  'Thermal runaway pattern identified. Excessive heat generation correlates with RPM instability. Possible motor winding fault — schedule preventive shutdown.',
  'Cavitation signature detected in vibration spectrum. Temperature anomaly consistent with insufficient coolant flow. Check impeller and inlet valve.',
  'Abnormal friction coefficient inferred from heat signature. Vibration harmonics suggest gear mesh fault. Plan replacement during next maintenance window.',
  'Electrical imbalance detected. Temperature spike with stable RPM indicates potential stator winding short. Isolate circuit and perform insulation resistance test.',
];

// --- Data Generator ---
function generatePayload(device: DeviceConfig, timestamp: number, forceAnomaly = false): TelemetryPayload {
  const isAnomaly = forceAnomaly || Math.random() < 0.015;
  return {
    device_id: device.id,
    timestamp,
    temperature: isAnomaly ? 120 + Math.random() * 15 : device.baseTemp + (Math.random() - 0.5) * 8,
    vibration: isAnomaly ? device.baseVib + 5 + Math.random() * 3 : device.baseVib + (Math.random() - 0.5) * 3,
    rpm: device.baseRpm + (Math.random() - 0.5) * 200,
  };
}

// --- Hook ---
export function useTelemetryStream() {
  const [tempSeries, setTempSeries] = useState<ChartDataPoint[]>([]);
  const [vibSeries, setVibSeries] = useState<ChartDataPoint[]>([]);
  const [rpmSeries, setRpmSeries] = useState<ChartDataPoint[]>([]);
  const [latestByDevice, setLatestByDevice] = useState<Record<string, TelemetryPayload>>({});
  const [incidents, setIncidents] = useState<Incident[]>([]);
  const [stats, setStats] = useState<TelemetryStats>({
    activeDevices: 10, totalDevices: 10, messagesIngested: 0,
    avgTemperature: 0, peakTemperature: 0, avgVibration: 0, avgRpm: 0, anomalyCount: 0,
  });

  const msgCount = useRef(0);
  const incCount = useRef(0);

  useEffect(() => {
    const now = Date.now();
    const initTemp: ChartDataPoint[] = [];
    const initVib: ChartDataPoint[] = [];
    const initRpm: ChartDataPoint[] = [];
    const initLatest: Record<string, TelemetryPayload> = {};
    const initIncidents: Incident[] = [];

    for (let i = 60; i > 0; i--) {
      const ts = now - i * 1000;
      const tp: ChartDataPoint = { timestamp: ts };
      const vp: ChartDataPoint = { timestamp: ts };
      const rp: ChartDataPoint = { timestamp: ts };
      for (const d of DEVICES) {
        const p = generatePayload(d, ts);
        tp[d.id] = p.temperature;
        vp[d.id] = p.vibration;
        rp[d.id] = p.rpm;
        initLatest[d.id] = p;
        if (p.temperature > 120) {
          incCount.current++;
          initIncidents.push({
            id: `inc-${incCount.current}`, device_id: d.id, device_name: d.name,
            timestamp: ts, temperature: p.temperature, vibration: p.vibration,
            severity: p.temperature > 130 ? 'critical' : 'warning',
            message: AI_DIAGNOSTICS[Math.floor(Math.random() * AI_DIAGNOSTICS.length)],
          });
        }
      }
      initTemp.push(tp); initVib.push(vp); initRpm.push(rp);
    }
    msgCount.current = 60 * DEVICES.length;
    setTempSeries(initTemp); setVibSeries(initVib); setRpmSeries(initRpm);
    setLatestByDevice(initLatest); setIncidents(initIncidents.slice(-30));

    const interval = setInterval(() => {
      const ts = Date.now();
      const tp: ChartDataPoint = { timestamp: ts };
      const vp: ChartDataPoint = { timestamp: ts };
      const rp: ChartDataPoint = { timestamp: ts };
      const nl: Record<string, TelemetryPayload> = {};
      for (const d of DEVICES) {
        const p = generatePayload(d, ts);
        tp[d.id] = p.temperature; vp[d.id] = p.vibration; rp[d.id] = p.rpm;
        nl[d.id] = p;
        if (p.temperature > 120) {
          incCount.current++;
          const newIncident: Incident = {
            id: `inc-${incCount.current}`,
            device_id: d.id,
            device_name: d.name,
            timestamp: ts,
            temperature: p.temperature,
            vibration: p.vibration,
            severity: p.temperature > 130 ? 'critical' : 'warning',
            message: AI_DIAGNOSTICS[Math.floor(Math.random() * AI_DIAGNOSTICS.length)],
          };
          setIncidents(prev => [newIncident, ...prev].slice(0, 50));
        }
      }
      msgCount.current += DEVICES.length;
      setTempSeries(prev => [...prev, tp].slice(-60));
      setVibSeries(prev => [...prev, vp].slice(-60));
      setRpmSeries(prev => [...prev, rp].slice(-60));
      setLatestByDevice(prev => ({ ...prev, ...nl }));
      const vals = Object.values(nl);
      const temps = vals.map(v => v.temperature);
      const vibs = vals.map(v => v.vibration);
      const rpms = vals.map(v => v.rpm);
      setStats({
        activeDevices: 10, totalDevices: 10, messagesIngested: msgCount.current,
        avgTemperature: temps.reduce((a, b) => a + b, 0) / temps.length,
        peakTemperature: Math.max(...temps),
        avgVibration: vibs.reduce((a, b) => a + b, 0) / vibs.length,
        avgRpm: rpms.reduce((a, b) => a + b, 0) / rpms.length,
        anomalyCount: incCount.current,
      });
    }, 1000);
    return () => clearInterval(interval);
  }, []);

  return { tempSeries, vibSeries, rpmSeries, latestByDevice, incidents, stats };
}
