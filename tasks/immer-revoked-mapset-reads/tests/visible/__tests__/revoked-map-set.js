"use strict"
import {produce, enableMapSet} from "../src/immer"

enableMapSet()

describe("revoked map and set drafts", () => {
	test("map reads that do not go through get", () => {
		let leaked
		produce({map: new Map([["a", 1]])}, draft => {
			leaked = draft.map
		})
		expect(() => leaked.has("a")).toThrow()
		expect(() => leaked.size).toThrow()
	})

	test("set size", () => {
		let leaked
		produce({set: new Set([1, 2])}, draft => {
			leaked = draft.set
		})
		expect(() => leaked.size).toThrow()
	})
})
