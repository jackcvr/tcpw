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
	"runtime/debug"
	"strings"
	"time"
)

var config struct {
	timeout   time.Duration
	interval  time.Duration
	quiet     bool
	verbose   bool
	endpoints StringList
	on        string
	command   []string
}

func printError(format string, args ...any) {
	if !config.quiet {
		fmt.Fprintf(os.Stderr, format+"\n", args...)
	}
}

func logInfo(format string, args ...any) {
	if !config.quiet {
		log.Printf(format, args...)
	}
}

func logDebug(format string, args ...any) {
	if !config.quiet && config.verbose {
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
	debug.SetGCPercent(25)
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	flag.CommandLine.SetOutput(os.Stderr)

	flag.DurationVar(&config.timeout, "t", 0, "Timeout in format N{ns,ms,s,m,h}, e.g. '5s' == 5 seconds. Zero for no timeout (default 0)")
	flag.DurationVar(&config.interval, "i", time.Second, "Interval between retries in format N{ns,ms,s,m,h}")
	flag.BoolVar(&config.quiet, "q", false, "Do not print anything (default false)")
	flag.BoolVar(&config.verbose, "v", false, "Verbose mode (default false)")
	flag.Var(&config.endpoints, "a", "Endpoint to await, in the form 'host:port'")
	flag.StringVar(&config.on, "on", "s", "Condition for command execution. Possible values: 's' - after success, 'f' - after failure, 'any' - always")
	flag.Usage = func() {
		const usageFormat = "Usage: %s [-t timeout] [-i interval] [-on (s|f|any)] [-q] [-v] [-a host:port ...] [command [args]]\n"
		printError(usageFormat, os.Args[0])
		flag.PrintDefaults()
		printError("  command args\n    \tExecute command with arguments after the test finishes (default: if connection succeeded)\n")
	}
}

func checkConfig() error {
	if len(config.endpoints) == 0 {
		return errors.New("no endpoints provided")
	}
	if config.on != "s" && config.on != "f" && config.on != "any" {
		return errors.New("only 's' or 'f' of 'any' are allowed for '-on' argument")
	}
	return nil
}

func main() {
	flag.Parse()
	if err := checkConfig(); err != nil {
		printError(err.Error())
		flag.Usage()
		os.Exit(22) // Invalid argument code
	}
	config.command = flag.Args()
	if err := Run(); err != nil {
		var exErr *exec.ExitError
		if errors.As(err, &exErr) {
			os.Exit(exErr.ExitCode())
		}
		os.Exit(1)
	}
}

func Run() error {
	err := Connect()
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			printError("timeout error")
		} else {
			printError(err.Error())
		}
	}
	if len(config.command) > 0 && ((config.on == "s" && err == nil) || (config.on == "f" && err != nil) || config.on == "any") {
		cmd := exec.Command(config.command[0], config.command[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
	}
	return err
}

func Connect() error {
	g, ctx := errgroup.WithContext(context.Background())
	if config.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, config.timeout)
		defer cancel()
	}

	d := net.Dialer{Timeout: config.timeout}
	for _, addr := range config.endpoints {
		g.Go(func() error {
			ticker := time.NewTicker(config.interval)
			defer ticker.Stop()
			logDebug("connecting to %s...", addr)
			for {
				res, err := TryDial(ctx, d, addr)
				if err != nil {
					return err
				}
				if res {
					logInfo("successfully connected to %s", addr)
					return nil
				} else {
					select {
					case <-ticker.C:
						break
					case <-ctx.Done():
						return ctx.Err()
					}
				}
			}
		})
	}

	return g.Wait()
}

func TryDial(ctx context.Context, d net.Dialer, addr string) (bool, error) {
	var addrErr *net.AddrError
	var dnsErr *net.DNSError
	if conn, err := d.DialContext(ctx, "tcp", addr); err != nil {
		logDebug(err.Error())
		if errors.As(err, &addrErr) || errors.As(err, &dnsErr) {
			return false, err
		}
		return false, nil
	} else {
		if err = conn.Close(); err != nil {
			printError(err.Error())
		}
		return true, nil
	}
}
