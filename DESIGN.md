# Design

## User experience

TLS certificates are expected to be generated out-of-band ahead of time.

### Starting the server

Options:

* `-c`/`--certificate`: takes a path to the server's TLS certificate.
* `-k`/`--key`: takes a path to the server's TLS key.
* `-C`/`--certificate-authority`: takes a path to a CA certificate to trust.
* `-a`/`--address`: takes an address to listen on.
* `-p`/`--port`: takes a port number to listen on.

Example command:

```console
server -c cert.pem -k key.pem -C ca.pem -a ::1 -p 8000
```

Ideally, the server would be instrumented with OpenTelemetry for observability,
and log events like processes starting, connection attempts being rejected, and
so on, but this may not be done to limit the scope for time.

### Common client options

* `-c`/`--certificate`: takes a path to the client's TLS certificate.
* `-k`/`--key`: takes a path to the client's TLS key.
* `-C`/`--certificate-authority`: takes a path to a CA certificate to trust.
* `-a`/`--address`: takes an address to connect to.
* `-p`/`--port`: takes a port number to connect to.

These options are available for all client commands.

Ideally, clients would implement trust-on-first-use to improve the UX, but this
is not done in the interest of time.

### Starting processes

Example command:

```console
client $COMMON_OPTIONS start echo 'Hello, world!'
```

Any arguments after `start` are passed to the server. `--` is frequently also
used by convention in cases like this to indicate "end of options", but in this
case `start` will also fulfill this role and not requiring `--` is less typing.

Once the server has started the process, the client will print out the ID that
was assigned by the server to the new process and then exit.

If the client has insufficient permissions to start a process, an error message
will be printed indicating that this is the case.

### Checking the status of a process

Example command:

```console
client $COMMON_OPTIONS status 4
```

The `status` subcommand takes the process ID to query as its single argument.

* If the process is still running, it will print `running`.
* If the process has exited normally, it will print `exited: 0` (where `0` will
  be the process's exit code).
* If the process was stopped by a client's request, it will print `stopped`.
  (See also `func (WaitStatus) Signaled`.)
* If the process does not exist, it will print `process not found`.

### Stopping a process

Example command:

```console
client $COMMON_OPTIONS stop 4
```

The `stop` subcommand takes the process ID to stop as its single argument.

* If the process ID does exist, it will print `process stopped` once the process
  has been stopped.
* If the process ID does not exist, it will print `process not found`.
* If the client has insufficient permissions to stop a process, an error
  message will be printed indicating that this is the case.

Ideally, there would be a system for gracefully stopping processes by sending
SIGINT, SIGTERM, and SIGKILL in that order over some time interval if the
process did not stop quickly enough. This will not be implemented in the
interest of time, processes will be sent a single signal and will be expected to
terminate as a result.

### Streaming the output of a process

Example command:

```console
client $COMMON_OPTIONS stream 4
```

The `stream` subcommand takes the process ID whose output to stream as its
single argument. `stdout` and `stderr` of the remote process will be forwarded
to `stdout` and `stderr` of the client process respectively. The output of the
remote process will be sent to the client since the process was started rather
than since the client began streaming the output.

If the process does not exist, it will print `process not found`.

## Implementation details

### mTLS

The client and server will use mTLS to authenticate each other, and to allow
the server to authorize the client. Clients with unknown certificates will be
rejected by the server.

Clients will generate their own private key followed by a certificate signing
request (CSR). The CSR will set `CN` to some identifier for the client and `OU`
to either `observe` or `control`, which will be explained in the next section.
The CSR will then be sent to the certificate authority (CA) which will either
reject the CSR due to an incorrect `OU` for the particular client or accept it
and respond with a certificate.

The server will generate its own private key followed by a CSR with `CN` set
to `server`. The CA will accept this CSR and respond with a certificate. (In
practice, we'd probably want `CN` to be the hostname clients would be expected
to connect to, but just doing `server` in all cases will make testing easier.)

The server and clients will trust the same CA so that they may interoperate and
validate each others' certificates.

The minimum supported TLS version shall be 1.3, which in turn requires a strong
set of cipher suites.

### Authorization

Clients will have two authorization levels:

1. `observe`: This level allows a client to use the `status` and `stream`
   subcommands, but not the `start` and `stop` subcommands.
2. `control`: This level allows a client to use the `status`, `stream`, `start`,
   and `stop` subcommands.

The authorization level is encoded in the `OU` field in the client certificate,
the value of which is controlled by the certificate authority. The server will
read this value to determine the client's authorization level.

### Output streaming

Output from processes managed by the server will be stored in-memory. Ideally,
a file-backed solution would be used, but this is not done in the interest of
time. Storing output is necessary to provide the desired "output starts from the
beginning of the process rather than from when the client connected" semantics.

Each process's output stream will have a buffer with a read-write lock around
it. When `read(2)`ing from a pipe returns, the write lock will be taken to copy
the new output into the buffer, then released. A read lock will be taken out
when callers of `func (*ProcessManager) Stream` want to read from the buffer,
then released. When the `Readers` returned by `(*ProcessManager) Stream` hit
EOF for the first time, it means the caller has caught up with all history in
the buffer. Rather than returning this EOF to the caller, the current call and
further calls will block while waiting for new process output. To facilitate
this, a condition variable will begin being waited on before the `Stream`
`Readers` attempt further reads, and the `ProcessManager` will broadcast on
the condition variable after it completes a new read from the pipe. Finally,
the `ProcessManager` will signal to the `Reader`s returned by `Stream` that the
pipe has been closed (e.g. due to the process terminating) by setting an atomic
boolean value, at which point the `Stream` `Reader`s will return EOF to their
callers once they have read the whole buffer. At this point, reads starting
from the beginning of the history can simply check the atomic boolean value and
return EOF after the buffer has finished being read out without involving the
condition variable.

### Library

A library will be implemented in `pkg/` that exposes the necessary functionality
for starting, stopping, querying, and streaming data from processes. Aside from
managing the processes themselves, it will not do any IO internally, it will be
up to the caller (e.g. the server) to decide what to do with streamed output.

### Reproducibility

Nix (in particular, [Lix]) will be used to ensure repeatability of builds and
development environments. A NixOS module could also be provided for the server
component, but this will not be done in the interest of time. [NixOS tests]
could be used for integration testing the NixOS module, server, and client at
once in an isolated environment (namely, one or more virtual machines).

[Lix]: https://lix.systems/
[NixOS tests]: https://nixos.org/manual/nixos/stable/#sec-nixos-tests
