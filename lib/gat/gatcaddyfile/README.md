## Server Blocks
Server blocks will be matched in order by their keys. Different SSL configurations may not be used on the same host port pair
(due to technical limitations).

## Directives
| Directive        | Description                                                              |
|------------------|--------------------------------------------------------------------------|
| ssl              | ssl configuration for this server                                        |
| allow_parameters | set which initial parameters are allowed                                 |
| user             | rewrite username                                                         |
| password         | use global password instead of password provided by pool                 |
| database         | rewrite database                                                         |
| parameters       | rewrite parameters                                                       |
| {provider}       | a pool provider. if pool is not found, the next provider will be checked |
