import { describe, it, expect } from "vitest";
import { compile, stringify, TokenData } from "./index.js";

describe("stringify param followed by an empty text token", () => {
  it("quotes the name when the next non-empty text continues it", () => {
    const data = new TokenData([
      { type: "text", value: "/" },
      { type: "param", name: "foo" },
      { type: "text", value: "" },
      { type: "text", value: "bar" },
    ]);
    const path = stringify(data);

    expect(path).toBe('/:"foo"bar');
    expect(compile(path)({ foo: "x" })).toBe(compile(data)({ foo: "x" }));
  });
});
