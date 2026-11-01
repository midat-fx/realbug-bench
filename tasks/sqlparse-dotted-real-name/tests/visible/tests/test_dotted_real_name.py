import sqlparse


def test_real_name_of_three_part_name():
    ident = sqlparse.parse("db.schema.orders")[0].tokens[0]
    assert ident.get_real_name() == "orders"
    assert ident.get_name() == "orders"


def test_real_name_of_three_part_name_with_alias():
    ident = sqlparse.parse("db.schema.orders AS o")[0].tokens[0]
    assert ident.get_real_name() == "orders"
    assert ident.get_alias() == "o"


def test_two_part_name_unchanged():
    ident = sqlparse.parse("schema.orders")[0].tokens[0]
    assert ident.get_real_name() == "orders"
    assert ident.get_parent_name() == "schema"
