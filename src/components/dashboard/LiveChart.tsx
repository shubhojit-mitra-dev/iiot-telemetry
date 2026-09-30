import { LineChart, Line, XAxis, YAxis, Tooltip, ResponsiveContainer, CartesianGrid } from 'recharts';
import { TelemetryPayload } from '../../hooks/useTelemetryStream';

interface LiveChartProps {
  data: TelemetryPayload[];
}

export default function LiveChart({ data }: LiveChartProps) {
  if (!data || data.length === 0) {
    return <div className="flex h-full items-center justify-center text-gray-500">Waiting for telemetry...</div>;
  }

  const formatTime = (tickItem: number) => {
    return new Date(tickItem).toLocaleTimeString([], { hour12: false, second: '2-digit' });
  };

  return (
    <ResponsiveContainer width="100%" height="100%">
      <LineChart data={data} margin={{ top: 5, right: 20, left: 10, bottom: 5 }}>
        <CartesianGrid strokeDasharray="3 3" stroke="#374151" vertical={false} />
        <XAxis 
          dataKey="timestamp" 
          tickFormatter={formatTime} 
          stroke="#9CA3AF" 
          tick={{ fill: '#9CA3AF', fontSize: 12 }} 
        />
        <YAxis 
          yAxisId="left" 
          stroke="#9CA3AF" 
          tick={{ fill: '#9CA3AF', fontSize: 12 }}
          domain={[60, 140]}
        />
        <YAxis 
          yAxisId="right" 
          orientation="right" 
          stroke="#9CA3AF" 
          tick={{ fill: '#9CA3AF', fontSize: 12 }}
          domain={[0, 20]}
        />
        <Tooltip 
          contentStyle={{ backgroundColor: '#111827', borderColor: '#374151', color: '#F3F4F6' }}
          itemStyle={{ color: '#F3F4F6' }}
          labelFormatter={formatTime}
        />
        <Line 
          yAxisId="left" 
          type="monotone" 
          dataKey="temperature" 
          stroke="#EF4444" 
          strokeWidth={2}
          dot={false}
          isAnimationActive={false} 
        />
        <Line 
          yAxisId="right" 
          type="monotone" 
          dataKey="vibration" 
          stroke="#3B82F6" 
          strokeWidth={2}
          dot={false}
          isAnimationActive={false} 
        />
      </LineChart>
    </ResponsiveContainer>
  );
}
