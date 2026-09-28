export interface Board {
  id: number;
  name: string;
  chat_id: number | null;
  created_by: number;
  created_at: string;
  columns?: Column[];
  members?: Member[];
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
  description: string | null;
  position: number;
  created_by: number;
  deadline: string | null;
  priority: "low" | "medium" | "high" | "urgent";
  created_at: string;
  updated_at: string;
  assignees: Assignee[];
}

export interface Assignee {
  user_id: number;
  username: string | null;
  display_name: string | null;
}

export interface Member {
  user_id: number;
  username: string | null;
  display_name: string | null;
  role: string;
}

export interface AuthUser {
  user_id: number;
  username: string | null;
  first_name: string | null;
  last_name: string | null;
}
