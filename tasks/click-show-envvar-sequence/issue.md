# `show_envvar` prints the repr of an `envvar` list, and `[env var: ]` for an empty one

An option can read its value from several environment variables (`envvar=["WEATHER_UNITS", "UNITS"]`).
With `show_envvar=True`, the help screen lists them correctly, but the error message prints the
Python repr of the list:

```python
import click

@click.command()
@click.option("--units", envvar=["WEATHER_UNITS", "UNITS"], show_envvar=True, required=True)
def cli(units):
    pass
```

```
$ cli
Error: Missing option '--units' (env var: '['WEATHER_UNITS', 'UNITS']').
```

An empty `envvar` (`[]` or `""`) reads no variable at all, but it is still shown as if it were set:
the error message says `(env var: '[]')` or `(env var: '')`, and the help screen shows
`[env var: ]`.

Expected behaviour, for every error that carries the option's hint (a missing required option, an
invalid value):

- A sequence `envvar` (list or tuple) names every variable, each in single quotes, separated by a
  comma and a space: `Error: Missing option '--units' (env var: 'WEATHER_UNITS', 'UNITS').`
  A one-element sequence reads like a single string: `(env var: 'UNITS')`.
- A string `envvar` is shown as before: `(env var: 'UNITS')`.
- An empty `envvar`, `[]`, `()` or `""`, behaves exactly like `envvar=None`: no `(env var: …)` part in
  the error message, and no `[env var: …]` in the help.
- With an `auto_envvar_prefix`, an option without its own variables already shows the automatic one in
  the help (`[env var: WEATHER_UNITS]` for prefix `WEATHER`), and click reads that variable. An empty
  `envvar` must do the same, because click also falls back to the automatic variable for it when
  reading the value.
- Without `show_envvar`, nothing changes. The help for a non-empty `envvar` (`[env var: WEATHER_UNITS, UNITS]`)
  is already right and must stay as it is.
