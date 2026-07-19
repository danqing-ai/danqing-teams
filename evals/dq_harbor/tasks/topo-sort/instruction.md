`/app/deps.txt` lists directed dependencies, one per line as `A -> B` meaning A must come **before** B.

Produce a valid topological order of all nodes that appear in the file, written to `/app/order.txt` (one node per line).
If multiple orders are valid, choose the one that is **lexicographically smallest** when compared as a sequence of lines
(i.e. prefer smaller node names as early as possible — Kahn's algorithm with a min-heap / sorted ready set).

The graph is a DAG (no cycles). Do not ask the user any questions.
