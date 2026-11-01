# Flow collection used as a block map key fails to parse after a blank line or a comment

A block mapping may use a flow sequence or flow mapping as a key (`[a, b]: value`). That works when
the key directly follows the previous entry, but if a blank line or a comment line sits in between,
parsing fails with an internal error:

```js
import { parseDocument } from 'yaml'

parseDocument('1: 2\n[3]: 4').toJS({ mapAsMap: true })   // Map { 1 => 2, [3] => 4 }   OK
parseDocument('1: 2\n\n[3]: 4').errors                    // [ YAMLParseError IMPOSSIBLE ... ]
parseDocument('1: 2\n# note\n[3]: 4').errors              // [ YAMLParseError IMPOSSIBLE ... ]
```

The same happens for `{a: 1}: x` keys and inside nested block maps. Blank lines and comments
between entries are everywhere in hand-written YAML, so a file that is valid YAML 1.2 is rejected
just because of its formatting.

Expected behaviour:

- A flow-collection key parses the same with or without blank lines or comment lines before it, at
  any nesting level, and the entries before and after it are unaffected.
- Comments and blank lines are kept: `String(doc)` still contains them.
- Everything that parses today parses to the same result, including explicit keys (`?`), empty
  explicit keys and block sequences of maps.
