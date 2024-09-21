package main

import (
	"context"
	"log"
	"net"
	"os"
	"testing"
	"time"
)

const badAddr = "localhost:99999"
const badAddrError = "dial tcp: address 99999: invalid port"

func newApp() App {
	return App{
		timeout:   1 * time.Second,
		interval:  100 * time.Millisecond,
		quiet:     true,
		endpoints: []string{},
		on:        "s",
		command:   []string{},
	}
}

func getFreeTCPAddr() *net.TCPAddr {
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		log.Panicf("Can't listen: %v", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr)
}

func startListener(addr string) *net.TCPAddr {
	if addr == "" {
		addr = "localhost:0"
	}
	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Panicf("Can't listen: %v", err)
	}
	go func() {
		defer l.Close()
		if _, err = l.Accept(); err != nil {
			log.Panicf("Can't accept: %v", err)
		}
	}()
	return l.Addr().(*net.TCPAddr)
}

func TestAppCheck(t *testing.T) {
	t.Run("Test success", func(t *testing.T) {
		app := newApp()
		app.endpoints = []string{"localhost:1234"}
		if err := app.Check(); err != nil {
			t.Fatal(err.Error())
		}
	})

	t.Run("Test error: no endpoints", func(t *testing.T) {
		app := newApp()
		if err := app.Check(); err.Error() != "no endpoints provided" {
			t.Fatal("Returned wrong error")
		}
	})

	t.Run("Test error: wrong '-on' value", func(t *testing.T) {
		app := newApp()
		app.endpoints = []string{"localhost:1234"}
		app.on = "w"
		if err := app.Check(); err.Error() != "only 's' or 'f' of 'any' are allowed for '-on' argument" {
			t.Fatal("Returned wrong error")
		}
	})
}

func TestTryDial(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second) // global timeout for inner tests
	t.Cleanup(func() {
		cancel()
	})

	app := newApp()

	t.Run("Test success", func(t *testing.T) {
		t.Parallel()

		addr := startListener("")
		res, err := app.TryDial(ctx, net.Dialer{Timeout: 1 * time.Second}, addr.String())
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if res != true {
			t.Fatal("Connection failed")
		}
	})

	t.Run("Test fail", func(t *testing.T) {
		t.Parallel()

		res, err := app.TryDial(ctx, net.Dialer{}, getFreeTCPAddr().String())
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if res != false {
			t.Fatal("Connection succeeded on fail test")
		}
	})

	t.Run("Test error", func(t *testing.T) {
		t.Parallel()

		res, err := app.TryDial(ctx, net.Dialer{}, badAddr)
		if err == nil {
			t.Fatalf("Unexpected success: %v", res)
		} else if err.Error() != badAddrError {
			t.Fatalf("Unexpected error string: %v", err)
		}
	})
}

func TestRun(t *testing.T) {
	t.Run("Test success", func(t *testing.T) {
		app := newApp()
		addr1 := getFreeTCPAddr().String()
		addr2 := getFreeTCPAddr().String()
		go func() {
			time.Sleep(250 * time.Millisecond)
			_ = startListener(addr1)
		}()
		go func() {
			time.Sleep(550 * time.Millisecond)
			_ = startListener(addr2)
		}()
		app.endpoints = []string{addr1, addr2}
		if err := app.Run(); err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
	})

	t.Run("Test fail", func(t *testing.T) {
		app := newApp()
		app.timeout = 100 * time.Millisecond
		app.endpoints = []string{getFreeTCPAddr().String()}
		if err := app.Run(); err == nil {
			t.Fatal("Connection succeeded on fail test")
		}
	})

	t.Run("Test error", func(t *testing.T) {
		app := newApp()
		app.timeout = 100 * time.Millisecond
		app.endpoints = []string{badAddr}
		if err := app.Run(); err == nil {
			t.Fatalf("Connection succeeded on fail test")
		} else if err.Error() != badAddrError {
			t.Fatalf("Unexpected error: %v", err)
		}
	})

	t.Run("Test success with command (-on s)", func(t *testing.T) {
		app := newApp()
		addr := startListener("")
		app.endpoints = []string{addr.String()}
		file := t.TempDir() + "/test"
		app.command = []string{"touch", file}
		if err := app.Run(); err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if _, err := os.Stat(file); os.IsNotExist(err) {
			t.Fatalf("File %s does not exist", file)
		}
	})

	t.Run("Test fail with command (-on s)", func(t *testing.T) {
		app := newApp()
		app.timeout = 100 * time.Millisecond
		app.endpoints = []string{getFreeTCPAddr().String()}
		file := t.TempDir() + "/test"
		app.command = []string{"touch", file}
		if err := app.Run(); err == nil {
			t.Fatalf("Connection succeeded on fail test")
		}
		if _, err := os.Stat(file); err == nil {
			t.Fatalf("File %s exists on fail test", file)
		}
	})

	t.Run("Test fail with command (-on f)", func(t *testing.T) {
		app := newApp()
		app.timeout = 100 * time.Millisecond
		app.on = "f"
		app.endpoints = []string{getFreeTCPAddr().String()}
		file := t.TempDir() + "/test"
		app.command = []string{"touch", file}
		if err := app.Run(); err != nil {
			t.Fatalf("Unexpected command(touch ...) error: %v", err)
		}
		if _, err := os.Stat(file); os.IsNotExist(err) {
			t.Fatalf("File %s does not exist, but '-on f' argument was provided", file)
		}
	})
}
