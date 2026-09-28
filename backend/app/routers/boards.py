from fastapi import APIRouter, HTTPException
from app.database import get_db
from app.models import BoardCreate, BoardResponse, BoardDetail, ColumnResponse, MemberResponse

router = APIRouter()


@router.get("", response_model=list[BoardResponse])
async def list_boards(user_id: int | None = None):
    db = await get_db()
    if user_id:
        rows = await db.execute_fetchall(
            """SELECT b.* FROM boards b
               JOIN board_members bm ON b.id = bm.board_id
               WHERE bm.user_id = ?
               ORDER BY b.created_at DESC""",
            (user_id,),
        )
    else:
        rows = await db.execute_fetchall(
            "SELECT * FROM boards ORDER BY created_at DESC"
        )
    return [BoardResponse(**dict(r)) for r in rows]


@router.post("", response_model=BoardResponse, status_code=201)
async def create_board(body: BoardCreate, created_by: int = 0):
    db = await get_db()
    cursor = await db.execute(
        "INSERT INTO boards (name, chat_id, created_by) VALUES (?, ?, ?)",
        (body.name, body.chat_id, created_by),
    )
    board_id = cursor.lastrowid
    default_columns = [
        ("К выполнению", 0, "#6366F1"),
        ("В работе", 1, "#F59E0B"),
        ("Готово", 2, "#10B981"),
    ]
    for name, pos, color in default_columns:
        await db.execute(
            "INSERT INTO columns (board_id, name, position, color) VALUES (?, ?, ?, ?)",
            (board_id, name, pos, color),
        )
    await db.execute(
        "INSERT OR IGNORE INTO board_members (board_id, user_id, role) VALUES (?, ?, 'owner')",
        (board_id, created_by),
    )
    await db.commit()
    row = await db.execute_fetchall("SELECT * FROM boards WHERE id = ?", (board_id,))
    return BoardResponse(**dict(row[0]))


@router.get("/{board_id}", response_model=BoardDetail)
async def get_board(board_id: int):
    db = await get_db()
    rows = await db.execute_fetchall("SELECT * FROM boards WHERE id = ?", (board_id,))
    if not rows:
        raise HTTPException(status_code=404, detail="Board not found")
    board = dict(rows[0])

    cols = await db.execute_fetchall(
        "SELECT * FROM columns WHERE board_id = ? ORDER BY position",
        (board_id,),
    )
    members = await db.execute_fetchall(
        "SELECT * FROM board_members WHERE board_id = ?",
        (board_id,),
    )
    return BoardDetail(
        **board,
        columns=[ColumnResponse(**dict(c)) for c in cols],
        members=[MemberResponse(**dict(m)) for m in members],
    )


@router.post("/{board_id}/members")
async def add_member(board_id: int, user_id: int, username: str = "", display_name: str = "", role: str = "member"):
    db = await get_db()
    await db.execute(
        """INSERT OR REPLACE INTO board_members (board_id, user_id, username, display_name, role)
           VALUES (?, ?, ?, ?, ?)""",
        (board_id, user_id, username, display_name, role),
    )
    await db.commit()
    return {"ok": True}
