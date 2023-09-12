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
