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

func resetConfig() {
	config.timeout = 1 * time.Second
	config.interval = 100 * time.Millisecond
	config.quiet = true
	config.endpoints = []string{}
	config.on = "s"
	config.command = []string{}
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

func TestCheckConfig(t *testing.T) {
	resetConfig()

	t.Run("Test success", func(t *testing.T) {
		defer resetConfig()
		config.endpoints = []string{"localhost:1234"}
		if err := checkConfig(); err != nil {
			t.Errorf(err.Error())
		}
	})

	t.Run("Test error: no endpoints", func(t *testing.T) {
		defer resetConfig()
		if err := checkConfig(); err.Error() != "no endpoints provided" {
			t.Errorf("Returned wrong error")
		}
	})

	t.Run("Test error: wrong '-on' value", func(t *testing.T) {
		defer resetConfig()
		config.endpoints = []string{"localhost:1234"}
		config.on = "w"
		if err := checkConfig(); err.Error() != "only 's' or 'f' of 'any' are allowed for '-on' argument" {
			t.Errorf("Returned wrong error")
		}
	})
}

func TestTryDial(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second) // global timeout for inner tests
	t.Cleanup(func() {
		cancel()
	})

	t.Run("Test success", func(t *testing.T) {
		t.Parallel()

		addr := startListener("")
		res, err := TryDial(ctx, net.Dialer{Timeout: 1 * time.Second}, addr.String())
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if res != true {
			t.Errorf("Connection failed")
		}
	})

	t.Run("Test fail", func(t *testing.T) {
		t.Parallel()

		res, err := TryDial(ctx, net.Dialer{}, getFreeTCPAddr().String())
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if res != false {
			t.Errorf("Connection succeeded on fail test")
		}
	})

	t.Run("Test error", func(t *testing.T) {
		t.Parallel()

		res, err := TryDial(ctx, net.Dialer{}, badAddr)
		if err == nil {
			t.Errorf("Unexpected success: %v", res)
		} else if err.Error() != badAddrError {
			t.Errorf("Unexpected error string: %v", err)
		}
	})
}

func TestRun(t *testing.T) {
	resetConfig()

	t.Run("Test success", func(t *testing.T) {
		defer resetConfig()
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
		config.endpoints = []string{addr1, addr2}
		if err := Run(); err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	t.Run("Test fail", func(t *testing.T) {
		defer resetConfig()
		config.timeout = 100 * time.Millisecond
		config.endpoints = []string{getFreeTCPAddr().String()}
		if err := Run(); err == nil {
			t.Errorf("Connection succeeded on fail test")
		}
	})

	t.Run("Test error", func(t *testing.T) {
		defer resetConfig()
		config.timeout = 100 * time.Millisecond
		config.endpoints = []string{badAddr}
		if err := Run(); err == nil {
			t.Errorf("Connection succeeded on fail test")
		} else if err.Error() != badAddrError {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	t.Run("Test success with command (-on s)", func(t *testing.T) {
		defer resetConfig()
		addr := startListener("")
		config.endpoints = []string{addr.String()}
		file := t.TempDir() + "/test"
		config.command = []string{"touch", file}
		if err := Run(); err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if _, err := os.Stat(file); os.IsNotExist(err) {
			t.Errorf("File %s does not exist", file)
		}
	})

	t.Run("Test fail with command (-on s)", func(t *testing.T) {
		defer resetConfig()
		config.timeout = 100 * time.Millisecond
		config.endpoints = []string{getFreeTCPAddr().String()}
		file := t.TempDir() + "/test"
		config.command = []string{"touch", file}
		if err := Run(); err == nil {
			t.Errorf("Connection succeeded on fail test")
		}
		if _, err := os.Stat(file); err == nil {
			t.Errorf("File %s exists on fail test", file)
		}
	})

	t.Run("Test fail with command (-on f)", func(t *testing.T) {
		defer resetConfig()
		config.timeout = 100 * time.Millisecond
		config.on = "f"
		config.endpoints = []string{getFreeTCPAddr().String()}
		file := t.TempDir() + "/test"
		config.command = []string{"touch", file}
		if err := Run(); err != nil {
			t.Errorf("Unexpected command(touch ...) error: %v", err)
		}
		if _, err := os.Stat(file); os.IsNotExist(err) {
			t.Errorf("File %s does not exist, but '-on f' argument was provided", file)
		}
	})
}
