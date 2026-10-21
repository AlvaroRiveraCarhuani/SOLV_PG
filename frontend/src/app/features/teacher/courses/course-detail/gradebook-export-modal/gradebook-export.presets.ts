export interface ColumnExportConfig {
  key: string;
  label: string;
  visible: boolean;
  headerColor: string;
  textColor: string;
  isGrade?: boolean;
  exerciseId?: string;
}

export interface PalettePreset {
  id: string;
  name: string;
  infoColor: string;
  labColor: string;
  avgColor: string;
}

export const PALETTE_PRESETS: PalettePreset[] = [
  {
    id: 'academic',
    name: 'Académico (Predeterminado)',
    infoColor: '#334155', // Slate
    labColor: '#2563EB',  // Azul Real
    avgColor: '#15803D'   // Verde Esmeralda
  },
  {
    id: 'ocean',
    name: 'Océano Índigo',
    infoColor: '#1E293B',
    labColor: '#0284C7',
    avgColor: '#0D9488'
  },
  {
    id: 'emerald',
    name: 'Esmeralda & Menta',
    infoColor: '#134E4A',
    labColor: '#059669',
    avgColor: '#16A34A'
  },
  {
    id: 'graphite',
    name: 'Grafito Clásico',
    infoColor: '#1F2937',
    labColor: '#4B5563',
    avgColor: '#111827'
  },
  {
    id: 'amber',
    name: 'Cálido Ámbar',
    infoColor: '#78350F',
    labColor: '#D97706',
    avgColor: '#15803D'
  },
  {
    id: 'royal',
    name: 'Púrpura Real',
    infoColor: '#4C1D95',
    labColor: '#7C3AED',
    avgColor: '#0D9488'
  }
];

export const COLOR_SWATCHES: { label: string; hex: string; textHex: string }[] = [
  { label: 'Azul', hex: '#2563EB', textHex: '#FFFFFF' },
  { label: 'Índigo', hex: '#4F46E5', textHex: '#FFFFFF' },
  { label: 'Celeste', hex: '#0284C7', textHex: '#FFFFFF' },
  { label: 'Teal', hex: '#0D9488', textHex: '#FFFFFF' },
  { label: 'Verde', hex: '#15803D', textHex: '#FFFFFF' },
  { label: 'Ámbar', hex: '#D97706', textHex: '#FFFFFF' },
  { label: 'Pizarra', hex: '#334155', textHex: '#FFFFFF' },
  { label: 'Grafito', hex: '#1E293B', textHex: '#FFFFFF' },
  { label: 'Violeta', hex: '#7C3AED', textHex: '#FFFFFF' }
];
