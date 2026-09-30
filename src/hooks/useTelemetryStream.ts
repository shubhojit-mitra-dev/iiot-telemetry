/// <reference types="vite/client" />
import { useState, useEffect, useRef, useCallback } from 'react';

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
  queueDepth: number;
  activeClients: number;
}

export type ChartDataPoint = { timestamp: number; [deviceId: string]: number };

// --- AI Diagnostic Context Engine ---
const AI_DIAGNOSTICS_TEMP = [
  'Thermal runaway pattern identified. Excessive heat generation correlates with RPM instability. Possible motor winding fault — schedule preventive shutdown.',
  'High temperature excursion detected. Heat signature indicates cooling jacket blockage or degraded heat transfer fluid.',
  'Abnormal friction coefficient inferred from heat signature. Continuous thermal escalation detected. Inspect lubrication level immediately.',
];

const AI_DIAGNOSTICS_VIB = [
  'Potential bearing degradation detected. Vibration harmonics suggest inner race wear and localized flaking. Recommend ultrasonic acoustic inspection.',
  'Cavitation signature detected in vibration spectrum. Mechanical resonance consistent with pump impeller imbalance or inlet starvation.',
  'Mechanical misalignment detected. Radial vibration exceedance indicates shaft angular misalignment or loose foundation bolts.',
];

const AI_DIAGNOSTICS_COMPOUND = [
  'Critical compound failure imminent. Concurrent temperature rise and severe vibration spike indicate catastrophic bearing seizure in progress.',
  'Dynamic imbalance with thermal runaway detected. Synchronous vibration harmonics and rapid temperature climb require immediate emergency stop.',
];

function selectDiagnosticMessage(payload: TelemetryPayload): string {
  if (payload.temperature > 120 && payload.vibration > 12) {
    return AI_DIAGNOSTICS_COMPOUND[Math.floor(Math.random() * AI_DIAGNOSTICS_COMPOUND.length)];
  }
  if (payload.temperature > 120) {
    return AI_DIAGNOSTICS_TEMP[Math.floor(Math.random() * AI_DIAGNOSTICS_TEMP.length)];
  }
  return AI_DIAGNOSTICS_VIB[Math.floor(Math.random() * AI_DIAGNOSTICS_VIB.length)];
}

const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:8080';
const WS_URL = import.meta.env.VITE_WS_URL || `${API_BASE.replace(/^http/, 'ws')}/ws/telemetry`;

