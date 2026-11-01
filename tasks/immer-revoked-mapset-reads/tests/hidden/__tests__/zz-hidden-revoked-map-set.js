"use strict"
// Hidden tests: never shown to the agent.
import {produce, createDraft, finishDraft, enableMapSet} from "../src/immer"

enableMapSet()

const REVOKED = /has been revoked|minified error nr: 3/

function leak(base, modify) {
	let leaked
	const result = produce(base, draft => {
		leaked = draft.c
		if (modify) modify(draft.c)
	})
	return [leaked, result]
}

const MAP_READS = {
	has: m => m.has("a"),
	size: m => m.size,
	keys: m => m.keys(),
	values: m => m.values(),
	entries: m => m.entries(),
	iterator: m => m[Symbol.iterator](),
	spread: m => [...m],
	forEach: m => m.forEach(() => {}),
	get: m => m.get("a"),
	deleteMissing: m => m.delete("missing"),
	set: m => m.set("z", 1),
	clear: m => m.clear()
}

const SET_READS = {
	has: s => s.has(1),
	size: s => s.size,
	keys: s => s.keys(),
	values: s => s.values(),
	entries: s => s.entries(),
	iterator: s => s[Symbol.iterator](),
	forEach: s => s.forEach(() => {}),
	deleteMissing: s => s.delete(99),
	add: s => s.add(3),
	clear: s => s.clear()
}

describe("hidden: revoked Map draft", () => {
	const cases = {
		"unmodified": [{c: new Map([["a", 1], ["b", {x: 1}]])}],
		"modified": [{c: new Map([["a", 1]])}, m => m.set("b", 2)],
		"empty": [{c: new Map()}]
	}
	for (const [label, [base, modify]] of Object.entries(cases)) {
		describe(label, () => {
			test.each(Object.keys(MAP_READS))("%s throws the revoked error", op => {
				const [leaked] = leak(base, modify)
				expect(() => MAP_READS[op](leaked)).toThrow(REVOKED)
			})
		})
	}
})

describe("hidden: revoked Set draft", () => {
	const cases = {
		"unmodified": [{c: new Set([1, 2, {x: 1}])}],
		"modified": [{c: new Set([1])}, s => s.add(2)],
		"empty": [{c: new Set()}]
	}
	for (const [label, [base, modify]] of Object.entries(cases)) {
		describe(label, () => {
			test.each(Object.keys(SET_READS))("%s throws the revoked error", op => {
				const [leaked] = leak(base, modify)
				expect(() => SET_READS[op](leaked)).toThrow(REVOKED)
			})
		})
	}
})

describe("hidden: finishDraft revokes too", () => {
	test("map", () => {
		const draft = createDraft({c: new Map([["a", 1]])})
		const m = draft.c
		finishDraft(draft)
		expect(() => m.size).toThrow(REVOKED)
		expect(() => m.has("a")).toThrow(REVOKED)
		expect(() => m.keys()).toThrow(REVOKED)
	})
	test("set", () => {
		const draft = createDraft({c: new Set([1])})
		const s = draft.c
		finishDraft(draft)
		expect(() => s.size).toThrow(REVOKED)
	})
	test("root map draft", () => {
		const m = createDraft(new Map([["a", 1]]))
		finishDraft(m)
		expect(() => m.size).toThrow(REVOKED)
		expect(() => m.forEach(() => {})).toThrow(REVOKED)
	})
})

describe("hidden: live drafts and results are unaffected", () => {
	test("reads inside the producer", () => {
		const base = {m: new Map([["a", 1], ["b", 2]]), s: new Set([1, 2])}
		const seen = []
		const result = produce(base, d => {
			seen.push(d.m.size, d.m.has("a"), [...d.m.keys()], d.s.size, d.s.has(2))
			d.m.delete("b")
			d.m.forEach((v, k) => seen.push([k, v]))
			d.s.add(3)
			seen.push(d.m.size, [...d.s])
		})
		expect(seen).toEqual([2, true, ["a", "b"], 2, true, ["a", 1], 1, [1, 2, 3]])
		expect(result.m.size).toBe(1)
		expect(result.m.has("a")).toBe(true)
		expect([...result.m.keys()]).toEqual(["a"])
		expect(result.s.size).toBe(3)
		expect([...result.s]).toEqual([1, 2, 3])
	})

	test("unmodified result is the base and stays usable", () => {
		const base = {m: new Map([["a", 1]])}
		const [leaked, result] = leak({c: base.m})
		expect(result.c).toBe(base.m)
		expect(result.c.size).toBe(1)
		expect(() => leaked.size).toThrow(REVOKED)
	})

	test("nested producers", () => {
		const base = {m: new Map([["k", new Map([["x", 1]])]])}
		const result = produce(base, d => {
			const inner = d.m.get("k")
			d.m.set("n", produce(new Map(), d2 => void d2.set("y", inner.size)))
		})
		expect(result.m.get("n").get("y")).toBe(1)
	})
})
