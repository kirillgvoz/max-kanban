import os
from contextlib import asynccontextmanager
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from app.database import get_db, close_db
from app.routers import tasks, boards, columns, auth, webhook

DB_PATH = os.getenv("DATABASE_PATH", "taskflow.db")


@asynccontextmanager
async def lifespan(app: FastAPI):
    db = await get_db()
    schema_path = os.path.join(os.path.dirname(__file__), "db", "schema.sql")
    with open(schema_path, "r") as f:
        await db.executescript(f.read())
    await db.commit()
    yield
    await close_db()


app = FastAPI(title="TaskFlow MAX", version="1.0.0", lifespan=lifespan)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

app.include_router(auth.router, prefix="/api/auth", tags=["auth"])
app.include_router(boards.router, prefix="/api/boards", tags=["boards"])
app.include_router(columns.router, prefix="/api/columns", tags=["columns"])
app.include_router(tasks.router, prefix="/api/tasks", tags=["tasks"])
app.include_router(webhook.router, prefix="/webhook", tags=["webhook"])


@app.get("/api/health")
async def health():
    return {"status": "ok"}
