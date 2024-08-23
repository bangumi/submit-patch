from pathlib import Path

import httpx


c = httpx.Client()
s = ""

for key, (value, color) in {
    "0": ("0", "green"),
    "lt10": ("<=10", "green"),
    "gt10": (">10", "ffc107"),
    "gt20": (">20", "ffc107"),
    "gt30": (">30", "ffc107"),
    "gt40": (">40", "ffc107"),
    "gt50": (">50", "ffc107"),
    "gt60": (">60", "dc3545"),
    "gt70": (">70", "dc3545"),
    "gt80": (">80", "dc3545"),
    "gt90": (">90", "dc3545"),
    "gt100": (">100", "dc3545"),
}.items():
    res = c.get(f"https://img.shields.io/badge/待审核-{value}-{color}")
    s += f"badge_{key} = " + repr(res.text) + ".encode()\n"

Path(__file__, "../../server/badge.py").write_text(s, encoding="utf8")
