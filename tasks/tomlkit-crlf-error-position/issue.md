# Parse errors report the wrong line and column for files with Windows (CRLF) line endings

When a TOML document uses `\r\n` line endings, the position in a `ParseError` drifts: every CRLF
line before the error shifts the reported position by one character, so the error lands on the
wrong line, often with column 0.

```python
>>> import tomlkit
>>> tomlkit.parse("a = 1\nb = 2\nc = @\nd = 4\n")
UnexpectedCharError: Unexpected character: '@' at line 3 col 4        # correct
>>> tomlkit.parse("a = 1\r\nb = 2\r\nc = @\r\nd = 4\r\n")
UnexpectedCharError: Unexpected character: '@' at line 4 col 0        # should be line 3 col 4
```

On a real `pyproject.toml` checked out on Windows the message points tens of lines away from the
problem. Files that mix `\n` and `\r\n` (common after copy-pasting between editors) are wrong too.

Expected behaviour:

- `ParseError.line` / `.col` (and the message) describe the same position whatever the line endings:
  a document with `\r\n` or mixed endings reports exactly what the same document with `\n` reports.
- Documents with `\n` line endings keep reporting exactly the positions they report today,
  including errors at the very end of the input.
- Parsing must keep the document intact: `tomlkit.parse(text).as_string() == text` for CRLF input.
