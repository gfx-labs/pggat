# Rob
Rob is a fair-share (used to be work-stealing, now uses a master queue to be simpler) scheduler for pggat. It is modeled
after the completely fair scheduler from linux. The previous scheduler for pggat were first-come, first-serve, meaning
that users making longer requests were over-represented. This new scheduler should be able to balance between users, 
giving a better overall experience.

If you want to take a shot at improving or rewriting the scheduler, put it in the next version folder
(e.g. `schedulers/v1`)

###  References
- https://tsung-wei-huang.github.io/papers/icpads20.pdf
