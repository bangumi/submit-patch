import os
import secrets
from datetime import timezone
from pathlib import Path

import tomli
from dotenv import load_dotenv


UTC = timezone.utc

PROJECT_PATH = Path(__file__, "..").resolve()

load_dotenv(PROJECT_PATH.joinpath(".env"))

SECRET_TOKEN = bytes.fromhex(os.environ.get("SECRET_TOKEN", secrets.token_hex(32)))
CSRF_SECRET_TOKEN = os.environ.get("CSRF_SECRET_TOKEN", secrets.token_urlsafe(32))

SERVER_BASE_URL = os.environ["SERVER_BASE_URL"]

BGM_TV_APP_ID = os.environ["BGM_TV_APP_ID"]
BGM_TV_APP_SECRET = os.environ["BGM_TV_APP_SECRET"]

PG_DSN = os.environ["PG_DSN"]
REDIS_DSN = os.environ["REDIS_DSN"]

__py_project = tomli.loads(PROJECT_PATH.joinpath("pyproject.toml").read_text(encoding="utf-8"))

VERSION = "v" + __py_project["tool"]["poetry"]["version"]
