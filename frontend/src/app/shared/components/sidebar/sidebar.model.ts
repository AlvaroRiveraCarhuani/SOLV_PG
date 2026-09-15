export type SidebarIconName = 
  | 'home'
  | 'terminal'
  | 'book'
  | 'award'
  | 'history'
  | 'settings'
  | 'activity'
  | 'users'
  | 'layers'
  | 'sliders'
  | 'shield-alert'
  | 'file-text';

export interface NavItem {
  label: string;
  route: string;
  iconName: SidebarIconName;
  exact?: boolean;
}

export interface NavSection {
  title?: string;
  items: NavItem[];
  isFooter?: boolean;
}
