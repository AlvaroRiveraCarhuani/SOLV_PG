export interface FormatLine {
  id: string;
  type: 'int' | 'ints' | 'matrix' | 'edges' | 'string' | 'raw';
  min?: number | string;  // number o "=id"
  max?: number | string;
  count?: number | string;  // solo para ints/edges
  rows?: number | string;   // solo para matrix
  cols?: number | string;   // solo para matrix
  item?: { type: string; min?: number; max?: number };  // para ints/matrix
  fields?: FormatLine[];    // solo para edges
  regex?: string;           // solo para string
  max_len?: number;         // solo para string
}

export interface ExerciseContract {
  version: number;
  input: {
    lines: FormatLine[];
  };
  output?: {
    lines?: FormatLine[];
    comparator?: 'exact' | 'float' | 'unordered';
    epsilon?: number;
  };
}

export interface FormatPreset {
  id: string;
  label: string;
  description: string;
  iconName: string;
  lineCountText: string;
  previewExample: string;
  contract: ExerciseContract;
}

export const FORMAT_PRESETS: FormatPreset[] = [
  {
    id: 'single_int',
    label: 'Un entero N',
    description: 'Una sola línea con un número entero',
    iconName: 'hash',
    lineCountText: '1 línea',
    previewExample: '42',
    contract: {
      version: 1,
      input: {
        lines: [
          { id: 'n', type: 'int', min: 1, max: 1000000 }
        ]
      }
    }
  },
  {
    id: 'n_and_ints',
    label: 'N y luego N enteros',
    description: 'Línea con N, seguida de N enteros en una sola línea',
    iconName: 'list-ordered',
    lineCountText: '2 líneas',
    previewExample: '5\n10 20 -5 8 100',
    contract: {
      version: 1,
      input: {
        lines: [
          { id: 'n', type: 'int', min: 1, max: 100000 },
          { id: 'a', type: 'ints', count: '=n', item: { type: 'int', min: -1000000000, max: 1000000000 } }
        ]
      }
    }
  },
  {
    id: 'matrix_nm',
    label: 'Matriz N×M',
    description: 'Dimensiones N y M, seguidas de una matriz de N filas y M columnas',
    iconName: 'grid',
    lineCountText: '3 líneas',
    previewExample: '3 4\n1 0 1 0\n0 2 3 1\n5 6 7 8',
    contract: {
      version: 1,
      input: {
        lines: [
          { id: 'n', type: 'int', min: 1, max: 1000 },
          { id: 'm', type: 'int', min: 1, max: 1000 },
          { id: 'mat', type: 'matrix', rows: '=n', cols: '=m', item: { type: 'int', min: 0, max: 1000000 } }
        ]
      }
    }
  },
  {
    id: 'graph_edges',
    label: 'Grafo con aristas',
    description: 'N nodos, M aristas y luego lista de M pares (u, v)',
    iconName: 'network',
    lineCountText: '3 líneas',
    previewExample: '4 3\n1 2\n2 3\n3 4',
    contract: {
      version: 1,
      input: {
        lines: [
          { id: 'n', type: 'int', min: 1, max: 100000 },
          { id: 'm', type: 'int', min: 1, max: 200000 },
          { id: 'edges', type: 'edges', count: '=m' }
        ]
      }
    }
  },
  {
    id: 'single_string',
    label: 'Una cadena',
    description: 'Una línea con texto o cadena alfanumérica',
    iconName: 'type',
    lineCountText: '1 línea',
    previewExample: 'hola_mundo',
    contract: {
      version: 1,
      input: {
        lines: [
          { id: 's', type: 'string', max_len: 1000 }
        ]
      }
    }
  }
];
