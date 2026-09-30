import { useState } from 'react';
import { useTelemetryStream, DEVICES } from './hooks/useTelemetryStream';
import StatCard from './components/dashboard/StatCard';
import LiveChart from './components/dashboard/LiveChart';
import DeviceTable from './components/dashboard/DeviceTable';
import IncidentLog from './components/dashboard/IncidentLog';
import FleetMatrix from './components/dashboard/FleetMatrix';

type Tab = 'overview' | 'fleet' | 'incidents';

function App() {
  const [activeTab, setActiveTab] = useState<Tab>('overview');
  const { tempSeries, vibSeries, rpmSeries, latestByDevice, incidents, stats, isConnected } = useTelemetryStream();

  return (
    <div className="min-h-screen bg-[#0B0C10] antialiased">
      {/* ─── Header ─── */}
      <header className="border-b border-[#222529] bg-[#161719] px-4 py-2 sticky top-0 z-50">
        <div className="flex items-center justify-between max-w-[1920px] mx-auto">
          <div className="flex items-center gap-3">
            <div className="w-7 h-7 rounded-[4px] bg-[#5794F2]/10 flex items-center justify-center">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="#5794F2" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <path d="M22 12h-4l-3 9L9 3l-3 9H2" />
              </svg>
            </div>
            <div>
              <h1 className="text-sm font-bold text-[#F4F5F7] tracking-tight leading-none">FactoryPulse</h1>
              <p className="text-[10px] text-[#9FA7B3] leading-none mt-0.5">Real-Time IIoT Monitoring · Go + Redis + S3 Medallion Lakehouse</p>
            </div>
          </div>

          <nav className="flex items-center gap-0.5">
            {(['overview', 'fleet', 'incidents'] as Tab[]).map(tab => (
              <button
                key={tab}
                onClick={() => setActiveTab(tab)}
                className={`px-3 py-1.5 text-xs font-medium rounded-[4px] transition-colors duration-150 ${
                  activeTab === tab
                    ? 'bg-[#5794F2]/10 text-[#5794F2]'
                    : 'text-[#9FA7B3] hover:text-[#F4F5F7] hover:bg-[#222529]'
                }`}
              >
                {tab.charAt(0).toUpperCase() + tab.slice(1)}
                {tab === 'incidents' && incidents.length > 0 && (
                  <span className="ml-1.5 px-1 py-0.5 rounded-[2px] bg-[#E02F44]/10 text-[#E02F44] text-[10px] font-bold">
                    {incidents.length}
                  </span>
                )}
              </button>
            ))}
          </nav>

          <div className="flex items-center gap-4">
            <div className="flex items-center gap-1.5 text-xs">
              <span className="relative flex h-2 w-2">
                <span className={`absolute inline-flex h-full w-full animate-ping rounded-full ${isConnected ? 'bg-[#56A64B]' : 'bg-[#FADE2A]'} opacity-75`}></span>
                <span className={`relative inline-flex h-2 w-2 rounded-full ${isConnected ? 'bg-[#56A64B]' : 'bg-[#FADE2A]'}`}></span>
              </span>
              <span className={`font-mono text-[10px] font-bold tracking-wider ${isConnected ? 'text-[#56A64B]' : 'text-[#FADE2A]'}`}>
                {isConnected ? 'LIVE WS' : 'CONNECTING'}
              </span>
            </div>
            <span className="text-[10px] font-mono text-[#9FA7B3]">{DEVICES.length} nodes · Go Pipeline</span>
          </div>
        </div>
      </header>

      {/* ─── Main Content ─── */}
      <main className="p-3 max-w-[1920px] mx-auto">

        {/* ═══ OVERVIEW TAB ═══ */}
        {activeTab === 'overview' && (
          <div className="space-y-2">

            {/* Row 1: Stat Cards */}
            <div className="grid grid-cols-12 gap-2">
              <div className="col-span-12 sm:col-span-6 lg:col-span-2">
                <StatCard
                  title="Active Nodes"
                  value={`${stats.activeDevices}/${stats.totalDevices}`}
                  status="info"
                  subtitle={stats.activeDevices === stats.totalDevices ? 'All fleet online' : `${stats.activeDevices} transmitting`}
                />
              </div>
              <div className="col-span-12 sm:col-span-6 lg:col-span-2">
                <StatCard
                  title="Messages Ingested"
                  value={stats.messagesIngested.toLocaleString()}
                  status="normal"
                  subtitle={stats.queueDepth > 0 ? `Queue Depth: ${stats.queueDepth}` : `${stats.activeClients} WS Client${stats.activeClients !== 1 ? 's' : ''}`}
                />
              </div>
              <div className="col-span-12 sm:col-span-6 lg:col-span-2">
                <StatCard
                  title="Avg Temperature"
                  value={stats.avgTemperature.toFixed(1)}
                  unit="°C"
                  status={stats.avgTemperature > 100 ? 'warning' : 'normal'}
                />
              </div>
              <div className="col-span-12 sm:col-span-6 lg:col-span-2">
                <StatCard
                  title="Peak Temperature"
                  value={stats.peakTemperature.toFixed(1)}
                  unit="°C"
                  status={stats.peakTemperature > 120 ? 'critical' : stats.peakTemperature > 100 ? 'warning' : 'normal'}
                />
              </div>
              <div className="col-span-12 sm:col-span-6 lg:col-span-2">
                <StatCard
                  title="Avg Vibration"
                  value={stats.avgVibration.toFixed(1)}
                  unit="mm/s"
                  status="normal"
                />
              </div>
              <div className="col-span-12 sm:col-span-6 lg:col-span-2">
                <StatCard
                  title="Anomalies Detected"
                  value={stats.anomalyCount}
                  status={stats.anomalyCount > 0 ? 'critical' : 'normal'}
                  subtitle="Server detected"
                />
              </div>
            </div>

            {/* Row 2: Temperature Chart (Full Width) */}
            <LiveChart
              data={tempSeries}
              title="▾ Temperature Monitoring — All Devices"
              unit="°C"
              thresholdLine={120}
            />

            {/* Row 3: Vibration + RPM Charts */}
            <div className="grid grid-cols-12 gap-2">
              <div className="col-span-12 lg:col-span-6">
                <LiveChart
                  data={vibSeries}
                  title="▾ Vibration Levels"
                  unit="mm/s"
                />
              </div>
              <div className="col-span-12 lg:col-span-6">
                <LiveChart
                  data={rpmSeries}
                  title="▾ RPM Monitor"
                  unit="RPM"
                />
              </div>
            </div>

            {/* Row 4: Device Table + Incident Log */}
            <div className="grid grid-cols-12 gap-2">
              <div className="col-span-12 lg:col-span-8">
                <DeviceTable devices={latestByDevice} />
              </div>
              <div className="col-span-12 lg:col-span-4">
                <IncidentLog incidents={incidents} compact />
              </div>
            </div>
          </div>
        )}

        {/* ═══ FLEET TAB ═══ */}
        {activeTab === 'fleet' && (
          <div className="space-y-2">
            <div className="text-xs font-medium text-[#9FA7B3] uppercase tracking-wider mb-1">▾ Edge Node Fleet — Detailed View</div>
            <FleetMatrix devices={latestByDevice} />
          </div>
        )}

        {/* ═══ INCIDENTS TAB ═══ */}
        {activeTab === 'incidents' && (
          <IncidentLog incidents={incidents} />
        )}
      </main>
    </div>
  );
}

export default App;
