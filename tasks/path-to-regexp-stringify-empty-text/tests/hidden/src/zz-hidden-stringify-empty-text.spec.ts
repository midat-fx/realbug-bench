// Hidden tests: never shown to the agent.
import { describe, it, expect } from "vitest";
import { compile, parse, stringify, TokenData, type Token } from "./index.js";

const params = (type: "param" | "wildcard") => ({
  foo: type === "param" ? "value" : ["value", "more"],
});

describe.each(["param", "wildcard"] as const)("hidden: %s followed by empty text", (type) => {
  const sigil = type === "param" ? ":" : "*";

  it.each([
    [1, "bar", '"foo"bar'],
    [2, "bar", '"foo"bar'],
    [5, "bar", '"foo"bar'],
    [2, "\u{1D6FC}", '"foo"\u{1D6FC}'],
    [3, "1", '"foo"1'],
    [1, "$x", '"foo"$x'],
    [2, "/bar", "foo/bar"],
    [3, ".json", "foo.json"],
    [2, "", "foo"],
  ])("%i empty token(s) then %j", (empties, suffix, expected) => {
    const tokens: Token[] = [{ type: "text", value: "/" }, { type, name: "foo" }];
    for (let i = 0; i < empties; i++) tokens.push({ type: "text", value: "" });
    tokens.push({ type: "text", value: suffix });
    const data = new TokenData(tokens);

    const path = stringify(data);
    expect(path).toBe(`/${sigil}${expected}`);
    expect(compile(path)(params(type))).toBe(compile(data)(params(type)));
    expect(parse(path).tokens.filter((t) => t.type === type)).toEqual([{ type, name: "foo" }]);
  });

  it("trailing empty tokens leave the name unquoted", () => {
    const data = new TokenData([
      { type: "text", value: "/" },
      { type, name: "foo" },
      { type: "text", value: "" },
      { type: "text", value: "" },
    ]);
    expect(stringify(data)).toBe(`/${sigil}foo`);
  });

  it("applies inside groups", () => {
    const data = new TokenData([
      { type: "text", value: "/a" },
      {
        type: "group",
        tokens: [
          { type: "text", value: "/" },
          { type, name: "foo" },
          { type: "text", value: "" },
          { type: "text", value: "" },
          { type: "text", value: "bar" },
        ],
      },
    ]);
    const path = stringify(data);
    expect(path).toBe(`/a{/${sigil}"foo"bar}`);
    expect(compile(path)(params(type))).toBe(compile(data)(params(type)));
  });

  it("a group after the name ends it, empty tokens or not", () => {
    const data = new TokenData([
      { type: "text", value: "/" },
      { type, name: "foo" },
      { type: "text", value: "" },
      { type: "group", tokens: [{ type: "text", value: "bar" }] },
    ]);
    expect(stringify(data)).toBe(`/${sigil}foo{bar}`);
  });
});

describe("hidden: unchanged behaviour", () => {
  it.each([
    "/:foo",
    "/:foo/bar",
    '/:"foo"bar',
    "/*path",
    "/:a{-:b}",
    '/:"with space"',
    "/users/:id.json",
  ])("round-trips %j", (path) => {
    expect(stringify(parse(path))).toBe(path);
  });

  it("adjacent params", () => {
    const data = new TokenData([
      { type: "param", name: "a" },
      { type: "text", value: "" },
      { type: "param", name: "b" },
    ]);
    expect(stringify(data)).toBe(":a:b");
  });
});
