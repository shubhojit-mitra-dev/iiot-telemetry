import { DEVICES, DEVICE_COLORS, TelemetryPayload } from '../../hooks/useTelemetryStream';

interface FleetMatrixProps {
  devices: Record<string, TelemetryPayload>;
}

export default function FleetMatrix({ devices }: FleetMatrixProps) {
  return (
    <div className="grid grid-cols-12 gap-2">
      {DEVICES.map(device => {
        const data = devices[device.id];
        const isCritical = data && data.temperature > 120;
        const isWarning = data && data.temperature > 100;
        const statusColor = isCritical ? '#E02F44' : isWarning ? '#FADE2A' : '#56A64B';
        const statusLabel = isCritical ? 'CRITICAL' : isWarning ? 'WARNING' : 'ONLINE';

        return (
          <div
            key={device.id}
            className={`col-span-12 sm:col-span-6 lg:col-span-4 xl:col-span-3 bg-[#161719] border rounded-[4px] p-4 hover:bg-[#1D2024] transition-colors duration-150 ${
              isCritical ? 'border-[#E02F44]/40' : 'border-[#222529]'
            }`}
          >
            <div className="flex items-center justify-between mb-3">
              <div className="flex items-center gap-2">
                <span className="w-2.5 h-2.5 rounded-full shrink-0" style={{ backgroundColor: DEVICE_COLORS[device.id] }} />
                <span className="text-sm font-bold text-[#F4F5F7] tracking-tight">{device.id}</span>
              </div>
              <span className={`px-1.5 py-0.5 rounded-[2px] text-[10px] font-bold`} style={{ backgroundColor: `${statusColor}15`, color: statusColor }}>
                {statusLabel}
              </span>
            </div>
            <div className="text-xs text-[#9FA7B3] mb-3">{device.name} — {device.type}</div>
            <div className="grid grid-cols-3 gap-3">
              <div>
                <div className="text-[10px] text-[#9FA7B3] uppercase tracking-wider mb-0.5">Temp</div>
                <div className={`text-lg font-mono font-bold tracking-tight ${isCritical ? 'text-[#E02F44]' : isWarning ? 'text-[#FADE2A]' : 'text-[#F4F5F7]'}`}>
                  {data?.temperature.toFixed(1) ?? '--'}
                  <span className="text-[10px] text-[#9FA7B3] font-normal ml-0.5">°C</span>
                </div>
              </div>
              <div>
                <div className="text-[10px] text-[#9FA7B3] uppercase tracking-wider mb-0.5">Vibration</div>
                <div className="text-lg font-mono font-bold text-[#F4F5F7] tracking-tight">
                  {data?.vibration.toFixed(1) ?? '--'}
                  <span className="text-[10px] text-[#9FA7B3] font-normal ml-0.5">mm/s</span>
                </div>
              </div>
              <div>
                <div className="text-[10px] text-[#9FA7B3] uppercase tracking-wider mb-0.5">RPM</div>
                <div className="text-lg font-mono font-bold text-[#F4F5F7] tracking-tight">
                  {data?.rpm.toFixed(0) ?? '--'}
                </div>
              </div>
            </div>
          </div>
        );
      })}
    </div>
  );
}
