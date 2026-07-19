`/app/bsearch.py` is supposed to implement iterative binary search:
`python3 /app/bsearch.py <sorted_ints_csv> <target>` prints the **0-based index** of target, or `-1` if absent.

It currently has bugs. Fix `/app/bsearch.py` in place so these hold:
- `python3 /app/bsearch.py 1,3,5,7,9 7` → `3`
- `python3 /app/bsearch.py 1,3,5,7,9 2` → `-1`
- `python3 /app/bsearch.py 1,3,5,7,9 1` → `0`
- `python3 /app/bsearch.py 1,3,5,7,9 9` → `4`

Do not ask the user any questions.
