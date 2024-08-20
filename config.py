import os
import sys
from datetime import timezone
from pathlib import Path


DEV = sys.platform == "win32"

UTC = timezone.utc

PROJECT_PATH = Path(__file__, "..").resolve()


SECRET_TOKEN = bytes.fromhex(os.environ["SECRET_TOKEN"])
CSRF_SECRET_TOKEN = os.environ["CSRF_SECRET_TOKEN"]

SERVER_BASE_URL = os.environ["SERVER_BASE_URL"]

BGM_TV_APP_ID = os.environ["BGM_TV_APP_ID"]
BGM_TV_APP_SECRET = os.environ["BGM_TV_APP_SECRET"]

PG_DSN = os.environ["PG_DSN"]
REDIS_DSN = os.environ["REDIS_DSN"]

TURNSTILE_SITE_KEY = os.environ["TURNSTILE_SITE_KEY"]
TURNSTILE_SECRET_KEY = os.environ["TURNSTILE_SECRET_KEY"]
