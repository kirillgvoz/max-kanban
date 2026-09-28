import os
from dotenv import load_dotenv

load_dotenv()

MAX_BOT_TOKEN = os.getenv("MAX_BOT_TOKEN", "")
MAX_API_BASE = "https://platform-api2.max.ru"
WEBHOOK_SECRET = os.getenv("WEBHOOK_SECRET", "taskflow_secret_key_change_me")
DATABASE_PATH = os.getenv("DATABASE_PATH", "taskflow.db")
FRONTEND_URL = os.getenv("FRONTEND_URL", "https://kirillgvoz.ru/max-kanban/")
BOT_NAME = os.getenv("BOT_NAME", "se14445725_bot")
