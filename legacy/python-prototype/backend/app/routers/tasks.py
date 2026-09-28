from fastapi import APIRouter, HTTPException
from app.database import get_db
from app.models import TaskCreate, TaskUpdate, TaskMove, TaskResponse, AssigneeResponse
from app.services import task_service
from app.services.notification import notify_new_task, notify_status_change

router = APIRouter()


@router.get("/board/{board_id}", response_model=list[TaskResponse])
async def list_tasks(board_id: int):
    tasks = await task_service.get_tasks_by_board(board_id)
    result = []
    for t in tasks:
        t["assignees"] = [AssigneeResponse(**a) for a in t.get("assignees", [])]
        result.append(TaskResponse(**t))
    return result


@router.get("/{task_id}", response_model=TaskResponse)
async def get_task(task_id: int):
    t = await task_service.get_task_detail(task_id)
    if not t:
        raise HTTPException(status_code=404, detail="Task not found")
    t["assignees"] = [AssigneeResponse(**a) for a in t.get("assignees", [])]
    return TaskResponse(**t)


@router.post("", response_model=TaskResponse, status_code=201)
async def create_task(body: TaskCreate, created_by: int = 0):
    task_id = await task_service.create_task(
        board_id=0, title=body.title, description=body.description,
        column_id=body.column_id, created_by=created_by,
        deadline=str(body.deadline) if body.deadline else None,
        priority=body.priority or "medium",
        assignee_ids=body.assignee_ids or [],
    )
    await notify_new_task(task_id)
    t = await task_service.get_task_detail(task_id)
    t["assignees"] = [AssigneeResponse(**a) for a in t.get("assignees", [])]
    return TaskResponse(**t)


@router.post("/create-on-board", response_model=TaskResponse, status_code=201)
async def create_task_on_board(body: TaskCreate, board_id: int, created_by: int = 0):
    task_id = await task_service.create_task(
        board_id=board_id, title=body.title, description=body.description,
        column_id=body.column_id, created_by=created_by,
        deadline=str(body.deadline) if body.deadline else None,
        priority=body.priority or "medium",
        assignee_ids=body.assignee_ids or [],
    )
    await notify_new_task(task_id)
    t = await task_service.get_task_detail(task_id)
    t["assignees"] = [AssigneeResponse(**a) for a in t.get("assignees", [])]
    return TaskResponse(**t)


@router.patch("/{task_id}", response_model=TaskResponse)
async def update_task(task_id: int, body: TaskUpdate):
    old = await task_service.get_task_detail(task_id)
    if not old:
        raise HTTPException(status_code=404, detail="Task not found")
    fields = body.model_dump(exclude_unset=True)
    if "deadline" in fields and fields["deadline"] is not None:
        fields["deadline"] = str(fields["deadline"])
    t = await task_service.update_task(task_id, **fields)
    if "column_id" in fields and fields["column_id"] != old["column_id"]:
        db = await get_db()
        col = await db.execute_fetchall(
            "SELECT name FROM columns WHERE id = ?", (fields["column_id"],)
        )
        if col:
            await notify_status_change(task_id, dict(col[0])["name"], "Пользователь")
    t["assignees"] = [AssigneeResponse(**a) for a in t.get("assignees", [])]
    return TaskResponse(**t)


@router.put("/{task_id}/move", response_model=TaskResponse)
async def move_task(task_id: int, body: TaskMove):
    old = await task_service.get_task_detail(task_id)
    if not old:
        raise HTTPException(status_code=404, detail="Task not found")
    t = await task_service.move_task(task_id, body.column_id, body.position)
    db = await get_db()
    col = await db.execute_fetchall(
        "SELECT name FROM columns WHERE id = ?", (body.column_id,)
    )
    if col:
        await notify_status_change(task_id, dict(col[0])["name"], "Пользователь")
    t = await task_service.get_task_detail(task_id)
    t["assignees"] = [AssigneeResponse(**a) for a in t.get("assignees", [])]
    return TaskResponse(**t)


@router.delete("/{task_id}")
async def delete_task(task_id: int):
    await task_service.delete_task(task_id)
    return {"ok": True}


@router.post("/{task_id}/assign")
async def assign_task(task_id: int, user_id: int, username: str = "", display_name: str = ""):
    await task_service.assign_task(task_id, user_id, username, display_name)
    return {"ok": True}


@router.delete("/{task_id}/unassign")
async def unassign_task(task_id: int, user_id: int):
    await task_service.unassign_task(task_id, user_id)
    return {"ok": True}
