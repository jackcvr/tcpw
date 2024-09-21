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

type App struct {
	timeout   time.Duration
	interval  time.Duration
	quiet     bool
	verbose   bool
	endpoints Endpoints
	on        string
	command   []string
}

type Endpoints []string

func (ep *Endpoints) String() string {
	return strings.Join(*ep, ", ")
}

func (ep *Endpoints) Set(value string) error {
	addr, err := net.ResolveTCPAddr("tcp", value)
	if err != nil {
		return err
	}
	*ep = append(*ep, addr.String())
	return nil
}

func (app App) Error(format string, args ...any) {
	if !app.quiet {
		fmt.Fprintf(os.Stderr, format+"\n", args...)
	}
}

func (app App) Info(format string, args ...any) {
	if !app.quiet {
		log.Printf(format, args...)
	}
}

func (app App) Debug(format string, args ...any) {
	if !app.quiet && app.verbose {
		log.Printf(format, args...)
	}
}

func (app App) Check() error {
	if len(app.endpoints) == 0 {
		return errors.New("no endpoints provided")
	}
	if app.on != "s" && app.on != "f" && app.on != "any" {
		return errors.New("only 's' or 'f' of 'any' are allowed for '-on' argument")
	}
	return nil
}

func (app App) Run() error {
	err := app.Connect()
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			app.Error("timeout error")
		} else {
			app.Error(err.Error())
		}
	}
	if len(app.command) > 0 && ((app.on == "s" && err == nil) || (app.on == "f" && err != nil) || app.on == "any") {
		cmd := exec.Command(app.command[0], app.command[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
	}
	return err
}

func (app App) Connect() error {
	g, ctx := errgroup.WithContext(context.Background())
	if app.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, app.timeout)
		defer cancel()
	}

	d := net.Dialer{Timeout: app.timeout}
	for _, addr := range app.endpoints {
		g.Go(func() error {
			ticker := time.NewTicker(app.interval)
			defer ticker.Stop()
			app.Debug("connecting to %s...", addr)
			for {
				res, err := app.TryDial(ctx, d, addr)
				if err != nil {
					return err
				}
				if res {
					app.Info("successfully connected to %s", addr)
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

func (app App) TryDial(ctx context.Context, d net.Dialer, addr string) (bool, error) {
	var addrErr *net.AddrError
	var dnsErr *net.DNSError
	if conn, err := d.DialContext(ctx, "tcp", addr); err != nil {
		app.Debug(err.Error())
		if errors.As(err, &addrErr) || errors.As(err, &dnsErr) {
			return false, err
		}
		return false, nil
	} else {
		if err = conn.Close(); err != nil {
			app.Error(err.Error())
		}
		return true, nil
	}
}

func init() {
	debug.SetGCPercent(25)
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	flag.CommandLine.SetOutput(os.Stderr)
}

func main() {
	var app App

	flag.DurationVar(&app.timeout, "t", 0, "Timeout in format N{ns,ms,s,m,h}, e.g. '5s' == 5 seconds. Zero for no timeout (default 0)")
	flag.DurationVar(&app.interval, "i", time.Second, "Interval between retries in format N{ns,ms,s,m,h}")
	flag.BoolVar(&app.quiet, "q", false, "Do not print anything (default false)")
	flag.BoolVar(&app.verbose, "v", false, "Verbose mode (default false)")
	flag.Var(&app.endpoints, "a", "Endpoint to await, in the form 'host:port'")
	flag.StringVar(&app.on, "on", "s", "Condition for command execution. Possible values: 's' - after success, 'f' - after failure, 'any' - always")
	flag.Usage = func() {
		const usageFormat = "Usage: %s [-t timeout] [-i interval] [-on (s|f|any)] [-q] [-v] [-a host:port ...] [command [args]]\n"
		app.Error(usageFormat, os.Args[0])
		flag.PrintDefaults()
		app.Error("  command args\n    \tExecute command with arguments after the test finishes (default: if connection succeeded)\n")
	}
	flag.Parse()

	if err := app.Check(); err != nil {
		app.Error(err.Error())
		flag.Usage()
		os.Exit(22) // Invalid argument code
	}
	app.command = flag.Args()
	if err := app.Run(); err != nil {
		var exErr *exec.ExitError
		if errors.As(err, &exErr) {
			os.Exit(exErr.ExitCode())
		}
		os.Exit(1)
	}
}
