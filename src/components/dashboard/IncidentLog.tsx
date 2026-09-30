import { Incident } from '../../hooks/useTelemetryStream';

interface IncidentLogProps {
  incidents: Incident[];
  compact?: boolean;
}

export default function IncidentLog({ incidents, compact = false }: IncidentLogProps) {
  return (
    <div className="bg-[#161719] border border-[#222529] rounded-[4px] p-3 h-full">
      <div className="flex items-center justify-between mb-3">
        <span className="text-xs font-medium text-[#9FA7B3] uppercase tracking-wider">▾ Anomaly Incident Log</span>
        {incidents.length > 0 && (
          <span className="px-1.5 py-0.5 rounded-[2px] bg-[#E02F44]/10 text-[#E02F44] font-bold text-[10px]">
            {incidents.length}
          </span>
        )}
      </div>
      <div className={`space-y-2 overflow-y-auto ${compact ? 'max-h-[320px]' : 'max-h-[600px]'}`}>
        {incidents.length === 0 && (
          <div className="text-[10px] text-[#9FA7B3] font-mono py-4 text-center">No anomalies recorded</div>
        )}
        {incidents.map(inc => (
          <div
            key={inc.id}
            className={`border-l-2 ${inc.severity === 'critical' ? 'border-[#E02F44]' : 'border-[#FADE2A]'} bg-[#0B0C10] rounded-r-[2px] p-2 hover:bg-[#222529]/30 transition-colors duration-150`}
          >
            <div className="flex items-center justify-between mb-0.5">
              <span className="text-[10px] font-mono text-[#9FA7B3]">
                {new Date(inc.timestamp).toLocaleTimeString([], { hour12: false })}
              </span>
              <span className={`px-1.5 py-0.5 rounded-[2px] text-[10px] font-bold ${
                inc.severity === 'critical' ? 'bg-[#E02F44]/10 text-[#E02F44]' : 'bg-[#FADE2A]/10 text-[#FADE2A]'
              }`}>
                {inc.severity.toUpperCase()}
              </span>
            </div>
            <div className="text-xs text-[#F4F5F7] font-semibold">{inc.device_id} — {inc.device_name}</div>
            <div className="text-[10px] text-[#9FA7B3] mt-0.5 font-mono">
              Temp: <span className="text-[#E02F44]">{inc.temperature.toFixed(1)}°C</span> | Vib: {inc.vibration.toFixed(1)} mm/s
            </div>
            <div className="text-[10px] text-[#5794F2] mt-1 leading-relaxed">
              <span className="text-[#5794F2]/60">AI → </span>{inc.message}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
