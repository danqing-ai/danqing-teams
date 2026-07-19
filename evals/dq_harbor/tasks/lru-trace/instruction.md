`/app/ops.txt` describes an LRU cache simulation.

Format (one command per line):
- `CAPACITY N` — appears once, first line. Cache holds at most N keys.
- `PUT key value` — insert or update `key` with integer `value`. On update, move key to most-recently-used (MRU). If inserting a new key would exceed capacity, evict the least-recently-used (LRU) key first.
- `GET key` — if present, move key to MRU (counts as a **hit**); if absent, do nothing to the cache (counts as a **miss**).

`PUT` does **not** affect hit/miss counters.

Write `/app/result.txt` with exactly two lines:
1. Keys currently in the cache from **MRU to LRU**, comma-separated (no spaces). If empty, write an empty first line.
2. `hits=H misses=M` with the GET hit/miss counts.

Do not ask the user any questions.