// --- Hook ---
export function useTelemetryStream() {
  const [tempSeries, setTempSeries] = useState<ChartDataPoint[]>([]);
  const [vibSeries, setVibSeries] = useState<ChartDataPoint[]>([]);
  const [rpmSeries, setRpmSeries] = useState<ChartDataPoint[]>([]);
  const [latestByDevice, setLatestByDevice] = useState<Record<string, TelemetryPayload>>({});
  const [incidents, setIncidents] = useState<Incident[]>([]);
  const [isConnected, setIsConnected] = useState<boolean>(false);
  const [stats, setStats] = useState<TelemetryStats>({
    activeDevices: 0,
    totalDevices: DEVICES.length,
    messagesIngested: 0,
    avgTemperature: 0,
    peakTemperature: 0,
    avgVibration: 0,
    avgRpm: 0,
    anomalyCount: 0,
    queueDepth: 0,
    activeClients: 0,
  });

  const wsRef = useRef<WebSocket | null>(null);
  const retryTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const healthIntervalRef = useRef<NodeJS.Timeout | null>(null);
  const reconnectAttemptRef = useRef<number>(0);
  const lastIncidentTimeRef = useRef<Record<string, number>>({});
  const isUnmountedRef = useRef<boolean>(false);
  const latestByDeviceRef = useRef<Record<string, TelemetryPayload>>({});

  // Dense state tracking to ensure all 10 machines have continuous, fluid chart lines
  const latestMetricsRef = useRef<{
    temp: Record<string, number>;
    vib: Record<string, number>;
    rpm: Record<string, number>;
  }>({
    temp: Object.fromEntries(DEVICES.map(d => [d.id, d.baseTemp])),
    vib: Object.fromEntries(DEVICES.map(d => [d.id, d.baseVib])),
    rpm: Object.fromEntries(DEVICES.map(d => [d.id, d.baseRpm])),
  });

  // Buffer incoming points onto sliding window charts (capped at 60 points to eliminate memory leaks)
  const appendChartPoint = useCallback((payload: TelemetryPayload) => {
    const ts = payload.timestamp > 1e11 ? payload.timestamp : payload.timestamp * 1000;

    // Update dense metric cache
    latestMetricsRef.current.temp[payload.device_id] = payload.temperature;
    latestMetricsRef.current.vib[payload.device_id] = payload.vibration;
    latestMetricsRef.current.rpm[payload.device_id] = payload.rpm;

    const updateSeries = (
      prev: ChartDataPoint[],
      metricKey: 'temp' | 'vib' | 'rpm'
    ): ChartDataPoint[] => {
      const last = prev[prev.length - 1];
      const snapshot = latestMetricsRef.current[metricKey];

      // Merge into current time bucket (500ms window) to group asynchronous fleet arrivals
      if (last && Math.abs(last.timestamp - ts) < 500) {
        const updated: ChartDataPoint = { ...last, ...snapshot, timestamp: last.timestamp };
        return [...prev.slice(0, -1), updated];
      }

      // Start new time bucket seeded with full fleet state vector
      const newPoint: ChartDataPoint = { timestamp: ts, ...snapshot };
      return [...prev, newPoint].slice(-60);
    };

    setTempSeries(prev => updateSeries(prev, 'temp'));
    setVibSeries(prev => updateSeries(prev, 'vib'));
    setRpmSeries(prev => updateSeries(prev, 'rpm'));
  }, []);

  // Update aggregated statistics derived from live device snapshot and backend health
  const refreshStats = useCallback((backendStats?: { processed: number; anomalies: number; queueDepth: number; clients: number }) => {
    const devices = Object.values(latestByDeviceRef.current);
    const activeCount = devices.length;

    if (activeCount > 0) {
      const temps = devices.map(d => d.temperature);
      const vibs = devices.map(d => d.vibration);
      const rpms = devices.map(d => d.rpm);

      const avgTemp = temps.reduce((a, b) => a + b, 0) / activeCount;
      const peakTemp = Math.max(...temps);
      const avgVib = vibs.reduce((a, b) => a + b, 0) / activeCount;
      const avgRpmVal = rpms.reduce((a, b) => a + b, 0) / activeCount;

      setStats(prev => ({
        ...prev,
        activeDevices: activeCount,
        totalDevices: DEVICES.length,
        messagesIngested: backendStats ? backendStats.processed : prev.messagesIngested,
        anomalyCount: backendStats ? backendStats.anomalies : prev.anomalyCount,
        queueDepth: backendStats ? backendStats.queueDepth : prev.queueDepth,
        activeClients: backendStats ? backendStats.clients : prev.activeClients,
        avgTemperature: avgTemp,
        peakTemperature: peakTemp,
        avgVibration: avgVib,
        avgRpm: avgRpmVal,
      }));
    } else if (backendStats) {
      setStats(prev => ({
        ...prev,
        messagesIngested: backendStats.processed,
        anomalyCount: backendStats.anomalies,
        queueDepth: backendStats.queueDepth,
        activeClients: backendStats.clients,
      }));
    }
  }, []);

  // Poll backend health stats every 3 seconds for accurate server-side throughput and queue telemetry
  const fetchHealthStats = useCallback(async () => {
    try {
      const res = await fetch(`${API_BASE}/api/v1/health`);
      if (!res.ok) return;
      const data = await res.json();
      refreshStats({
        processed: data.total_processed ?? 0,
        anomalies: data.anomalies ?? 0,
        queueDepth: data.queue_depth ?? 0,
        clients: data.active_clients ?? 0,
      });
    } catch {
      // Backend temporarily offline or unreachable
    }
  }, [refreshStats]);

  // Initial devices state fetch on mount
  const fetchInitialDevices = useCallback(async () => {
    try {
      const res = await fetch(`${API_BASE}/api/v1/devices`);
      if (!res.ok) return;
      const devicesMap: Record<string, TelemetryPayload> = await res.json();
      if (devicesMap && Object.keys(devicesMap).length > 0) {
        latestByDeviceRef.current = devicesMap;
        setLatestByDevice(devicesMap);

        // Prepopulate initial charts from snapshot
        const ts = Date.now();
        const initialTemp: ChartDataPoint = { timestamp: ts };
        const initialVib: ChartDataPoint = { timestamp: ts };
        const initialRpm: ChartDataPoint = { timestamp: ts };

        Object.values(devicesMap).forEach(payload => {
          initialTemp[payload.device_id] = payload.temperature;
          initialVib[payload.device_id] = payload.vibration;
          initialRpm[payload.device_id] = payload.rpm;
        });

        setTempSeries([initialTemp]);
        setVibSeries([initialVib]);
        setRpmSeries([initialRpm]);
      }
    } catch {
      // Server not yet running or empty initial repository
    }
  }, []);

  // Connect and maintain resilient WebSocket subscription
  const connectWebSocket = useCallback(() => {
    if (isUnmountedRef.current) return;

    try {
      const ws = new WebSocket(WS_URL);
      wsRef.current = ws;

      ws.onopen = () => {
        if (isUnmountedRef.current) {
          ws.close();
          return;
        }
        setIsConnected(true);
        reconnectAttemptRef.current = 0;
      };

      ws.onmessage = (event: MessageEvent) => {
        try {
          const payload: TelemetryPayload = JSON.parse(event.data);
          if (!payload.device_id) return;

          // 1. Update latest state per device
          latestByDeviceRef.current[payload.device_id] = payload;
          setLatestByDevice(prev => ({
            ...prev,
            [payload.device_id]: payload,
          }));

          // 2. Append to rolling chart buffers
          appendChartPoint(payload);

          // 3. Local Anomaly & AI Diagnostics Alert Detection (debounced per device)
          if (payload.temperature > 120 || payload.vibration > 12) {
            const now = Date.now();
            const lastLog = lastIncidentTimeRef.current[payload.device_id] || 0;
            if (now - lastLog > 4000) { // 4-second incident debounce per device
              lastIncidentTimeRef.current[payload.device_id] = now;
              const deviceMeta = DEVICES.find(d => d.id === payload.device_id);
              const incident: Incident = {
                id: `inc-${now}-${payload.device_id}`,
                device_id: payload.device_id,
                device_name: deviceMeta ? deviceMeta.name : payload.device_id,
                timestamp: payload.timestamp > 1e11 ? payload.timestamp : payload.timestamp * 1000,
                temperature: payload.temperature,
                vibration: payload.vibration,
                severity: payload.temperature > 130 || payload.vibration > 14 ? 'critical' : 'warning',
                message: selectDiagnosticMessage(payload),
              };
              setIncidents(prev => [incident, ...prev].slice(0, 50));
            }
          }

          // 4. Update stats
          refreshStats();
        } catch {
          // Malformed WebSocket message
        }
      };

      ws.onclose = () => {
        setIsConnected(false);
        if (isUnmountedRef.current) return;

        // Exponential backoff reconnection: 1s, 1.5s, 2.25s ... max 10s
        const backoffMs = Math.min(1000 * Math.pow(1.5, reconnectAttemptRef.current), 10000);
        reconnectAttemptRef.current += 1;
        retryTimeoutRef.current = setTimeout(connectWebSocket, backoffMs);
      };

      ws.onerror = () => {
        ws.close();
      };
    } catch {
      setIsConnected(false);
      if (isUnmountedRef.current) return;
      if (retryTimeoutRef.current) clearTimeout(retryTimeoutRef.current);
      const backoffMs = Math.min(1000 * Math.pow(1.5, reconnectAttemptRef.current), 10000);
      reconnectAttemptRef.current += 1;
      retryTimeoutRef.current = setTimeout(connectWebSocket, backoffMs);
    }
  }, [appendChartPoint, refreshStats]);

  useEffect(() => {
    isUnmountedRef.current = false;

    // 1. Initial State Load
    fetchInitialDevices();
    fetchHealthStats();

    // 2. Periodic Health Poll
    healthIntervalRef.current = setInterval(fetchHealthStats, 3000);

    // 3. Connect Real-Time WebSocket Hub
    connectWebSocket();

    return () => {
      isUnmountedRef.current = true;
      if (retryTimeoutRef.current) clearTimeout(retryTimeoutRef.current);
      if (healthIntervalRef.current) clearInterval(healthIntervalRef.current);
      if (wsRef.current) {
        wsRef.current.close(1000, 'Component unmounted');
        wsRef.current = null;
      }
    };
  }, [fetchInitialDevices, fetchHealthStats, connectWebSocket]);

  return {
    tempSeries,
    vibSeries,
    rpmSeries,
    latestByDevice,
    incidents,
    stats,
    isConnected,
  };
}
