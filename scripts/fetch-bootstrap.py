import io
import zipfile
from pathlib import Path

import httpx


project_root = Path(__file__, "../..").resolve()

static_path = project_root.joinpath("server/static")

client = httpx.Client(proxies="http://192.168.1.3:7890")

r = client.get(
    "https://github.com/twbs/bootstrap/releases/latest/download/bootstrap-5.3.3-dist.zip",
    follow_redirects=True,
)

content = r.raise_for_status().content

with io.BytesIO(content) as f:
    with zipfile.ZipFile(f) as tar:
        for member in tar.filelist:
            if member.is_dir():
                static_path.joinpath(member.filename).mkdir(exist_ok=True, parents=True)
                continue
            static_path.joinpath(member.filename).write_bytes(tar.read(member.filename))
