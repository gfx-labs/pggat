## PostgreSQL blocks

Use Caddy's standard `caddyfile` adapter. PostgreSQL servers use `[pg]` blocks. Caddy HTTP server blocks can coexist in the same file.

```caddyfile
[pg] {
    stat_log_period 1m
}

[pg] :5433 {
    ssl self_signed

    pool /mydb {
        pool basic session
        address localhost:5432
        username postgres
        password postgres
        database mydb
    }
}

[http.server] example.com {
    respond "web traffic"
}
```

A keyless `[pg]` block configures pggat-wide options. `stat_log_period` defaults to `1m`. Each keyed `[pg]` block defines a listener; its keys become listener addresses. You can put the keyless options block before or after the listener blocks.

| Directive | Description |
| --- | --- |
| `stat_log_period` | pggat app log interval, accepted in keyless `[pg]` blocks |
| `ssl` | listener SSL configuration |
| `{handler}` | pggat handlers, evaluated in order |
