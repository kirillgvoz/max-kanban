export interface Organization {
  id: number;
  name: string;
  slug: string;
  avatar_url?: string;
  created_by: number;
  created_at: string;
}

export interface OrgMember {
  org_id: number;
  user_id: number;
  username?: string;
  display_name?: string;
  role: string;
  joined_at: string;
}

export interface OrgDetail extends Organization {
  members: OrgMember[];
}

export interface Board {
  id: number;
  org_id: number;
  name: string;
  description: string;
  is_archived: boolean;
  created_by: number;
  created_at: string;
}

export interface BoardDetail extends Board {
  columns: Column[];
  members: OrgMember[];
}

export interface BoardChat {
  id: number;
  board_id: number;
  chat_id: number;
  title?: string;
  created_by: number;
  created_at: string;
}

export interface Column {
  id: number;
  board_id: number;
  name: string;
  position: number;
  color: string;
}

export interface Task {
  id: number;
  board_id: number;
  column_id: number;
  title: string;
  description: string;
  position: number;
  priority: "low" | "medium" | "high" | "urgent";
  deadline: string | null;
  created_by: number;
  created_at: string;
  updated_at: string;
  assignees: Assignee[];
}

export interface TaskDetail extends Task {
  checklists: Checklist[];
  comments: Comment[];
}

export interface Assignee {
  task_id: number;
  user_id: number;
  username?: string;
  display_name?: string;
}

export interface Checklist {
  id: number;
  task_id: number;
  title: string;
  position: number;
  items: ChecklistItem[];
}

export interface ChecklistItem {
  id: number;
  checklist_id: number;
  text: string;
  is_done: boolean;
  position: number;
}

export interface Comment {
  id: number;
  task_id: number;
  user_id: number;
  username?: string;
  display_name?: string;
  text: string;
  created_at: string;
}

export interface AuthUser {
  user_id: number;
  username?: string;
  first_name?: string;
  last_name?: string;
  display_name?: string;
}

export type WSEvent = {
  type: string;
  data: any;
};
