import hashlib
import json
import shutil
from pathlib import Path

import yaml


project_root = Path(__file__, "../../").resolve()
common_path = project_root.joinpath("common").resolve()

target_path = project_root.joinpath("static/common")

shutil.rmtree(target_path)

target_path.mkdir(exist_ok=True, parents=True)

file: Path
for file in common_path.iterdir():
    if file.suffix in {".yaml", ".yml"}:
        target_path.joinpath(
            file.with_suffix(
                "." + hashlib.sha3_256(file.read_bytes()).hexdigest()[:6] + ".json"
            ).name
        ).write_bytes(json.dumps(yaml.safe_load(file.read_bytes())).encode())
