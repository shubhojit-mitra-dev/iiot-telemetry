import { DEVICES, DEVICE_COLORS, TelemetryPayload } from '../../hooks/useTelemetryStream';

interface DeviceTableProps {
  devices: Record<string, TelemetryPayload>;
}

export default function DeviceTable({ devices }: DeviceTableProps) {
  return (
    <div className="bg-[#161719] border border-[#222529] rounded-[4px] p-3">
      <div className="text-xs font-medium text-[#9FA7B3] uppercase tracking-wider mb-3">▾ Device Fleet Status</div>
      <div className="overflow-x-auto">
        <table className="w-full text-left font-mono text-xs border-collapse">
          <thead>
            <tr className="border-b border-[#222529]">
              <th className="py-2 pr-3 text-[10px] text-[#9FA7B3] font-medium uppercase tracking-wider">Device</th>
              <th className="py-2 pr-3 text-[10px] text-[#9FA7B3] font-medium uppercase tracking-wider">Machine</th>
              <th className="py-2 pr-3 text-[10px] text-[#9FA7B3] font-medium uppercase tracking-wider">Type</th>
              <th className="py-2 pr-3 text-[10px] text-[#9FA7B3] font-medium uppercase tracking-wider">Status</th>
              <th className="py-2 pr-3 text-[10px] text-[#9FA7B3] font-medium uppercase tracking-wider text-right">Temp</th>
              <th className="py-2 pr-3 text-[10px] text-[#9FA7B3] font-medium uppercase tracking-wider text-right">Vibration</th>
              <th className="py-2 text-[10px] text-[#9FA7B3] font-medium uppercase tracking-wider text-right">RPM</th>
            </tr>
          </thead>
          <tbody>
            {DEVICES.map((device, idx) => {
              const data = devices[device.id];
              const isCritical = data && data.temperature > 120;
              const isWarning = data && data.temperature > 100;
              return (
                <tr
                  key={device.id}
                  className={`border-b border-[#222529]/50 hover:bg-[#222529] transition-colors duration-150 ${
                    idx % 2 === 0 ? 'bg-[#161719]' : 'bg-[#1D2024]'
                  } ${isCritical ? '!bg-[#E02F44]/5' : ''}`}
                >
                  <td className="py-2 pr-3">
                    <div className="flex items-center gap-2">
                      <span className="w-2 h-2 rounded-full shrink-0" style={{ backgroundColor: DEVICE_COLORS[device.id] }} />
                      <span className="text-[#F4F5F7]">{device.id}</span>
                    </div>
                  </td>
                  <td className="py-2 pr-3 text-[#9FA7B3]">{device.name}</td>
                  <td className="py-2 pr-3 text-[#9FA7B3]">{device.type}</td>
                  <td className="py-2 pr-3">
                    {isCritical ? (
                      <span className="px-1.5 py-0.5 rounded-[2px] bg-[#E02F44]/10 text-[#E02F44] font-bold text-[10px]">CRITICAL</span>
                    ) : isWarning ? (
                      <span className="px-1.5 py-0.5 rounded-[2px] bg-[#FADE2A]/10 text-[#FADE2A] font-bold text-[10px]">WARNING</span>
                    ) : (
                      <span className="px-1.5 py-0.5 rounded-[2px] bg-[#56A64B]/10 text-[#56A64B] font-bold text-[10px]">ONLINE</span>
                    )}
                  </td>
                  <td className={`py-2 pr-3 text-right ${isCritical ? 'text-[#E02F44] font-bold' : isWarning ? 'text-[#FADE2A]' : 'text-[#F4F5F7]'}`}>
                    {data?.temperature.toFixed(1) ?? '--'}°C
                  </td>
                  <td className="py-2 pr-3 text-right text-[#F4F5F7]">{data?.vibration.toFixed(1) ?? '--'} mm/s</td>
                  <td className="py-2 text-right text-[#F4F5F7]">{data?.rpm.toFixed(0) ?? '--'}</td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
}
