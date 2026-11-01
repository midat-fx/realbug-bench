# Hidden tests: never shown to the agent.
import pytest

import tomlkit
from tomlkit.exceptions import ParseError


def position(text):
    with pytest.raises(ParseError) as e:
        tomlkit.parse(text)
    assert f"at line {e.value.line} col {e.value.col}" in str(e.value)
    return e.value.line, e.value.col


# (document with "\n" line endings, expected position)
CASES = [
    ("a = 1\nb = 2\nc = @\nd = 4\n", (3, 4)),
    ("a = @\n", (1, 4)),
    ("".join(f"k{i} = {i}\n" for i in range(40)) + "bad = @\n", (41, 6)),
    ('a = """x\ny\nz"""\nb = @\n', (4, 4)),
    ("[t]\nx = 1\n# comment\n\n[u]\ny = 2\nz = [1, @]\n", (7, 8)),
    ('a = 1\nb = "abc\nc = 2\n', (2, 8)),
    ("a = 1\nb =\nc = 2\n", (2, 3)),
    ("a = 1\nb = ", (2, 4)),
    ("x = 1\n[t]\ny = ", (3, 4)),
]


@pytest.mark.parametrize(("text", "expected"), CASES)
def test_hidden_lf_positions_unchanged(text, expected):
    assert position(text) == expected


@pytest.mark.parametrize(("text", "expected"), CASES)
def test_hidden_crlf_matches_lf(text, expected):
    assert position(text.replace("\n", "\r\n")) == expected


@pytest.mark.parametrize(
    ("text", "expected"),
    [
        ("a = 1\r\nb = 2\nc = @\r\n", (3, 4)),
        ("a = 1\nb = 2\r\nc = @\n", (3, 4)),
        ("".join(f"k{i} = {i}" + ("\r\n" if i % 3 else "\n") for i in range(30)) + "bad = @\r\n", (31, 6)),
        ("a = 1\r\nb = ", (2, 4)),
    ],
)
def test_hidden_mixed_line_endings(text, expected):
    assert position(text) == expected


def test_hidden_crlf_round_trip():
    text = '# top\r\n[t]\r\na = 1  # c\r\nb = """x\r\ny"""\r\n\r\n[u]\r\nc = [1, 2]\r\n'
    assert tomlkit.parse(text).as_string() == text
    mixed = "a = 1\r\nb = 2\nc = 3\r\n"
    assert tomlkit.dumps(tomlkit.parse(mixed)) == mixed
