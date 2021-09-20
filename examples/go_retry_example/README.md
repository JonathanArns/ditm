This is a small example to demonstrate how the default http client of Go
sends retries for idempotent requests, but only if the connection has been
used successfully before. This last detail of the behavior is not documented
in the official documentation.

The discrepancy was discovered with the help of ditm, which is why the
example is located in here.
