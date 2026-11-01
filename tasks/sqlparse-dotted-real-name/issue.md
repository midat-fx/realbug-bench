# `get_real_name()` returns the wrong component for names with three or more dotted parts

For a qualified identifier, `Identifier.get_real_name()` (and therefore `get_name()` when there is
no alias) should return the object name: the last component. That works for `schema.table`, but as
soon as the name has three or more parts it returns an intermediate component instead:

```python
>>> import sqlparse
>>> ident = sqlparse.parse("db.schema.orders")[0].tokens[0]
>>> ident.get_real_name()
'schema'          # expected 'orders'
>>> sqlparse.parse("select o.id from db.schema.orders o")[0].tokens[-1].get_real_name()
'schema'          # expected 'orders'
```

Tools that use sqlparse to find which tables a query reads (lineage, permission checks) report
the schema as the table for every fully-qualified name, e.g. `warehouse.sales.orders` or
SQL Server's `server.db.dbo.table`.

Expected behaviour:

- `get_real_name()` returns the component after the **last** dot, for any number of parts, with the
  usual quote handling (`"a"."b"."c"`, `` `a`.`b`.`c` ``, `[a].[b].[c]`), with or without an alias,
  and for qualified function calls (`db.schema.fn(1)` → `fn`).
- `get_parent_name()` is documented as "identified by the first occurring dot" and must keep
  returning the first component (`db` for `db.schema.orders`). Code relies on that.
- Two-part names and aliases (`get_alias()`) behave exactly as before.
