import json
from pathlib import Path

import yaml


project_root = Path(__file__, "../../").resolve()
common_path = project_root.joinpath("common").resolve()

target_path = project_root.joinpath("static/common")

target_path.mkdir(exist_ok=True, parents=True)

file: Path
for file in common_path.iterdir():
    if file.suffix in {".yaml", ".yml"}:
        print(file)
        target_path.joinpath(file.with_suffix(".json").name).write_bytes(
            json.dumps(yaml.safe_load(file.read_bytes())).encode()
        )
