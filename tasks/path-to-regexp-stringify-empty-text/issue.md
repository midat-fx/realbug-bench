# `stringify()` merges a parameter name with the following text when an empty text token sits between them

`stringify()` quotes a parameter or wildcard name when the text right after it would otherwise be
read back as part of the name (`:"foo"bar`). The check only looks at the token immediately after the
parameter. Token data built programmatically (or by transforming parsed tokens) can contain empty
text tokens, and then the check is skipped:

```ts
import { TokenData, stringify, parse } from "path-to-regexp";

const data = new TokenData([
  { type: "text", value: "/" },
  { type: "param", name: "foo" },
  { type: "text", value: "" },
  { type: "text", value: "bar" },
]);

stringify(data);              // "/:foobar"  -> expected '/:"foo"bar'
parse(stringify(data)).tokens // one param named "foobar": a different route
```

The resulting path matches different URLs and compiles with different parameters than the token
data it came from.

Expected behaviour:

- Empty text tokens contribute nothing to the path, so the output must be the same as if they were
  absent: a name is quoted exactly when the first character that actually follows it continues an
  identifier. This applies to params and wildcards, and to tokens inside groups.
- Names are not quoted when they do not need to be (`/:foo/bar`, `/:foo` stay as they are).
- `parse(stringify(data))` describes the same route as `data`.
