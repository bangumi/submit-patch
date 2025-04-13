import hashlib
import itertools
from functools import lru_cache
from pathlib import Path

from server.config import DEV, PROJECT_PATH


def digest_file(p: Path) -> Path:
    h = hashlib.blake2b()
    with p.open("rb") as f:
        while data := f.read(4096):
            h.update(data)

    digest = h.hexdigest()[:8]
    return p.with_suffix("." + digest + p.suffix)


def build_map() -> dict[str, str]:
    files = {}
    for file in itertools.chain(
        PROJECT_PATH.joinpath("static").glob("src/**/*"),
    ):
        src = "/" + file.relative_to(PROJECT_PATH).as_posix()
        target = digest_file(file)
        files[src] = "/" + target.relative_to(PROJECT_PATH).as_posix()
    return files


if DEV:
    __files = {}
else:
    __files = build_map()


@lru_cache
def static_file_path(s: str) -> str:
    return __files.get(s, s)
