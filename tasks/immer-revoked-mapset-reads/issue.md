# Revoked Map and Set drafts still answer some reads after `produce` returns

Once `produce` (or `finishDraft`) returns, its drafts are revoked and any further use is supposed to
fail loudly with the "Cannot use a proxy that has been revoked" error. Plain object and array
drafts get this from `Proxy.revocable`. Map and Set drafts (with `enableMapSet()`) are subclasses
that check a `revoked_` flag manually, and several read paths never check it, so a leaked draft keeps
returning data:

```js
import { produce, enableMapSet } from "immer"
enableMapSet()

let leaked
produce({ users: new Map([["a", 1]]) }, draft => {
	leaked = draft.users
})

leaked.get("a")   // throws, as expected
leaked.has("a")   // true   -> should throw
leaked.size       // 1      -> should throw
leaked.keys()     // works  -> should throw
```

This hides bugs where a draft escapes into a callback or an async continuation: the code appears to
work and silently reads stale state, while the same mistake with an object draft is caught.

Expected behaviour:

- Every operation on a revoked Map or Set draft throws the same error as the other revoked
  drafts (`die(3, …)`, "Cannot use a proxy that has been revoked…"), whether or not the collection
  is empty and whether or not the draft was modified. This includes iteration and `delete` of a
  key that is not present.
- Nothing changes while the draft is alive, and the values `produce` returns are ordinary,
  fully usable Maps and Sets.
