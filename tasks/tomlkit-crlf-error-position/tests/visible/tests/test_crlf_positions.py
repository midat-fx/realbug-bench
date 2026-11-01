import pytest

from tomlkit.exceptions import UnexpectedCharError
from tomlkit.parser import Parser


@pytest.mark.parametrize("line_ending", ["\n", "\r\n"])
def test_position_does_not_depend_on_line_ending(line_ending):
    content = line_ending.join(["a = 1", "b = 2", "c = @", "d = 4"]) + line_ending

    with pytest.raises(UnexpectedCharError) as e:
        Parser(content).parse()

    assert (e.value.line, e.value.col) == (3, 4)
