import contextlib
import gzip
import io
import json
import shutil
from pathlib import Path
from tarfile import TarFile

import httpx
from sslog import logger


project_root = Path(__file__, "../..").resolve()
static_path = project_root.joinpath("server/static")
client = httpx.Client(proxies="http://192.168.1.3:7890")


def download_npm_package(
    name: str,
    path_filter: tuple[str, ...] | None = None,
    version_spec: str = "latest",
) -> None:
    target = static_path.joinpath(name)
    package_json = target.joinpath("installed.json")

    latest_version = (
        client.get(f"https://cdn.jsdelivr.net/npm/{name}@{version_spec}/package.json")
        .raise_for_status()
        .json()["version"]
    )

    logger.info(f"[{name}]: latest version {latest_version}")

    if package_json.exists():
        if json.loads(package_json.read_bytes())["version"] == latest_version:
            if target.joinpath(latest_version).exists():
                return

    logger.info(f"[{name}]: download new version {latest_version}")

    with contextlib.suppress(FileNotFoundError):
        shutil.rmtree(target)

    target.mkdir(exist_ok=True)
    package_json.write_bytes(json.dumps({"version": latest_version}).encode())

    registry = client.get(f"https://registry.npmjs.org/{name}/").raise_for_status().json()
    version = registry["versions"][latest_version]
    tarball = client.get(version["dist"]["tarball"]).raise_for_status()

    with TarFile(fileobj=io.BytesIO(gzip.decompress(tarball.content))) as tar:
        for file in tar:
            if not file.isfile():
                continue
            fn = file.path.removeprefix("package/")
            if fn == "package.json":
                continue
            if path_filter:
                if not (fn.startswith(path_filter)):
                    continue
            target_file = target.joinpath(latest_version, fn)
            target_file.parent.mkdir(parents=True, exist_ok=True)
            f = tar.extractfile(file)
            if not f:
                raise Exception(f"extractfile return unexpected none for file {file}")
            target_file.write_bytes(f.read())


def main() -> None:
    package_json = json.loads(Path(__file__, "../../package.json").read_bytes())

    for pkg, skip in [
        ("diff2html", ("bundles",)),
        ("bootstrap", None),
        ("jquery", ("dist",)),
        ("js-cookie", None),
    ]:
        download_npm_package(
            pkg,
            skip,
            version_spec=package_json["dependencies"][pkg],
        )


main()
