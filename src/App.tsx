import { Activity, AlertTriangle, Cpu, Server } from "lucide-react";
import { useTelemetryStream } from "./hooks/useTelemetryStream";
import LiveChart from "./components/dashboard/LiveChart";
import FleetMatrix from "./components/dashboard/FleetMatrix";

function App() {
  const { data, latestByDevice, anomaly, clearAnomaly } = useTelemetryStream(true);

  return (
    <div className="min-h-screen bg-gray-950 p-6">
      <header className="mb-8 flex items-center justify-between border-b border-gray-800 pb-4">
        <div className="flex items-center gap-3">
          <Cpu className="h-8 w-8 text-blue-500" />
          <h1 className="text-2xl font-bold tracking-tight text-gray-100">
            IIoT Telemetry Command Center
          </h1>
        </div>
        <div className="flex items-center gap-4 text-sm text-gray-400">
          <div className="flex items-center gap-2">
            <span className="relative flex h-3 w-3">
              <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-green-400 opacity-75"></span>
              <span className="relative inline-flex h-3 w-3 rounded-full bg-green-500"></span>
            </span>
            System Live
          </div>
        </div>
      </header>

      {anomaly && (
        <div className="mb-6 flex items-center justify-between rounded-lg border border-red-900 bg-red-950/50 p-4 text-red-200 shadow-sm animate-in fade-in slide-in-from-top-4">
          <div className="flex items-center gap-3">
            <AlertTriangle className="h-6 w-6 text-red-500" />
            <div>
              <p className="font-semibold text-red-400">Critical Anomaly Detected: {anomaly.device_id}</p>
              <p className="text-sm opacity-90">
                AI Diagnostic: Potential bearing failure detected due to extreme temperature ({anomaly.temperature.toFixed(1)}°C).
              </p>
            </div>
          </div>
          <button 
            onClick={clearAnomaly}
            className="rounded bg-red-900/50 px-3 py-1 text-sm hover:bg-red-900 transition-colors"
          >
            Acknowledge
          </button>
        </div>
      )}

      <div className="grid gap-6 lg:grid-cols-3">
        <div className="lg:col-span-2 space-y-6">
          <div className="rounded-xl border border-gray-800 bg-gray-900/50 p-6 shadow-sm backdrop-blur-sm">
            <div className="mb-4 flex items-center gap-2">
              <Activity className="h-5 w-5 text-blue-400" />
              <h2 className="text-lg font-semibold">Live Telemetry Stream</h2>
            </div>
            <div className="h-[400px] w-full">
              <LiveChart data={data} />
            </div>
          </div>
        </div>

        <div>
          <div className="rounded-xl border border-gray-800 bg-gray-900/50 p-6 shadow-sm backdrop-blur-sm">
            <div className="mb-4 flex items-center gap-2">
              <Server className="h-5 w-5 text-purple-400" />
              <h2 className="text-lg font-semibold">Fleet Status Matrix</h2>
            </div>
            <FleetMatrix devices={latestByDevice} />
          </div>
        </div>
      </div>
    </div>
  );
}

export default App;
