# `go-remote`

A tool for spawning and managing processes over the network in Go.

## Usage

First, run the `gen-proto` script in the `bin` directory with the repository
root as the current working directory.

Next, run the `gen-certs` script in the `bin` directory with an argument of a
directory to create and write certs into. For example, you might run it twice
with `gen/tls-a` and `gen/tls-b` as arguments to test clients and servers with
mismatching CAs. (A subdirectory of the `gen` directory in particular is a good
choice since that directory is `.gitignore`'d.)

Run the server with a command like the following:

```console
go run cmd/server/main.go \
  -a ::1 \
  -p 8888 \
  -C path/to/ca.pem \
  -c path/to/server.pem \
  -k path/to/server.key
```

Run the client with a command like the following for observe privileges:

```console
go run cmd/client/main.go \
  -a ::1 \
  -p 8888 \
  -C path/to/ca.pem \
  -c path/to/client-observe.pem \
  -k path/to/client-observe.key \
  ...
```

Or like the following for control (and observe, implied by control) privileges:

```console
go run cmd/client/main.go \
  -a ::1 \
  -p 8888 \
  -C path/to/ca.pem \
  -c path/to/client-control.pem \
  -k path/to/client-control.key \
  ...
```

In place of `...`, the client takes one of a few subcommands:

* `start ...`: start a process on the server, where `...` is the process's
  command line. For example, `start echo 'Hello, world!'` will run the `echo`
  command on the server with `Hello, world!` as its first argument (and `echo`
  as its zeroth argument). On success, the client will print out a process ID
  for future use. This command requires control privileges.
* `stop <id>`: stop a process on the server, where `<id>` is a process ID
  printed by a previous successful `start` command. This command requires
  control privileges.
* `status <id>`: get the status of a process on the server, where `<id>` is
  a process ID printed by a previous successful `start` command. This command
  requires observe privileges.
* `stream <id>`: stream the output of a process on the server since that process
  was started, where `<id>` is a process ID printed by a previous successful
  `start` command. `stdout` and `stderr` of the process will be forwarded
  to `stdout` and `stderr` of the client respectively. This command requires
  observe privileges.
