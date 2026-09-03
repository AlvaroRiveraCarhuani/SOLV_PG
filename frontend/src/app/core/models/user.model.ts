export type UserRole = 'student' | 'teacher' | 'admin';

export interface User {
  id: string;
  first_name: string;
  last_name: string;
  email: string;
  role: UserRole;
  tenant_id: string;
  avatar_url?: string;
  created_at?: string;
}
