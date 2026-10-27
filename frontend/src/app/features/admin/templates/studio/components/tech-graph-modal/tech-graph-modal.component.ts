import { ChangeDetectionStrategy, Component, computed, input, output, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { LucideAlertTriangle, LucideCheckCircle2, LucideNetwork } from '@lucide/angular';
import { TemplateComponent } from '@features/admin/models/admin-templates.models';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { ModalShellComponent } from '@shared/components/modal-shell/modal-shell.component';
import { TechLogoComponent } from '../../../components/tech-logo/tech-logo.component';
import { computeCompatibleAddonsForSelection } from '../../template-studio.utils';

export type TechGraphRole = 'lenguaje' | 'web' | 'complemento' | 'database';

export interface TechGraphNode {
  key: string;
  name: string;
  slug: string;
  iconKey: string;
  role: TechGraphRole;
  col: 0 | 1 | 2;
  x: number;
  y: number;
  isActive: boolean;
  isOrphan: boolean;
  isUniversal: boolean;
  degree: number;
}

export interface TechGraphEdge {
  key: string;
  source: string;
  target: string;
  kind: 'compatible' | 'combinable';
}

export interface TechGraph {
  languages: TechGraphNode[];
  chipCounts: Record<string, number>;
  selectedLanguage: TechGraphNode | null;
  nodes: TechGraphNode[];
  edges: TechGraphEdge[];
  activeCount: number;
  inactiveCount: number;
  orphanCount: number;
  runtimeCount: number;
  webCount: number;
  databaseCount: number;
  addonCount: number;
  visibleAddons: number;
  totalAddons: number;
  hasMoreAddons: boolean;
}

const VISIBLE_ADDON_LIMIT = 12;

function familyTokensOf(comp: TemplateComponent): Set<string> {
  const tokens = new Set<string>();
  if (comp.slug) tokens.add(comp.slug.toLowerCase());
  if (comp.id) tokens.add(comp.id.toLowerCase());
  if (comp.icon_key) tokens.add(comp.icon_key.toLowerCase());
  for (const family of comp.compatible_runtimes || []) {
    if (family) tokens.add(family.toLowerCase());
  }
  return tokens;
}

function isUniversalCompat(comp: TemplateComponent): boolean {
  const compat = comp.compatible_runtimes || [];
  return compat.length === 0 || compat.some(value => ['universal', 'any'].includes(value.trim().toLowerCase()));
}

function matchesLanguage(node: TemplateComponent, lang: TemplateComponent): boolean {
  if (isUniversalCompat(node)) return true;
  const families = familyTokensOf(lang);
  return (node.compatible_runtimes || []).some(value => families.has(value.trim().toLowerCase()));
}

function bySortOrderThenName(a: TemplateComponent, b: TemplateComponent): number {
  const order = (a.sort_order ?? 0) - (b.sort_order ?? 0);
  return order || (a.name || '').localeCompare(b.name || '');
}

@Component({
  selector: 'tech-graph-modal',
  standalone: true,
  imports: [FormsModule, ModalShellComponent, TechLogoComponent, MachineDataDirective, LucideNetwork, LucideCheckCircle2, LucideAlertTriangle],
  templateUrl: './tech-graph-modal.component.html',
  styleUrl: './tech-graph-modal.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class TechGraphModalComponent {
  readonly components = input<TemplateComponent[]>([]);
  readonly dismiss = output<void>();

  readonly selectedLangKey = signal<string | null>(null);
  readonly selectedKey = signal<string | null>(null);
  readonly searchQuery = signal<string>('');
  readonly showAllAddons = signal<boolean>(false);

  readonly graph = computed(() =>
    this.buildGraph(this.components(), this.selectedLangKey(), this.searchQuery(), this.showAllAddons())
  );
  readonly languages = computed(() => this.graph().languages);
  readonly nodes = computed(() => this.graph().nodes);
  readonly edges = computed(() => this.graph().edges);
  readonly selectedLanguage = computed(() => this.graph().selectedLanguage);

  readonly selectedNode = computed<TechGraphNode | null>(() => {
    const key = this.selectedKey();
    return key ? this.nodes().find(node => node.key === key) || null : null;
  });

  readonly accessibleRelationships = computed(() => {
    const nodeMap = new Map(this.nodes().map(node => [node.key, node.name]));
    return this.edges().map(edge => {
      const relation = edge.kind === 'compatible' ? 'Compatible con' : 'Combinable con';
      return `${nodeMap.get(edge.source) ?? edge.source}: ${relation} ${nodeMap.get(edge.target) ?? edge.target}`;
    });
  });

  selectLanguage(key: string): void {
    if (this.selectedLangKey() === key) return;
    this.selectedLangKey.set(key);
    this.selectedKey.set(null);
    this.showAllAddons.set(false);
  }

  selectNode(key: string): void {
    this.selectedKey.update(current => current === key ? null : key);
  }

  clearSelection(): void {
    this.selectedKey.set(null);
  }

  setSearchQuery(value: string): void {
    this.searchQuery.set(value);
    this.showAllAddons.set(false);
  }

  roleLabel(role: TechGraphRole): string {
    switch (role) {
      case 'lenguaje': return 'Lenguaje principal';
      case 'web': return 'Interfaz web';
      case 'database': return 'Base de datos';
      default: return 'Complemento';
    }
  }

  private toNode(comp: TemplateComponent, role: TechGraphRole, degree: number): TechGraphNode {
    return {
      key: comp.id,
      name: comp.name,
      slug: comp.slug || comp.id,
      iconKey: comp.icon_key,
      role,
      col: role === 'lenguaje' ? 0 : role === 'complemento' ? 2 : 1,
      x: 0,
      y: 0,
      isActive: comp.is_active !== false,
      isOrphan: false,
      isUniversal: role === 'complemento' && isUniversalCompat(comp),
      degree
    };
  }

  private buildGraph(
    components: TemplateComponent[],
    selectedLangKey: string | null,
    query: string,
    showAllAddons: boolean
  ): TechGraph {
    const languages = components
      .filter(comp => comp.layer === 'runtime' && comp.sub_category !== 'web')
      .sort(bySortOrderThenName);
    const webs = components
      .filter(comp => comp.layer === 'runtime' && comp.sub_category === 'web')
      .sort(bySortOrderThenName);
    const databases = components
      .filter(comp => comp.layer === 'database' && comp.service_engine !== 'none' && comp.slug !== 'db-none')
      .sort(bySortOrderThenName);
    const addons = components.filter(comp => comp.layer === 'addon').sort(bySortOrderThenName);
    const orphanCount = addons.filter(addon =>
      !languages.some(language => matchesLanguage(addon, language)) &&
      !webs.some(web => matchesLanguage(addon, web))
    ).length;
    const normalizedQuery = query.trim().toLocaleLowerCase();
    const matchesQuery = (comp: TemplateComponent) =>
      !normalizedQuery || `${comp.name} ${comp.slug} ${comp.description}`.toLocaleLowerCase().includes(normalizedQuery);

    if (languages.length === 0) {
      return {
        languages: [], chipCounts: {}, selectedLanguage: null, nodes: [], edges: [],
        activeCount: components.filter(comp => comp.is_active !== false).length,
        inactiveCount: components.filter(comp => comp.is_active === false).length,
        orphanCount, runtimeCount: 0, webCount: webs.length, databaseCount: databases.length,
        addonCount: addons.length, visibleAddons: 0, totalAddons: 0, hasMoreAddons: false
      };
    }

    const selected = languages.find(language => language.id === selectedLangKey) || languages[0];
    const compatibleAddons = computeCompatibleAddonsForSelection(components, selected, null, 'IDE_PERSISTENTE');
    const matchingAddons = compatibleAddons.filter(matchesQuery);
    const visibleAddons = showAllAddons ? matchingAddons : matchingAddons.slice(0, VISIBLE_ADDON_LIMIT);
    const languageNode = this.toNode(selected, 'lenguaje', compatibleAddons.length);
    const middleNodes = [
      ...webs.filter(matchesQuery).map(comp => this.toNode(comp, 'web', 0)),
      ...databases.filter(matchesQuery).map(comp => this.toNode(comp, 'database', 0))
    ];
    const addonNodes = visibleAddons.map(comp => this.toNode(comp, 'complemento', 0));
    const nodes = [languageNode, ...middleNodes, ...addonNodes];
    const edges: TechGraphEdge[] = [
      ...middleNodes.map(node => ({ key: `${languageNode.key}->${node.key}`, source: languageNode.key, target: node.key, kind: 'combinable' as const })),
      ...addonNodes.map(node => ({ key: `${languageNode.key}->${node.key}`, source: languageNode.key, target: node.key, kind: 'compatible' as const }))
    ];
    const compatibleIds = new Set(compatibleAddons.map(addon => addon.id));
    const orphanAddons = addons.filter(addon => !compatibleIds.has(addon.id));
    const visibleComponents = [selected, ...middleNodes.map(node => components.find(comp => comp.id === node.key)!).filter(Boolean), ...visibleAddons];
    const chipCounts: Record<string, number> = {};
    for (const language of languages) {
      chipCounts[language.id] = webs.length + databases.length +
        computeCompatibleAddonsForSelection(components, language, null, 'IDE_PERSISTENTE').length;
    }

    return {
      languages: languages.map(language => this.toNode(language, 'lenguaje', 0)),
      chipCounts,
      selectedLanguage: languageNode,
      nodes,
      edges,
      activeCount: visibleComponents.filter(comp => comp.is_active !== false).length,
      inactiveCount: visibleComponents.filter(comp => comp.is_active === false).length,
      orphanCount: orphanAddons.length,
      runtimeCount: languages.length,
      webCount: webs.length,
      databaseCount: databases.length,
      addonCount: addons.length,
      visibleAddons: visibleAddons.length,
      totalAddons: matchingAddons.length,
      hasMoreAddons: matchingAddons.length > visibleAddons.length
    };
  }
}
