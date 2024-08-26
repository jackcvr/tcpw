package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"golang.org/x/sync/errgroup"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"
)

var (
	timeout    time.Duration
	interval   time.Duration
	quiet      bool
	verbose    bool
	endpoints  StringList
	on         string
	command    []string
	flagOutput = flag.CommandLine.Output()
)

func _printError(format string, args ...any) {
	if !quiet {
		if _, err := fmt.Fprintf(flagOutput, format+"\n", args...); err != nil {
			panic(err)
		}
	}
}

func _printInfo(format string, args ...any) {
	if !quiet {
		log.Printf(format, args...)
	}
}

func _printDebug(format string, args ...any) {
	if !quiet && verbose {
		log.Printf(format, args...)
	}
}

type StringList []string

func (s *StringList) String() string {
	return strings.Join(*s, ", ")
}

func (s *StringList) Set(value string) error {
	if _, _, err := net.SplitHostPort(value); err != nil {
		return err
	}
	*s = append(*s, value)
	return nil
}

func init() {
	flag.DurationVar(&timeout, "t", 0, "Timeout in format N{ns,ms,s,m,h}, e.g. '5s' == 5 seconds. Zero for no timeout (default 0)")
	flag.DurationVar(&interval, "i", time.Second, "Interval between retries in format N{ns,ms,s,m,h}")
	flag.BoolVar(&quiet, "q", false, "Do not print anything (default false)")
	flag.BoolVar(&verbose, "v", false, "Verbose mode (default false)")
	flag.Var(&endpoints, "a", "Endpoint to await, in the form 'host:port'")
	flag.StringVar(&on, "on", "s", "Condition for command execution. Possible values: 's' - after success, 'f' - after failure, 'any' - always (default 's')")
	flag.Usage = func() {
		const usageFormat = "Usage: %s [-t timeout] [-i interval] [-on (s|f|any)] [-q] [-v] [-a host:port ...] [command [args]]\n"
		_printError(usageFormat, os.Args[0])
		flag.PrintDefaults()
		_printError("  command args\n    \tExecute command with arguments after the test finishes (default: if connection succeeded)\n")
	}
	flag.Parse()

	if len(endpoints) == 0 {
		_printError("No endpoints provided\n")
		flag.Usage()
		os.Exit(22) // Invalid argument
	}

	command = flag.Args()
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
}

func main() {
	os.Exit(_main())
}

func _main() int {
	err := func() error {
		g, ctx := errgroup.WithContext(context.Background())
		if timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
		}

		for _, addr := range endpoints {
			g.Go(func() error {
				var d net.Dialer
				var addrErr *net.AddrError
				var dnsErr *net.DNSError
				_printDebug("connecting to %s...", addr)
				for {
					if conn, err := d.DialContext(ctx, "tcp", addr); err != nil {
						_printDebug(err.Error())
						if errors.As(err, &addrErr) || errors.As(err, &dnsErr) {
							return err
						}
						select {
						case <-ctx.Done():
							return ctx.Err()
						default:
							time.Sleep(interval)
						}
					} else {
						_printInfo("successfully connected to %s", addr)
						if err = conn.Close(); err != nil {
							_printError(err.Error())
						}
						return nil
					}
				}
			})
		}

		return g.Wait()
	}()

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			_printError("timeout error")
		} else {
			_printError(err.Error())
		}
	}

	if len(command) > 0 && ((on == "s" && err == nil) || (on == "f" && err != nil) || on == "any") {
		cmd := exec.Command(command[0], command[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		var exErr *exec.ExitError
		if err != nil {
			if errors.As(err, &exErr) {
				return exErr.ExitCode()
			} else {
				_printError(err.Error())
			}
		}
	}

	if err != nil {
		return 1
	}
	return 0
}
