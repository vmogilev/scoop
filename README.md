# scoop
scoop: package indexer (this is my solution to a coding challenge)

## Problem Definition
Develop a package indexer that keeps track of package dependencies. Clients will connect to this server and inform which packages should be indexed, and which dependencies they might have on other packages.

The server will open a TCP socket on port 8080. It must accept connections from multiple clients at the same time, all trying to add and remove items to the index concurrently. Clients are independent of each other, and it is expected that they will send repeated or contradicting messages. New clients can connect and disconnect at any moment, and sometimes clients can behave badly and try to send broken messages.

Messages from clients follow this pattern:

```
<command>|<package>|<dependencies>\n
```

Where:
* `<command>` is mandatory, and is either `INDEX`, `REMOVE`, or `QUERY`
* `<package>` is mandatory, the name of the package referred to by the command, e.g. `mysql`, `openssl`, `pkg-config`, `postgresql`, etc.
* `<dependencies>` is optional, and if present it will be a comma-delimited list of packages that need to be present before `<package>` is installed. e.g. `cmake,sphinx-doc,xz`
* The message always ends with the character `\n`

Here are some sample messages:
```
INDEX|cloog|gmp,isl,pkg-config\n
INDEX|ceylon|\n
REMOVE|cloog|\n
QUERY|cloog|\n
```

For each message sent, the client will wait for a response code from the server. Possible response codes are `OK\n`, `FAIL\n`, or `ERROR\n`. After receiving the response code, the client can send more messages.

The response code returned should be as follows:
* For `INDEX` commands, the server returns `OK\n` if the package can be indexed. It returns `FAIL\n` if the package cannot be indexed because some of its dependencies aren't indexed yet and need to be installed first. If a package already exists, then its list of dependencies is updated to the one provided with the latest command.
* For `REMOVE` commands, the server returns `OK\n` if the package could be removed from the index. It returns `FAIL\n` if the package could not be removed from the index because some other indexed package depends on it. It returns `OK\n` if the package wasn't indexed.
* For `QUERY` commands, the server returns `OK\n` if the package is indexed. It returns `FAIL\n` if the package isn't indexed.
* If the server doesn't recognize the command or if there's any problem with the message sent by the client it should return `ERROR\n`.

## Problem Solution (Design Rationale)
Scoop's design follows Idiomatic `Go` concurrentcy pattern.  There are three background processes:

- Server
- Listener
- Worker

**Listener** is accepting client connections, parses client commands to messages and sends them (concurently) to the Worker via a channel.  

**Worker** is constantly pooling this channel and serializes access to a shared data structure.  Responses from the worker are sent back to the clients over a another channel embedded in the message itself.

**Server** implements graceful shutdown by listening to `syscall.SIGTERM` and `os.Interrupt`.  On shutdown, the server caches the shared data structure to disk and then reads it during startup.  It also implements a lock to ensure no two instances operate on the same cached data store concurrently.

End.