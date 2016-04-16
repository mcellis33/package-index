## Get Started

Requirements: Docker.

`./run.sh` to build a docker image tagged `package-index`. This image runs a
package index server on port 8080. `run.sh` also runs all of the unit tests,
functional tests, and benchmarks for the repo.

To run the package index without running the unit tests, functional tests, or
benchmarks, run

```
docker run -d package-index
```

For options, see

```
docker run --rm package-index /package-index --help
```

Note that depending on your docker setup, you will need to forward the server
port (by default 8080) from the container in order to access the service.
`run.sh` gives test-suite access to the package-index container with the
docker bridge network.

## Design Notes

Code should be as self-documenting as possible. See the code comments for more
details on tradeoffs.

The spec does not define a maximum message size. In order to prevent OOM
attacks, the server uses a buffer of bounded size to read each message from
the socket. If a message is too large, the server sends an error response.

The spec does not define a timeout for incomplete messages. Well-behaved
clients will close their sockets when they are done sending requests. To keep
poorly implemented clients from hogging connections, the server closes idle
connections.

To reduce memory pressure, the server pools message read buffers.

The asymptotic complexity of each operation, letting d be the number of
dependencies, is

* INDEX - O(d)
* REMOVE - O(d)
* QUERY - O(1)

