# How it works
All tests are listed in the `tests` directory. They are each run line by line against a real postgres
database and a database proxied through pggat. If the output differs in any meaningful way, the test
will fail.

# Running without a database
The tests can be run without a postgres database by using previous test results in place of the
database.

# Test format
The tests are formatted as a set of "instructions". Each instruction corresponds to zero or more packets
to be sent to the server.

## Instructions
| Instruction | Arguments      | Description                                                                      |
|-------------|----------------|----------------------------------------------------------------------------------|
| PX          | bool           | Controls whether the next instructions can be run in parallel                    |
| SQ          | string         | Runs a simple query                                                              |
| QA          | string, ...any | Runs a query with arguments. This will run a prepare, bind, explain, and execute |

## Parallel tests
By default, many instances of a single test will be run at the same time. If the test or parts of the test
cannot be run in parallel, prepend `PX false`.
