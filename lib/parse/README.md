parsing sql in a single pass since 4:21pm on thursday

a basic SQL query parser (specifically postgres)

this is **not** intended to parse arguments correctly or verify sql commands, the only goal of this is to split an sql query into its statements in a single pass. Arguments are only split by spaces and may be wrong if they have operators, function calls, etc.