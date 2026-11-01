# Hidden tests: never shown to the agent.
import pytest

import sqlparse
from sqlparse import sql


def first(s):
    return sqlparse.parse(s)[0].tokens[0]


@pytest.mark.parametrize(
    ("text", "real", "parent"),
    [
        ("db.schema.tbl.col", "col", "db"),
        ("a.b.c.d.e.f", "f", "a"),
        ("srv.db.dbo.tbl", "tbl", "srv"),
        ("x.y.z", "z", "x"),
        ("a.b", "b", "a"),
    ],
)
def test_hidden_last_component_any_depth(text, real, parent):
    ident = first(text)
    assert ident.get_real_name() == real
    assert ident.get_name() == real
    assert ident.get_parent_name() == parent


@pytest.mark.parametrize(
    ("text", "real", "parent"),
    [
        ('"my.db"."my.schema"."my.tbl"', "my.tbl", "my.db"),
        ('"a"."b"."c"."d"', "d", "a"),
        ("`d`.`s`.`t`", "t", "d"),
        ("[db].[dbo].[t]", "[t]", "[db]"),
    ],
)
def test_hidden_quoted_components(text, real, parent):
    ident = first(text)
    assert ident.get_real_name() == real
    assert ident.get_parent_name() == parent


@pytest.mark.parametrize(
    ("text", "real", "alias"),
    [
        ("x.y.z AS w", "z", "w"),
        ("x.y.z w", "z", "w"),
        ("srv.db.dbo.tbl AS t", "tbl", "t"),
        ("`d`.`s`.`t` as q", "t", "q"),
    ],
)
def test_hidden_alias(text, real, alias):
    ident = first(text)
    assert ident.get_real_name() == real
    assert ident.get_alias() == alias
    assert ident.get_name() == alias


def test_hidden_function_and_cast():
    assert first("mydb.sch.func(1)").get_real_name() == "func"
    assert first("a.b.c::int").get_real_name() == "c"


def test_hidden_wildcard_matches_two_part_behaviour():
    assert first("t.*").get_real_name() == "*"
    assert first("db.sch.tbl.*").get_real_name() == "*"


def test_hidden_identifiers_inside_statement():
    stmt = sqlparse.parse("select t.c from db.sch.tbl as t join d2.s2.t2 u on 1=1")[0]
    got = [
        (i.get_real_name(), i.get_parent_name(), i.get_alias())
        for i in stmt.get_sublists()
        if isinstance(i, sql.Identifier)
    ]
    assert got == [("c", "t", None), ("tbl", "db", "t"), ("t2", "d2", "u")]


def test_hidden_identifier_list():
    stmt = sqlparse.parse("select a.b.c, d.e.f.g as h from t")[0]
    il = next(i for i in stmt.get_sublists() if isinstance(i, sql.IdentifierList))
    got = [(i.get_real_name(), i.get_parent_name(), i.get_alias()) for i in il.get_identifiers()]
    assert got == [("c", "a", None), ("g", "d", "h")]
