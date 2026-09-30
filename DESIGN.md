# UI Design Specification: Grafana-Style Industrial Dashboard

## 1. Design Philosophy
Enterprise-grade, high-density engineering dashboard. The aesthetic mimics **Grafana Enterprise** and **Datadog** — optimized for maximum readability, technical density, operational clarity, and cognitive ease. Dark mode only. No light theme.

## 2. Color Palette (Exact Hex Values)
- **Primary Background:** `#0B0C10` (Deep obsidian)
- **Panel/Card Background:** `#161719` (Slightly lighter contrast surface)
- **Borders/Dividers:** `#222529` (Subtle dark gray)
- **Primary Text:** `#F4F5F7` (High-contrast white)
- **Secondary/Muted Text:** `#9FA7B3` (Cool gray for labels)
- **Success/Healthy:** `#56A64B`
- **Warning:** `#FADE2A`
- **Critical/Error:** `#E02F44`
- **Information/Accent:** `#5794F2`

## 3. Typography & Grid
- Font: `font-mono` for all data values, `font-sans antialiased tracking-tight` for labels.
- Grid: Strict 12-column grid (`grid grid-cols-12 gap-2`).
- Border Radii: Uniform `rounded-[4px]` everywhere.
- Density: High. Use `p-2 md:p-3`, minimize whitespace.

## 4. Layout (Top-Down)
```
[Header] Logo + Tabs (Overview | Fleet | Incidents) + LIVE indicator
[Row 1]  6x Stat Cards (Big bold numbers: Active Nodes, Ingest Rate, Avg Temp, Peak Temp, Avg Vibration, Anomalies)
[Row 2]  Full-width Temperature Area Chart (all 10 device lines, 120°C threshold reference line)
[Row 3]  Vibration Chart (col-span-6) + RPM Chart (col-span-6)
[Row 4]  Device Fleet Table (col-span-8) + Incident Log (col-span-4)
```

## 5. Component Rules
- Stat Cards: Title (10px uppercase gray), Value (2xl mono bold), Subtitle (xs mono gray)
- Charts: Area charts with gradient fills, no animation, dark tooltips, device color legends below
- Tables: Zebra striping, monospace, right-align numerics, status pills (ONLINE/CRITICAL)
- Incidents: Left-border colored severity indicator, AI diagnostic text in blue italic
- Hover states: `hover:bg-[#222529] transition-colors duration-150`
- Data fallbacks: Render `"--"` for null/undefined values
