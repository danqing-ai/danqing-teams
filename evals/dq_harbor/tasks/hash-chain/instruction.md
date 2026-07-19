`/app/chain.json` is a list of blocks: `{"idx":N,"data":"...","prev":"<hex>","hash":"<hex>"}`.

Hash rule: `hash = sha256( prev + data )` as lowercase hex ASCII, where for the genesis block `prev` is 64 zero hex digits.

One block's `hash` field is wrong (corruption). Fix **only corrupted `hash` fields** (recompute from the rule) and write the corrected chain to `/app/fixed_chain.json` (JSON array, same order/fields). Compact or pretty JSON OK.

Do not ask the user any questions.
