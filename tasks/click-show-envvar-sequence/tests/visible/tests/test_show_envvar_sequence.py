import click


def test_missing_option_names_every_envvar(runner):
    option = click.Option(
        ["--units"], envvar=["WEATHER_UNITS", "UNITS"], show_envvar=True, required=True
    )
    result = runner.invoke(click.Command("cli", params=[option]))
    assert result.exit_code == 2
    assert (
        "Error: Missing option '--units' (env var: 'WEATHER_UNITS', 'UNITS')."
        in result.output
    )


def test_empty_string_envvar_shows_no_help_extra(runner):
    option = click.Option(["--units"], envvar="", show_envvar=True)
    result = runner.invoke(click.Command("cli", params=[option]), ["--help"])
    assert not result.exception
    assert "  --units TEXT\n" in result.output
    assert "env var" not in result.output
