

# pggat

![image](https://github.com/user-attachments/assets/a4e7881f-fc5a-4349-b141-3148aff0f09f)

pggat is a Postgres pooler similar to PgBouncer. It is different in that it supports load balancing to rdwr/rd replicas.

the name is because [this song is a banger](https://www.youtube.com/watch?v=-DqCc2DJ0sg)




## Pooling Modes
There are currently two pooling modes which compromise between balancing and feature support. Most apps should work out of the box with transaction pooling.

### Transaction Pooling (default)
Send each transaction to a new node. This mode supports all postgres features that do not rely on session state (plus a few exceptions noted below).

This is similar to PgBouncer's transaction pooling except we additionally support protocol level prepared statements and all parameters (they may change at unexpected times, but clients should be able to handle this)

Using LISTEN commands in this mode will lead to undefined behavior (you may not receive the notifications you want, and you may receive notifications you did not ask for).

### Session Pooling
Send each session to a new node. This mode supports all postgres features, but will not balance as well unless clients make new sessions often.

## Unsupported features
One day these will maybe be supported
- Reserve pool (for serving long-stalled clients)
- Auth methods other than plaintext, MD5, and SASL-SCRAM-SHA256
- GSSAPI
- Timeouts (other than transaction idle timeout)
- Statement Pooling (probably won't add, the benefit over transaction pooling is negligible and compatibility suffers greatly)
- pgbouncer stats database (probably won't add, a lot of work for something that can be done more easily by other means like prometheus)
