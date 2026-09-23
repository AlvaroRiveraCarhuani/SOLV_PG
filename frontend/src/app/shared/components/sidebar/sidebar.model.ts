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
  | 'boxes'
  | 'sliders'
  | 'shield-alert'
  | 'file-text'
  | 'graduation-cap';

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
