# tcpw

`tcpw` (TCP Wait) is a tool that waits until TCP endpoints are open.

It is useful for synchronizing the spin-up of interdependent services, such as linked docker containers.

That is yet another alternative to `wait-for-it.sh`.

## Rationale

- available as a binary executable without any dependencies (file size <1mb).
  - Optimized with: `-ldflags="-s -w" -trimpath` and `upx`
  - Pre-built binaries:
    - [tcpw-linux-x86_64](https://raw.githubusercontent.com/jackcvr/tcpw/main/tcpw/tcpw-x86_64)
    - [tcpw-linux-aarch64](https://raw.githubusercontent.com/jackcvr/tcpw/main/tcpw/tcpw-aarch64)
    - [tcpw-linux-armv7l](https://raw.githubusercontent.com/jackcvr/tcpw/main/tcpw/tcpw-armv7l)
- available on `PyPI`
- additionally, you can set:
    - more than one endpoint: `-a google.com:80 -a booble.gum:8080 ...`
    - command, which can be executed only after success, failure or any result: `-on f -a google.com:9999 echo "Endpoint is down"`
    - polling interval: `-i 500ms`
    - `timeout/interval` in different time units: `ns,ms,s,m,h`


## Installation

Download executable file:

`sh -c "wget -O tcpw https://raw.githubusercontent.com/jackcvr/tcpw/main/tcpw/tcpw-$(uname -m) && chmod +x tcpw"`

or:

`pip install tcpw`

[![PyPI - Version](https://img.shields.io/pypi/v/tcpw.svg)](https://pypi.org/project/tcpw)
[![PyPI - Python Version](https://img.shields.io/pypi/pyversions/tcpw.svg)](https://pypi.org/project/tcpw)

## Usage

```text
Usage: tcpw [-t timeout] [-i interval] [-on (s|f|any)] [-q] [-v] [-a host:port ...] [command [args]]

  -a value
    	Endpoint to await, in the form 'host:port'
  -i duration
    	Interval between retries in format N{ns,ms,s,m,h} (default 1s)
  -on string
    	Condition for command execution. Possible values: 's' - after success, 'f' - after failure, 'any' - always (default 's') (default "s")
  -q	Do not print anything (default false)
  -t duration
    	Timeout in format N{ns,ms,s,m,h}, e.g. '5s' == 5 seconds. Zero for no timeout (default 0)
  -v	Verbose mode (default false)
  command args
    	Execute command with arguments after the test finishes (default: if connection succeeded)
```

## Examples

Wait 5 seconds for port 80 on `www.google.com`, and if it is available, echo the message `Google is up`:

```bash
$ tcpw -t 5s -a www.google.com:80 echo "Google is up"
2024/08/26 20:06:47.209012 successfully connected to www.google.com:80
Google is up
```

Next command waits 2 seconds for www.google.com:80 and localhost:5000, checking them every 500 milliseconds
with enabled verbose mode and executes `echo` regardless of the result:

```bash
$ tcpw -t 2s -i 500ms -v -on any -a www.google.com:80 -a localhost:5000 echo "Printed anyway"
2024/08/26 20:08:24.153240 connecting to localhost:5000...
2024/08/26 20:08:24.153327 connecting to www.google.com:80...
2024/08/26 20:08:24.153541 dial tcp 127.0.0.1:5000: connect: connection refused
2024/08/26 20:08:24.179927 successfully connected to www.google.com:80
2024/08/26 20:08:24.654984 dial tcp 127.0.0.1:5000: connect: connection refused
2024/08/26 20:08:25.155997 dial tcp 127.0.0.1:5000: connect: connection refused
2024/08/26 20:08:25.661397 dial tcp 127.0.0.1:5000: connect: connection refused
2024/08/26 20:08:26.161613 dial tcp: lookup localhost: i/o timeout
timeout error
Printed anyway
```

## License

[MIT](https://spdx.org/licenses/MIT.html) 