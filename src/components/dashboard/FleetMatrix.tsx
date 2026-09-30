import { TelemetryPayload } from '../../hooks/useTelemetryStream';

interface FleetMatrixProps {
  devices: Record<string, TelemetryPayload>;
}

export default function FleetMatrix({ devices }: FleetMatrixProps) {
  const deviceList = Array.from({ length: 5 }, (_, i) => `Device-0${i + 1}`);

  return (
    <div className="grid grid-cols-2 gap-3">
      {deviceList.map((id) => {
        const data = devices[id];
        const isCritical = data && data.temperature > 120;

        return (
          <div 
            key={id}
            className={`flex flex-col rounded-lg border p-3 transition-colors ${
              isCritical 
                ? 'border-red-500 bg-red-950/40' 
                : data 
                  ? 'border-gray-700 bg-gray-800/40' 
                  : 'border-gray-800 bg-gray-900/40'
            }`}
          >
            <div className="flex items-center justify-between mb-2">
              <span className={`text-sm font-semibold ${isCritical ? 'text-red-400' : 'text-gray-300'}`}>
                {id}
              </span>
              {data && (
                <span className={`relative flex h-2 w-2`}>
                  {isCritical && <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-red-400 opacity-75"></span>}
                  <span className={`relative inline-flex h-2 w-2 rounded-full ${isCritical ? 'bg-red-500' : 'bg-green-500'}`}></span>
                </span>
              )}
            </div>
            
            {data ? (
              <div className="text-xs text-gray-400 space-y-1">
                <div className="flex justify-between">
                  <span>Temp:</span>
                  <span className={isCritical ? 'text-red-300 font-bold' : 'text-gray-200'}>{data.temperature.toFixed(1)}°C</span>
                </div>
                <div className="flex justify-between">
                  <span>Vib:</span>
                  <span className="text-gray-200">{data.vibration.toFixed(1)}mm/s</span>
                </div>
              </div>
            ) : (
              <div className="text-xs text-gray-600 mt-2">Waiting...</div>
            )}
          </div>
        );
      })}
    </div>
  );
}
