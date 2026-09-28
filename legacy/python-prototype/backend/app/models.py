from pydantic import BaseModel
from datetime import date, datetime
from typing import Optional


# --- Board ---
class BoardCreate(BaseModel):
    name: str
    chat_id: Optional[int] = None


class BoardResponse(BaseModel):
    id: int
    name: str
    chat_id: Optional[int] = None
    created_by: int
    created_at: str


class BoardDetail(BoardResponse):
    columns: list["ColumnResponse"]
    members: list["MemberResponse"]


# --- Column ---
class ColumnCreate(BaseModel):
    name: str
    position: Optional[int] = 0
    color: Optional[str] = "#6B7280"


class ColumnUpdate(BaseModel):
    name: Optional[str] = None
    position: Optional[int] = None
    color: Optional[str] = None


class ColumnResponse(BaseModel):
    id: int
    board_id: int
    name: str
    position: int
    color: str


# --- Task ---
class TaskCreate(BaseModel):
    title: str
    description: Optional[str] = None
    column_id: int
    deadline: Optional[date] = None
    priority: Optional[str] = "medium"
    assignee_ids: Optional[list[int]] = []


class TaskUpdate(BaseModel):
    title: Optional[str] = None
    description: Optional[str] = None
    column_id: Optional[int] = None
    position: Optional[int] = None
    deadline: Optional[date] = None
    priority: Optional[str] = None


class TaskMove(BaseModel):
    column_id: int
    position: int


class TaskResponse(BaseModel):
    id: int
    board_id: int
    column_id: int
    title: str
    description: Optional[str] = None
    position: int
    created_by: int
    deadline: Optional[str] = None
    priority: str
    created_at: str
    updated_at: str
    assignees: list["AssigneeResponse"] = []


class AssigneeResponse(BaseModel):
    user_id: int
    username: Optional[str] = None
    display_name: Optional[str] = None


class MemberResponse(BaseModel):
    user_id: int
    username: Optional[str] = None
    display_name: Optional[str] = None
    role: str


# --- Auth ---
class AuthRequest(BaseModel):
    initData: str


class AuthResponse(BaseModel):
    user_id: int
    username: Optional[str] = None
    first_name: Optional[str] = None
    last_name: Optional[str] = None


# --- Callback ---
class CallbackPayload(BaseModel):
    action: str
    task_id: int
