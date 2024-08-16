from pathlib import Path

import pytest
import yaml

from lint.wiki import Wiki, parse, read_type


spec_repo_path = Path(r"~\proj\bangumi\wiki-syntax-spec").expanduser().resolve()


def test_read_type():
    assert not read_type("{{Infobox\n")
    assert read_type("{{Infobox Ta\n") == "Ta"
    assert read_type("{{Infobox Ta\n}}") == "Ta"


def as_dict(w: Wiki) -> dict:
    data = []
    for f in w.fields:
        if isinstance(f.value, list):
            data.append(
                {
                    "key": f.key,
                    "array": True,
                    "values": [{"v": v.value} | ({"k": v.key} if v.key else {}) for v in f.value],
                },
            )
        else:
            data.append({"key": f.key, "value": f.value or ""})

    return {"type": w.type, "data": data}


valid = [
    file.name
    for file in spec_repo_path.joinpath("tests/valid").iterdir()
    if file.name.endswith(".wiki")
]


@pytest.mark.parametrize("name", valid)
def test_bangumi_wiki(name: str):
    file = spec_repo_path.joinpath("tests/valid", name)
    wiki_raw = file.read_text()
    assert as_dict(parse(wiki_raw)) == yaml.safe_load(file.with_suffix(".yaml").read_text()), name
