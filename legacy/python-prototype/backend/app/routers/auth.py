import hmac
import hashlib
from urllib.parse import unquote
from fastapi import APIRouter, HTTPException
from app.models import AuthRequest, AuthResponse
from app.config import MAX_BOT_TOKEN

router = APIRouter()


def validate_init_data(init_data: str, bot_token: str) -> dict | None:
    try:
        params = dict(p.split("=", 1) for p in init_data.split("&"))
        received_hash = params.pop("hash", None)
        if not received_hash:
            return None
        params = {k: unquote(v) for k, v in params.items()}
        check_string = "\n".join(f"{k}={v}" for k, v in sorted(params.items()))
        secret_key = hmac.new(b"WebAppData", bot_token.encode(), hashlib.sha256).digest()
        computed = hmac.new(secret_key, check_string.encode(), hashlib.sha256).hexdigest()
        if computed == received_hash:
            return params
        return None
    except Exception:
        return None


@router.post("/validate", response_model=AuthResponse)
async def validate_auth(body: AuthRequest):
    data = validate_init_data(body.initData, MAX_BOT_TOKEN)
    if not data:
        raise HTTPException(status_code=401, detail="Invalid initData")
    user = data.get("user", {})
    return AuthResponse(
        user_id=int(user.get("id", 0)),
        username=user.get("username"),
        first_name=user.get("first_name"),
        last_name=user.get("last_name"),
    )
