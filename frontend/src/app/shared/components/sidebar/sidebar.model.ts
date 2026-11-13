export type SidebarIconName = 
  | 'home'
  | 'terminal'
  | 'code'
  | 'book'
  | 'award'
  | 'history'
  | 'settings'
  | 'activity'
  | 'users'
  | 'layers'
  | 'boxes'
  | 'inbox'
  | 'sliders'
  | 'shield-alert'
  | 'file-text'
  | 'graduation-cap'
  | 'presentation'
  | 'sparkles'
  | 'tag';

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
