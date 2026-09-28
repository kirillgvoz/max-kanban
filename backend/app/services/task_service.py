from app.database import get_db


async def create_task(board_id: int, title: str, description: str | None,
                      column_id: int, created_by: int, deadline: str | None,
                      priority: str, assignee_ids: list[int]) -> int:
    db = await get_db()
    max_pos = await db.execute_fetchall(
        "SELECT COALESCE(MAX(position), -1) + 1 FROM tasks WHERE column_id = ?",
        (column_id,),
    )
    position = dict(max_pos[0])["COALESCE(MAX(position), -1) + 1"]

    cursor = await db.execute(
        """INSERT INTO tasks (board_id, column_id, title, description,
           position, created_by, deadline, priority)
           VALUES (?, ?, ?, ?, ?, ?, ?, ?)""",
        (board_id, column_id, title, description, position, created_by, deadline, priority),
    )
    await db.commit()
    task_id = cursor.lastrowid

    for uid in assignee_ids:
        await db.execute(
            "INSERT OR IGNORE INTO task_assignees (task_id, user_id) VALUES (?, ?)",
            (task_id, uid),
        )
    await db.commit()
    return task_id


async def move_task(task_id: int, new_column_id: int, new_position: int) -> dict:
    db = await get_db()
    await db.execute(
        "UPDATE tasks SET column_id = ?, position = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
        (new_column_id, new_position, task_id),
    )
    await db.commit()
    row = await db.execute_fetchall("SELECT * FROM tasks WHERE id = ?", (task_id,))
    return dict(row[0]) if row else {}


async def update_task(task_id: int, **fields) -> dict:
    db = await get_db()
    updates = []
    values = []
    for k, v in fields.items():
        if v is not None:
            updates.append(f"{k} = ?")
            values.append(v)
    if not updates:
        row = await db.execute_fetchall("SELECT * FROM tasks WHERE id = ?", (task_id,))
        return dict(row[0]) if row else {}
    updates.append("updated_at = CURRENT_TIMESTAMP")
    values.append(task_id)
    await db.execute(
        f"UPDATE tasks SET {', '.join(updates)} WHERE id = ?", values
    )
    await db.commit()
    row = await db.execute_fetchall("SELECT * FROM tasks WHERE id = ?", (task_id,))
    return dict(row[0]) if row else {}


async def delete_task(task_id: int):
    db = await get_db()
    await db.execute("DELETE FROM tasks WHERE id = ?", (task_id,))
    await db.commit()


async def get_tasks_by_board(board_id: int) -> list[dict]:
    db = await get_db()
    rows = await db.execute_fetchall(
        "SELECT * FROM tasks WHERE board_id = ? ORDER BY column_id, position",
        (board_id,),
    )
    tasks = []
    for r in rows:
        t = dict(r)
        assignees = await db.execute_fetchall(
            "SELECT * FROM task_assignees WHERE task_id = ?", (t["id"],)
        )
        t["assignees"] = [dict(a) for a in assignees]
        tasks.append(t)
    return tasks


async def get_task_detail(task_id: int) -> dict | None:
    db = await get_db()
    rows = await db.execute_fetchall("SELECT * FROM tasks WHERE id = ?", (task_id,))
    if not rows:
        return None
    t = dict(rows[0])
    assignees = await db.execute_fetchall(
        "SELECT * FROM task_assignees WHERE task_id = ?", (task_id,)
    )
    t["assignees"] = [dict(a) for a in assignees]
    return t


async def assign_task(task_id: int, user_id: int, username: str | None = None,
                      display_name: str | None = None):
    db = await get_db()
    await db.execute(
        """INSERT OR REPLACE INTO task_assignees (task_id, user_id, username, display_name)
           VALUES (?, ?, ?, ?)""",
        (task_id, user_id, username, display_name),
    )
    await db.commit()


async def unassign_task(task_id: int, user_id: int):
    db = await get_db()
    await db.execute(
        "DELETE FROM task_assignees WHERE task_id = ? AND user_id = ?",
        (task_id, user_id),
    )
    await db.commit()
