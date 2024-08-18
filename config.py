import os
import sys
from datetime import timezone
from pathlib import Path

import tomli
from dotenv import load_dotenv


DEV = sys.platform == "win32"

UTC = timezone.utc

PROJECT_PATH = Path(__file__, "..").resolve()

load_dotenv(PROJECT_PATH.joinpath(".env"))

SECRET_TOKEN = bytes.fromhex(os.environ["SECRET_TOKEN"])
CSRF_SECRET_TOKEN = os.environ["CSRF_SECRET_TOKEN"]

SERVER_BASE_URL = os.environ["SERVER_BASE_URL"]

BGM_TV_APP_ID = os.environ["BGM_TV_APP_ID"]
BGM_TV_APP_SECRET = os.environ["BGM_TV_APP_SECRET"]

PG_DSN = os.environ["PG_DSN"]
REDIS_DSN = os.environ["REDIS_DSN"]

__py_project = tomli.loads(PROJECT_PATH.joinpath("pyproject.toml").read_text(encoding="utf-8"))

VERSION = "v" + __py_project["tool"]["poetry"]["version"]
