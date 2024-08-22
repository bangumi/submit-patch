import contextlib
import gzip
import io
import json
import shutil
from collections.abc import Callable
from pathlib import Path
from tarfile import TarFile

import httpx
import semver
from loguru import logger


project_root = Path(__file__, "../..").resolve()
static_path = project_root.joinpath("server/static")
client = httpx.Client(proxies="http://192.168.1.3:7890")


def download_npm_package(
    name: str,
    path_filter: tuple[str, ...] | None = None,
    version_filter: Callable[[str], bool] | None = None,
) -> None:
    target = static_path.joinpath(name)
    package_json = target.joinpath("package.json")

    data = client.get(f"https://registry.npmjs.org/{name}/").raise_for_status().json()

    if not version_filter:
        latest_version = data["dist-tags"]["latest"]
    else:
        latest_version = sorted(
            [v for v in data["versions"] if version_filter(v)],
            key=semver.VersionInfo.parse,
            reverse=True,
        )[0]

    logger.info("[{}]: latest version {}", name, latest_version)

    if package_json.exists():
        if json.loads(package_json.read_bytes())["version"] == latest_version:
            return

    logger.info("[{}]: download new version {}", name, latest_version)

    with contextlib.suppress(FileNotFoundError):
        shutil.rmtree(target)

    target.mkdir(exist_ok=True)
    package_json.write_bytes(json.dumps({"version": latest_version}).encode())

    version = data["versions"][latest_version]
    tarball = client.get(version["dist"]["tarball"]).raise_for_status()

    with TarFile(fileobj=io.BytesIO(gzip.decompress(tarball.content))) as tar:
        for file in tar:
            if not file.isfile():
                continue
            fn = file.path.removeprefix("package/")
            if path_filter:
                if not (fn.startswith(path_filter)):
                    continue
            target_file = target.joinpath(latest_version, fn)
            target_file.parent.mkdir(parents=True, exist_ok=True)
            f = tar.extractfile(file)
            if not f:
                raise Exception(f"extractfile return unexpected none for file {file}")
            target_file.write_bytes(f.read())


def build_version_filter(major: int) -> Callable[[str], bool]:
    def f(s: str) -> bool:
        v = semver.VersionInfo.parse(s)
        if v.prerelease:
            return False
        return v.major <= major

    return f


download_npm_package("diff2html", ("bundles",), version_filter=build_version_filter(3))
download_npm_package("bootstrap", version_filter=build_version_filter(5))
download_npm_package("jquery", ("dist",), version_filter=build_version_filter(3))
