# Phase 2: React Frontend & Mock Demo (Vercel Deployment)

**Target Audience:** Implementing Agents / Frontend Engineers
**Objective:** Rapidly construct the React frontend using a unified repository structure. The initial version will use a Mock Data Generator to simulate the WebSocket stream so it can be deployed to Vercel immediately for the recruiter. We will seamlessly swap the mock generator with the real Go WebSocket backend later.

## 1. Unified Repository Architecture
We are abandoning the standard `backend/` and `frontend/` split. Both Go and Node.js toolchains will share the root directory to match modern, flat mono-repo patterns (similar to the Fider architecture).

```text
iiot-telemetry/
├── app/                    # Go Backend (Ingestion, WebSockets, Redis)
├── cmd/                    # Go Entrypoints
├── components/             # React UI Components
│   ├── dashboard/          # Dashboard specific widgets (Charts, Matrix)
│   ├── layout/             # Navigation, Shell, Alerts
│   └── ui/                 # Reusable raw components (Buttons, Cards)
├── hooks/                  # React Custom Hooks (e.g., useTelemetryStream)
├── public/                 # Static Assets (Favicon, etc)
├── styles/                 # Global CSS and Tailwind directives
├── docs/                   # Documentation (You are here)
├── main.go                 # Go Application Root
├── go.mod                  # Go Dependencies
├── package.json            # Node/Frontend Dependencies
├── vite.config.ts          # Vite Configuration
├── tailwind.config.js      # Tailwind Configuration
└── tsconfig.json           # TypeScript Configuration
```

## 2. Technology Stack
- **Framework:** React 18 (Bootstrapped via Vite)
- **Language:** TypeScript
- **Styling:** TailwindCSS (Dark mode optimized for Industrial feel)
- **Charting:** Recharts (High performance canvas/svg rendering)
- **Icons:** Lucide React

## 3. The "Mock-First" Strategy (Immediate Vercel Deployment)
To meet the immediate deadline for the recruiter, the frontend must appear 100% functional without the Go backend running.

- **`useTelemetryStream.ts` Hook:** This hook will be designed with a toggle.
  - *Mode A (Mock):* Uses `setInterval` every 50ms to push fake telemetry payloads (DeviceID, Temp, Vibration, RPM) into the state array. Occasional temperature spikes > 120°C will be artificially injected.
  - *Mode B (Real):* Connects to the future `ws://api.domain.com/ws/telemetry`.
- **Initial Vercel Deploy:** We will ship with Mode A active.

## 4. Component Breakdown & Implementation Details

### A. Telemetry Context & State Management
- Do not use Redux (too heavy). Lifting state to a parent `Dashboard` component is sufficient for a 12-hour build.
- **Buffer Limits:** The charting array MUST be capped at a maximum length (e.g., `50` data points). Use a sliding window approach (`slice(-50)`) when pushing new data to prevent immediate memory leaks and browser crashes.

### B. Fleet Status Matrix (`components/dashboard/FleetMatrix.tsx`)
- A CSS Grid displaying 10 cards (Device-01 to Device-10).
- Normal State: Green pulsing indicator.
- Critical State: If the latest payload for a specific device has `Temperature > 120`, the card must turn Red and flash a "CRITICAL" warning.

### C. Live Telemetry Chart (`components/dashboard/LiveChart.tsx`)
- Utilize `Recharts` (`LineChart`, `XAxis`, `YAxis`, `Tooltip`, `Line`).
- Plot `Temperature` (Red line) and `Vibration` (Blue line) over the last 50 ticks.
- Disable heavy animations (`isAnimationActive={false}`) to ensure the browser doesn't choke on 50ms interval updates.

### D. Incident Alert Banner (`components/layout/IncidentAlert.tsx`)
- A fixed banner at the top of the screen.
- Triggers when an anomaly is detected.
- Includes the mock AI Diagnostic text (e.g., *"AI Diagnostic: Potential bearing failure detected due to severe vibration and heat."*)

## 5. Execution Steps for the Agent
1. Initialize Vite project in a temporary folder and move its config files to the repository root.
2. Install TailwindCSS, Recharts, and Lucide React.
3. Build the Mock Generator hook (`useTelemetryStream.ts`).
4. Build the UI components (Layout, Matrix, Chart).
5. Assemble `App.tsx` and verify the mock data flows smoothly.
6. Commit the code (following the 2-file atomic limit).
7. Wait for the user to deploy to Vercel.
