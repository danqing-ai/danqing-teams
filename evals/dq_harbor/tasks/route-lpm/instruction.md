`/app/routes.txt` is an IPv4 routing table. Each line: `<cidr> via <nexthop>`
Example: `10.1.2.0/24 via C`

`/app/queries.txt` has one IPv4 address per line.

For each query, choose the matching route with the **longest prefix** (highest prefix length). Ties do not occur in the fixture. Every address matches at least the default route if present.

Write `/app/answers.txt` with one line per query, same order:
`<ip> -> <nexthop>`

Do not ask the user any questions.
