import pytest

import click


def missing(runner, envvar, **kwargs):
    option = click.Option(
        ["--units"], envvar=envvar, show_envvar=True, required=True, **kwargs
    )
    result = runner.invoke(click.Command("cli", params=[option]))
    assert result.exit_code == 2
    return result.output


def help_text(runner, envvar, prefix=None, show_envvar=True):
    option = click.Option(["--units"], envvar=envvar, show_envvar=show_envvar)
    result = runner.invoke(
        click.Command("cli", params=[option]), ["--help"], auto_envvar_prefix=prefix
    )
    assert not result.exception
    return result.output


@pytest.mark.parametrize(
    ("envvar", "hint"),
    [
        ("UNITS", " (env var: 'UNITS')"),
        (["UNITS"], " (env var: 'UNITS')"),
        (("UNITS",), " (env var: 'UNITS')"),
        (["WEATHER_UNITS", "UNITS"], " (env var: 'WEATHER_UNITS', 'UNITS')"),
        (("WEATHER_UNITS", "UNITS"), " (env var: 'WEATHER_UNITS', 'UNITS')"),
        (["A", "B", "C"], " (env var: 'A', 'B', 'C')"),
        ("MY UNITS", " (env var: 'MY UNITS')"),
    ],
)
def test_hidden_missing_option_hint(runner, envvar, hint):
    assert f"Error: Missing option '--units'{hint}." in missing(runner, envvar)


@pytest.mark.parametrize("envvar", [None, [], (), ""])
def test_hidden_missing_option_empty_envvar(runner, envvar):
    out = missing(runner, envvar)
    assert "Error: Missing option '--units'." in out
    assert "env var" not in out


def test_hidden_missing_option_empty_envvar_ignores_prefix(runner):
    option = click.Option(["--units"], envvar=[], show_envvar=True, required=True)
    result = runner.invoke(click.Command("cli", params=[option]), auto_envvar_prefix="W")
    assert "Error: Missing option '--units'." in result.output


@pytest.mark.parametrize(
    ("envvar", "hint"),
    [
        (("COUNT", "N"), " (env var: 'COUNT', 'N')"),
        (["COUNT"], " (env var: 'COUNT')"),
        ([], ""),
        ("", ""),
    ],
)
def test_hidden_invalid_value_hint(runner, envvar, hint):
    option = click.Option(["-c", "--count"], type=int, envvar=envvar, show_envvar=True)
    result = runner.invoke(click.Command("cli", params=[option]), ["--count", "x"])
    assert result.exit_code == 2
    assert f"Invalid value for '-c' / '--count'{hint}: 'x' is not a valid integer." in (
        result.output
    )


def test_hidden_hint_without_show_envvar(runner):
    option = click.Option(["--units"], envvar=["A", "B"], required=True)
    result = runner.invoke(click.Command("cli", params=[option]))
    assert "Error: Missing option '--units'." in result.output


@pytest.mark.parametrize("envvar", [None, [], (), ""])
def test_hidden_help_empty_envvar(runner, envvar):
    out = help_text(runner, envvar)
    assert "  --units TEXT\n" in out
    assert "env var" not in out


@pytest.mark.parametrize("envvar", [None, [], (), ""])
def test_hidden_help_empty_envvar_uses_auto_prefix(runner, envvar):
    assert "  --units TEXT  [env var: WEATHER_UNITS]\n" in help_text(
        runner, envvar, prefix="WEATHER"
    )


@pytest.mark.parametrize(
    ("envvar", "extra"),
    [
        ("UNITS", "[env var: UNITS]"),
        (["WEATHER_UNITS", "UNITS"], "[env var: WEATHER_UNITS, UNITS]"),
        (("WEATHER_UNITS", "UNITS"), "[env var: WEATHER_UNITS, UNITS]"),
    ],
)
def test_hidden_help_non_empty_envvar_unchanged(runner, envvar, extra):
    assert f"  --units TEXT  {extra}\n" in help_text(runner, envvar, prefix="WEATHER")


@pytest.mark.parametrize("envvar", [None, [], "UNITS", ["A", "B"]])
def test_hidden_help_without_show_envvar(runner, envvar):
    assert "env var" not in help_text(runner, envvar, prefix="WEATHER", show_envvar=False)


def test_hidden_empty_envvar_reads_auto_variable(runner):
    seen = []
    option = click.Option(["--units"], envvar=[], show_envvar=True)
    cli = click.Command("cli", params=[option], callback=lambda units: seen.append(units))
    result = runner.invoke(cli, auto_envvar_prefix="WEATHER", env={"WEATHER_UNITS": "si"})
    assert not result.exception
    assert seen == ["si"]
