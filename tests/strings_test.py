import pytest

from server.strings import escape_invisible


@pytest.mark.parametrize(
    ("input", "expected"),
    [
        ("\u200e\t\u200b", "\\u200e\t\\u200b"),
    ],
)
def test_escape_invisible(input, expected):
    assert escape_invisible(input) == expected
