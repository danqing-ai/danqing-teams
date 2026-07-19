`/app/sales.csv` has columns: `region,product,amount` (header + rows).

Pivot into `/app/pivot.csv` with:
- first column header `region`
- then one column per distinct product name (sorted alphabetically)
- cell = sum of amounts for that region×product (integer; missing = `0`)
- rows sorted by region ascending
- CSV with header, no spaces around commas

Do not ask the user any questions.
