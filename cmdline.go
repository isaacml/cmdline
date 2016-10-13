/*
	cmdline package for Golang 1 - (c)Antonio J. Tomas (2016/03/21)

	Description:
		An easier Object to launch long-time executables, controlled by many threads in a safe way.
		Best suited for multimedia players and encoders that run indefinitely and are controlled thru a visual interface.
*/

package cmdline

import (
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"bufio"
	"time"
)

// Exec object with its private properties and below its methods or member functions
type Exec struct {
	cmd     *exec.Cmd  // pointer to the execution controller in the os/exec basic package
	cmdline string     // the complete cmdline with all the args on it
	running bool       // false, true = not running, running
	mu_run  sync.Mutex // mutex to lock all the properties between threads
}

// Thread-safe Cmdline function to enter any length and argumented commandline to be executed by the OS
// This is the constructor of the Exec object. Must be called 1st if you want to use the rest of methods
func Cmdline(cmdline string) *Exec {
	exe := &Exec{
		cmd: exec.Command(""), // the rest of properties are auto initialized by Go because they are not pointers
	}

	exe.mu_run.Lock()
	defer exe.mu_run.Unlock()
	exe.cmdline = cmdline
	exe.running = false
	args := strings.Fields(exe.cmdline)
	exe.cmd.Path = args[0]
	exe.cmd.Args = args

	return exe
}

// Thread-safe function to send a SIGINT signal (Ctrl-C) to our Exec object instead of the SIGINT used by the Stop() func
func (e *Exec) SigInt() error {
	e.mu_run.Lock()
	defer e.mu_run.Unlock()
	e.running = false
	return e.cmd.Process.Signal(syscall.SIGINT)
}

// Thread-safe function to know the state of the executable at any moment
func (e *Exec) IsRunning() bool {
	e.mu_run.Lock()
	defer e.mu_run.Unlock()

	return e.running
}

// Thread-safe function to run completely a cmdline. It will start it and wait for its ending
func (e *Exec) Run() error {
	var err error

	e.mu_run.Lock()
	if e.running {
		defer e.mu_run.Unlock()
		return fmt.Errorf("cmdline: ALREADY_RUNNING_ERROR")
	}
	e.running = true
	e.mu_run.Unlock()
	err = e.cmd.Run()
	e.mu_run.Lock()
	e.running = false
	e.mu_run.Unlock()

	return err
}

// Run a command, and gets out if locked a timeout secs with no output to stderr or just finished its work
// It will start and will wait until its ending
// delim stands for the end of line '\n' or '\r'
func (e *Exec) RunTimeoutStderr(secs int, delim byte) error {
	var err error
	var run int64
	
	e.mu_run.Lock()
	if e.running {
		defer e.mu_run.Unlock()
		return fmt.Errorf("cmdline: ALREADY_RUNNING_ERROR")
	}
	e.running = true
	e.mu_run.Unlock()

	go func(){
		for {
			diff := time.Now().Unix() - run
			if (run != 0) && (diff > int64(secs)) { // running
				e.cmd.Process.Kill()
			}
			time.Sleep(1*time.Second)
			e.mu_run.Lock()
			if !e.running {
				e.mu_run.Unlock()
				break 
			}
			e.mu_run.Unlock()
		}
	}()
	stderr, err1 := e.cmd.StderrPipe()
	if err1 == nil {
		mediareader := bufio.NewReader(stderr)
		e.cmd.Start()
		for{ // bucle de reproduccion normal
			run = time.Now().Unix()
			_,err2 := mediareader.ReadString(delim) // blocks until read
			if err2 != nil {
				break;
			}
		}
		e.cmd.Wait()
	}else{
		err = err1
	}

	e.mu_run.Lock()
	e.running = false
	e.mu_run.Unlock()

	return err
}

