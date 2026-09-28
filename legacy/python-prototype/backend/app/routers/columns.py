from fastapi import APIRouter, HTTPException
from app.database import get_db
from app.models import ColumnCreate, ColumnUpdate, ColumnResponse

router = APIRouter()


@router.post("", response_model=ColumnResponse, status_code=201)
async def create_column(board_id: int, body: ColumnCreate):
    db = await get_db()
    if body.position == 0:
        max_pos = await db.execute_fetchall(
            "SELECT COALESCE(MAX(position), -1) + 1 FROM columns WHERE board_id = ?",
            (board_id,),
        )
        position = dict(max_pos[0])["COALESCE(MAX(position), -1) + 1"]
    else:
        position = body.position
    cursor = await db.execute(
        "INSERT INTO columns (board_id, name, position, color) VALUES (?, ?, ?, ?)",
        (board_id, body.name, position, body.color),
    )
    await db.commit()
    row = await db.execute_fetchall("SELECT * FROM columns WHERE id = ?", (cursor.lastrowid,))
    return ColumnResponse(**dict(row[0]))


@router.patch("/{column_id}", response_model=ColumnResponse)
async def update_column(column_id: int, body: ColumnUpdate):
    db = await get_db()
    updates, values = [], []
    if body.name is not None:
        updates.append("name = ?")
        values.append(body.name)
    if body.position is not None:
        updates.append("position = ?")
        values.append(body.position)
    if body.color is not None:
        updates.append("color = ?")
        values.append(body.color)
    if not updates:
        raise HTTPException(status_code=400, detail="No fields to update")
    values.append(column_id)
    await db.execute(f"UPDATE columns SET {', '.join(updates)} WHERE id = ?", values)
    await db.commit()
    row = await db.execute_fetchall("SELECT * FROM columns WHERE id = ?", (column_id,))
    if not row:
        raise HTTPException(status_code=404, detail="Column not found")
    return ColumnResponse(**dict(row[0]))


@router.delete("/{column_id}")
async def delete_column(column_id: int):
    db = await get_db()
    await db.execute("DELETE FROM columns WHERE id = ?", (column_id,))
    await db.commit()
    return {"ok": True}
