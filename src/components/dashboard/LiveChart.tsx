import { AreaChart, Area, XAxis, YAxis, Tooltip, ResponsiveContainer, CartesianGrid, ReferenceLine } from 'recharts';
import { ChartDataPoint, DEVICES, DEVICE_COLORS } from '../../hooks/useTelemetryStream';

interface TelemetryChartProps {
  data: ChartDataPoint[];
  title: string;
  unit: string;
  thresholdLine?: number;
}

function CustomTooltip({ active, payload, label }: { active?: boolean; payload?: { dataKey: string; value: number; color: string }[]; label?: number }) {
  if (!active || !payload || !label) return null;
  return (
    <div className="bg-[#0B0C10] border border-[#222529] rounded-[4px] p-2 text-[10px] font-mono max-h-48 overflow-y-auto shadow-xl">
      <div className="text-[#9FA7B3] mb-1 pb-1 border-b border-[#222529]">
        {new Date(label).toLocaleTimeString([], { hour12: false })}
      </div>
      {payload.slice(0, 5).map((entry) => (
        <div key={entry.dataKey} className="flex items-center gap-2 py-0.5">
          <span className="w-2 h-[2px] rounded shrink-0" style={{ backgroundColor: entry.color }} />
          <span className="text-[#9FA7B3] truncate">{entry.dataKey}</span>
          <span className="text-[#F4F5F7] ml-auto">{entry.value?.toFixed(1)}</span>
        </div>
      ))}
      {payload.length > 5 && <div className="text-[#9FA7B3] pt-1">+{payload.length - 5} more</div>}
    </div>
  );
}

export default function LiveChart({ data, title, unit, thresholdLine }: TelemetryChartProps) {
  if (!data.length) {
    return (
      <div className="bg-[#161719] border border-[#222529] rounded-[4px] p-3 h-[300px] flex items-center justify-center">
        <div className="animate-pulse text-[#9FA7B3] text-xs font-mono">Initializing stream...</div>
      </div>
    );
  }

  const formatTime = (ts: number) => new Date(ts).toLocaleTimeString([], { hour12: false, minute: '2-digit', second: '2-digit' });

  return (
    <div className="bg-[#161719] border border-[#222529] rounded-[4px] p-3">
      <div className="flex items-center justify-between mb-3">
        <span className="text-xs font-medium text-[#9FA7B3] uppercase tracking-wider">{title}</span>
        <span className="text-[10px] font-mono text-[#9FA7B3]">{unit}</span>
      </div>
      <div className="h-[220px]">
        <ResponsiveContainer width="100%" height="100%">
          <AreaChart data={data} margin={{ top: 5, right: 10, left: -10, bottom: 5 }}>
            <defs>
              {DEVICES.map(d => (
                <linearGradient key={d.id} id={`grad-${d.id}-${title.replace(/\s/g, '')}`} x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor={DEVICE_COLORS[d.id]} stopOpacity={0.12} />
                  <stop offset="95%" stopColor={DEVICE_COLORS[d.id]} stopOpacity={0} />
                </linearGradient>
              ))}
            </defs>
            <CartesianGrid strokeDasharray="3 3" stroke="#222529" vertical={false} />
            <XAxis dataKey="timestamp" tickFormatter={formatTime} stroke="#333" tick={{ fill: '#9FA7B3', fontSize: 10 }} axisLine={{ stroke: '#222529' }} />
            <YAxis stroke="#333" tick={{ fill: '#9FA7B3', fontSize: 10 }} axisLine={{ stroke: '#222529' }} domain={['auto', 'auto']} />
            <Tooltip content={<CustomTooltip />} />
            {thresholdLine !== undefined && (
              <ReferenceLine y={thresholdLine} stroke="#E02F44" strokeDasharray="5 5" strokeOpacity={0.5} label={{ value: `${thresholdLine}${unit}`, fill: '#E02F44', fontSize: 10, position: 'right' }} />
            )}
            {DEVICES.map(d => (
              <Area
                key={d.id}
                type="monotone"
                dataKey={d.id}
                stroke={DEVICE_COLORS[d.id]}
                fill={`url(#grad-${d.id}-${title.replace(/\s/g, '')})`}
                strokeWidth={1.5}
                dot={false}
                isAnimationActive={false}
                connectNulls
              />
            ))}
          </AreaChart>
        </ResponsiveContainer>
      </div>
      <div className="flex flex-wrap gap-x-3 gap-y-1 mt-2 px-1">
        {DEVICES.map(d => (
          <div key={d.id} className="flex items-center gap-1 text-[10px] font-mono text-[#9FA7B3]">
            <span className="w-2 h-[2px] rounded shrink-0" style={{ backgroundColor: DEVICE_COLORS[d.id] }} />
            {d.id}
          </div>
        ))}
      </div>
    </div>
  );
}
