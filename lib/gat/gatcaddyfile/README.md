## Server Blocks
Server blocks will be matched in order by their keys. Different SSL configurations may not be used on the same host port pair
(due to technical limitations).

## Directives
| Directive | Description                       |
|-----------|-----------------------------------|
| ssl       | ssl configuration for this server |
| {handler} | each handler will be run in order |
