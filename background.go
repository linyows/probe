package probe

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// backgroundStopGrace is how long a background process is given to exit on
// SIGTERM before it is killed.
const backgroundStopGrace = 3 * time.Second

// backgroundProc is a process a step left running, along with the file its
// output goes to.
type backgroundProc struct {
	pid int
	log string
	// started is when the group's leader started, or 0 if it was already gone
	// when the step returned. It tells the leader apart from a later process
	// that happens to get the same pid.
	started int64
}

// backgroundProcs records the processes that steps start in the background,
// so that they can be stopped when the workflow is over. An action runs in a
// plugin process that exits as soon as the step is done, and the background
// process is put in a session of its own to survive that, so nothing else
// would ever stop it.
type backgroundProcs struct {
	mu    sync.Mutex
	procs []backgroundProc
	// pending counts shell actions still running. One can start a process
	// even after its step timed out, so stop gives them a moment to report.
	pending int
	// stopped is set once stop has run; a process reported after that is
	// stopped as soon as it is.
	stopped bool
	// stopping is held for the whole of stop, so that a second call, such
	// as one from a signal, returns only once the processes are gone.
	stopping sync.Mutex
}

func newBackgroundProcs() *backgroundProcs {
	return &backgroundProcs{}
}

// begin is called before an action runs, and the function it returns once
// the action has returned and its result has been tracked.
func (b *backgroundProcs) begin(uses string) func() {
	if b == nil || uses != "shell" {
		return func() {}
	}
	b.mu.Lock()
	b.pending++
	b.mu.Unlock()
	return func() {
		b.mu.Lock()
		b.pending--
		b.mu.Unlock()
	}
}

// track records the process a step started in the background, if it did.
func (b *backgroundProcs) track(uses string, ret map[string]any) {
	if b == nil || uses != "shell" {
		return
	}
	req, _ := ret["req"].(map[string]any)
	if bg, _ := req["background"].(bool); !bg {
		return
	}
	res, _ := ret["res"].(map[string]any)
	pid := toInt(res["pid"])
	if pid <= 0 {
		return
	}
	log, _ := res["log"].(string)
	p := backgroundProc{pid: pid, log: log}
	if started, err := processStartTime(pid); err == nil {
		p.started = started
	}

	b.mu.Lock()
	if b.stopped {
		b.mu.Unlock()
		terminate([]backgroundProc{p})
		return
	}
	b.procs = append(b.procs, p)
	b.mu.Unlock()
}

// stopOnSignal stops the recorded processes when probe is interrupted,
// terminated or hung up on, and then lets the signal end probe as it would
// have. The background processes are in sessions of their own, so the
// signal never reaches them, and nothing else would stop them. The first
// signal restores the default handling, so a second one ends probe at once.
// The returned function ends the watch.
func (b *backgroundProcs) stopOnSignal() func() {
	if b == nil {
		return func() {}
	}
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	done := make(chan struct{})
	finished := make(chan struct{})
	go func() {
		select {
		case sig := <-ch:
			signal.Stop(ch)
			b.stop()
			if s, ok := sig.(syscall.Signal); ok {
				_ = syscall.Kill(os.Getpid(), s)
			}
			// The signal ends the process; finished is never closed, so the
			// workflow cannot return and exit before that.
		case <-done:
			close(finished)
		}
	}()
	return func() {
		signal.Stop(ch)
		close(done)
		// A signal can end the workflow early, for example by killing the
		// plugin of a running step. Wait for its handling, so that probe
		// still ends by the signal after the cleanup.
		<-finished
	}
}

// stop terminates every recorded process and removes its log file.
func (b *backgroundProcs) stop() {
	if b == nil {
		return
	}
	b.stopping.Lock()
	defer b.stopping.Unlock()

	deadline := time.Now().Add(backgroundStopGrace)
	for b.pendingCount() > 0 && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}

	b.mu.Lock()
	procs := b.procs
	b.procs = nil
	b.stopped = true
	b.mu.Unlock()

	terminate(procs)
}

func (b *backgroundProcs) pendingCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.pending
}

// terminate stops the given processes and removes their logs. Each process
// leads its own process group, so the whole group is signalled and the
// commands the shell started go with it. A group is only signalled while it
// is still the one the step started.
func terminate(procs []backgroundProc) {
	var owned []backgroundProc
	for _, p := range procs {
		if p.owned() {
			_ = syscall.Kill(-p.pid, syscall.SIGTERM)
			owned = append(owned, p)
		}
	}

	deadline := time.Now().Add(backgroundStopGrace)
	for _, p := range owned {
		for groupAlive(p.pid) && time.Now().Before(deadline) {
			time.Sleep(50 * time.Millisecond)
		}
		if groupAlive(p.pid) && p.owned() {
			_ = syscall.Kill(-p.pid, syscall.SIGKILL)
		}
	}

	for _, p := range procs {
		if p.log != "" {
			_ = os.Remove(p.log)
		}
	}
}

// owned reports whether the process group led by p.pid is still the one the
// step started. A pid is free for reuse once the whole group has gone, so a
// live process with that pid has to be the leader that was recorded. Without
// one, a live group is the leader's children carrying on: the system does not
// hand out a pid that a group still uses.
func (p backgroundProc) owned() bool {
	started, err := processStartTime(p.pid)
	if err == nil {
		return p.started != 0 && started == p.started
	}
	return groupAlive(p.pid)
}

// groupAlive reports whether any process in the group led by pid is left.
// A group that cannot be signalled is not one this run started.
func groupAlive(pid int) bool {
	return syscall.Kill(-pid, 0) == nil
}

func toInt(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	default:
		return 0
	}
}