// Run a command, and gets out if locked a timeout secs with no output to stdout or just finished its work
// It will start and will wait until its ending
// delim stands for the end of line '\n' or '\r'
func (e *Exec) RunTimeoutStdout(secs int, delim byte) error {
	var err error
	var run int64
	
	e.mu_run.Lock()
	if e.running {
		defer e.mu_run.Unlock()
		return fmt.Errorf("cmdline: ALREADY_RUNNING_ERROR")
	}
	e.running = true
	e.mu_run.Unlock()

	go func(){
		for {
			diff := time.Now().Unix() - run
			if (run != 0) && (diff > int64(secs)) { // running
				e.cmd.Process.Kill()
			}
			time.Sleep(1*time.Second)
			e.mu_run.Lock()
			if !e.running {
				e.mu_run.Unlock()
				break 
			}
			e.mu_run.Unlock()
		}
	}()
	stdout, err1 := e.cmd.StdoutPipe()
	if err1 == nil {
		mediareader := bufio.NewReader(stdout)
		e.cmd.Start()
		for{ // bucle de reproduccion normal
			run = time.Now().Unix()
			_,err2 := mediareader.ReadString(delim) // blocks until read
			if err2 != nil {
				break;
			}
		}
		e.cmd.Wait()
	}else{
		err = err1
	}

	e.mu_run.Lock()
	e.running = false
	e.mu_run.Unlock()

	return err
}

// Thread-safe Start() method, starts execution of a commandline and does not wait until it ends
// If you use this method to start your process and this one exits prematurely, it will be in the OS memory as a <defunct>
// process until you call Stop(), just to clear it all, and let the system in its original state
// So after any Start() call, you will need an Stop() to clear it all, even if the process got out by its own prematurely
func (e *Exec) Start() error {
	var err error

	e.mu_run.Lock()
	defer e.mu_run.Unlock()
	if e.running {
		return fmt.Errorf("cmdline: ALREADY_RUNNING_ERROR")
	}
	if err = e.cmd.Start(); err == nil {
		e.running = true
	}

	return err
}

// Thread-safe Stop() method, stops totally the execution of a commandline and waits until it ends all
// Once you use Stop() method, the Exec object will not be available for future use, so you will have
// to use the Cmdline() constructor again in order to Start() the cmdline again
func (e *Exec) Stop() error {
	var err error

	e.mu_run.Lock()
	defer e.mu_run.Unlock()
	if !e.running {
		return fmt.Errorf("cmdline: NOT_RUNNING_ERROR")
	}
	e.running = false
	if err = e.cmd.Process.Kill(); err != nil {
		return fmt.Errorf("cmdline: NOT_KILLING_ERROR")
	}
	_, err = e.cmd.Process.Wait()
	if err != nil {
		e.running = true // not killed properly, still zombie in memory (very odd and unusual)
	}

	return err
}

// Thread-safe StderrPipe() method, creates a Reader I/O to the stderr of your commandline executed that you can read from
// This method must be used before using Start()
func (e *Exec) StderrPipe() (io.ReadCloser, error) {
	var err error
	var stderr io.ReadCloser

	e.mu_run.Lock()
	defer e.mu_run.Unlock()
	if e.running {
		return nil, fmt.Errorf("cmdline: PIPE_RUNNING_ERROR")
	}
	stderr, err = e.cmd.StderrPipe()

	return stderr, err
}

// Thread-safe StdoutPipe() method, creates a Reader I/O to the stdout of your commandline executed that you can read from
// This method must be used before using Start()
func (e *Exec) StdoutPipe() (io.ReadCloser, error) {
	var err error
	var stdout io.ReadCloser

	e.mu_run.Lock()
	defer e.mu_run.Unlock()
	if e.running {
		return nil, fmt.Errorf("cmdline: PIPE_RUNNING_ERROR")
	}
	stdout, err = e.cmd.StdoutPipe()

	return stdout, err
}

// Thread-safe StdinPipe() method, creates a Writer I/O to the stdin of your commandline executed that you can write to
// This method must be used before using Start()
func (e *Exec) StdinPipe() (io.WriteCloser, error) {
	var err error
	var stdin io.WriteCloser

	e.mu_run.Lock()
	defer e.mu_run.Unlock()
	if e.running {
		return nil, fmt.Errorf("cmdline: PIPE_RUNNING_ERROR")
	}
	stdin, err = e.cmd.StdinPipe()

	return stdin, err
}
