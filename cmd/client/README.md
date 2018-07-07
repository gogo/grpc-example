## Client example

This shows a small example of using a client to connect to the gRPC server.
It shows how a number of important things:

1. How to connect securely to a server, provided you have access to the server Certificate.
1. How to parse a gRPC status error message with an error Detail.
1. How to consume all messages on a server side stream.

Note that running the client twice without restarting the server inbetween
will make it fail the second time, since a user is added to the server in the
first call, and can't be added again.
