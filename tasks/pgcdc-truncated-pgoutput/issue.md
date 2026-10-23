# pgoutput decoders panic on truncated messages instead of returning an error

**Component:** `pq/message` (`message.New`, `pq/message/format`, `pq/message/tuple`)

A pgoutput payload that is shorter than its format requires makes the decoder panic instead of
returning an error the caller could handle. For example, a Relation
message cut right after the table name:

```
panic: runtime error: index out of range [25] with length 25
```

The decoders read fixed-width fields (`binary.BigEndian.Uint32(data[i:])`, `data[i]`) without
checking that the bytes are there.

Places we found so far:

- `message.New` indexes `data[0]` before checking that `data` is non-empty.
- `format.NewBegin` accepts 20 bytes, but a Begin message is 21 bytes (type + LSN + timestamp + xid),
  so the xid read runs past the end. Same off-by-one in `format.NewCommit` (needs 26).
- `tuple.NewData` / `(*tuple.Data).Decode` trust the column count and the per-column length
  prefix, so a tuple that ends early panics.
- `format.NewUpdate`: when the old tuple has fewer columns than the new tuple, copying an
  unchanged TOASTed column from the old tuple indexes past the old tuple.

There may be more — please audit every decoder reachable from `message.New`, not just these.

## Expected

- Any message that is shorter than its format requires (truncated anywhere, including in the
  middle of a column or a column name) makes `message.New` **and** the individual decoders
  (`format.New*`, `tuple.NewData`) return an error. Nothing in `pq/message` may panic on short input.
- A Relation message that fails to decode must not be stored in the relation cache.
- Well-formed messages decode exactly as before.
- Keep the existing error wording style. The length errors for Begin/Commit read
  `begin message length must be at least 21 byte, but got N` /
  `commit message length must be at least 26 byte, but got N`, and the tuple errors contain
  `tuple data length`.

The existing tests in `pq/message/...` show the expected behaviour for the cases above.
