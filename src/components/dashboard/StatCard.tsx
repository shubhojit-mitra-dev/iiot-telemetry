interface StatCardProps {
  title: string;
  value: string | number;
  unit?: string;
  status?: 'normal' | 'warning' | 'critical' | 'info';
  subtitle?: string;
}

const statusColors: Record<string, string> = {
  normal: 'text-[#F4F5F7]',
  warning: 'text-[#FADE2A]',
  critical: 'text-[#E02F44]',
  info: 'text-[#5794F2]',
};

const borderColors: Record<string, string> = {
  normal: 'border-[#222529]',
  warning: 'border-[#FADE2A]/30',
  critical: 'border-[#E02F44]/30',
  info: 'border-[#5794F2]/30',
};

export default function StatCard({ title, value, unit, status = 'normal', subtitle }: StatCardProps) {
  return (
    <div className={`bg-[#161719] border ${borderColors[status]} rounded-[4px] p-3 h-full`}>
      <div className="text-[10px] font-medium text-[#9FA7B3] uppercase tracking-wider mb-1.5">{title}</div>
      <div className={`text-2xl md:text-3xl font-mono font-bold ${statusColors[status]} tracking-tight leading-none`}>
        {value ?? '--'}
        {unit && <span className="text-sm font-normal text-[#9FA7B3] ml-1">{unit}</span>}
      </div>
      {subtitle && <div className="text-[10px] text-[#9FA7B3] mt-1.5 font-mono">{subtitle}</div>}
    </div>
  );
}
