from app.database import get_db
from app.services.max_bot import send_task_notification, send_status_change_notification


async def notify_new_task(task_id: int):
    db = await get_db()
    task = await db.execute_fetchall(
        """SELECT t.*, b.chat_id, b.name as board_name
           FROM tasks t JOIN boards b ON t.board_id = b.id
           WHERE t.id = ?""",
        (task_id,),
    )
    if not task:
        return
    t = dict(task[0])

    assignees = await db.execute_fetchall(
        "SELECT * FROM task_assignees WHERE task_id = ?", (task_id,)
    )
    assignee_names = [
        dict(a).get("display_name")
        or dict(a).get("username")
        or f"Пользователь {dict(a).get('user_id', '?')}"
        for a in assignees
    ]
    assignee_str = ", ".join(assignee_names) if assignee_names else "Не назначен"

    board = await db.execute_fetchall(
        "SELECT chat_id FROM boards WHERE id = ?", (t["board_id"],)
    )
    if not board or not dict(board[0]).get("chat_id"):
        return

    try:
        await send_task_notification(
            chat_id=dict(board[0])["chat_id"],
            task_id=task_id,
            task_title=t["title"],
            assignee_name=assignee_str,
            deadline=t.get("deadline"),
            board_id=t["board_id"],
        )
    except Exception:
        return


async def notify_status_change(task_id: int, new_column_name: str, actor_name: str):
    db = await get_db()
    task = await db.execute_fetchall(
        """SELECT t.*, b.chat_id
           FROM tasks t JOIN boards b ON t.board_id = b.id
           WHERE t.id = ?""",
        (task_id,),
    )
    if not task:
        return
    t = dict(task[0])

    board = await db.execute_fetchall(
        "SELECT chat_id FROM boards WHERE id = ?", (t["board_id"],)
    )
    if not board or not dict(board[0]).get("chat_id"):
        return

    try:
        await send_status_change_notification(
            chat_id=dict(board[0])["chat_id"],
            task_id=task_id,
            task_title=t["title"],
            new_status=new_column_name,
            actor_name=actor_name,
        )
    except Exception:
        return
