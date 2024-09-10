import os
import sys
from pathlib import Path

from dateutil import tz


DEV = sys.platform == "win32"

UTC = tz.UTC

PROJECT_PATH = Path(__file__, "../..").resolve()

SERVER_BASE_URL = os.environ["SERVER_BASE_URL"]

CSRF_SECRET_TOKEN = os.environ["CSRF_SECRET_TOKEN"]

BGM_TV_APP_ID = os.environ["BGM_TV_APP_ID"]
BGM_TV_APP_SECRET = os.environ["BGM_TV_APP_SECRET"]

PG_DSN = os.environ["PG_DSN"]
REDIS_DSN = os.environ["REDIS_DSN"]

TURNSTILE_SITE_KEY = os.environ["TURNSTILE_SITE_KEY"]
TURNSTILE_SECRET_KEY = os.environ["TURNSTILE_SECRET_KEY"]

HEADER_KEY_API = "x-api-token"

SUPER_USERS = {}

for key in os.environ:
    if not key.startswith("SUPER_USER_TOKEN_"):
        continue
    if __token := os.environ.get(key):
        if __token in SUPER_USERS:
            raise ValueError(f"find duplicated super user token for {key}")
        SUPER_USERS[__token] = {"user_id": int(key.removeprefix("SUPER_USER_TOKEN_"))}
