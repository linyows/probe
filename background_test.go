package probe

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// startGroup starts cmd in a process group of its own, as the shell action
// does for a background step, and returns its pid.
func startGroup(t *testing.T, cmd string) int {
	t.Helper()
	c := exec.Command("/bin/sh", "-c", cmd)
	c.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := c.Start(); err != nil {
		t.Fatalf("start %q: %v", cmd, err)
	}
	// Reap the process once it is stopped, so that the group is really gone.
	go func() { _ = c.Wait() }()
	pid := c.Process.Pid
	t.Cleanup(func() { _ = syscall.Kill(-pid, syscall.SIGKILL) })
	return pid
}

// waitGroupGone reports whether the group led by pid is gone within a second.
// A killed process stays visible until it is reaped, which happens on its
// own schedule.
func waitGroupGone(pid int) bool {
	deadline := time.Now().Add(time.Second)
	for groupAlive(pid) {
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(20 * time.Millisecond)
	}
	return true
}

func backgroundResult(pid int, log string) map[string]any {
	return map[string]any{
		"req": map[string]any{"cmd": "sleep 60", "background": true},
		"res": map[string]any{"code": int64(-1), "pid": int64(pid), "log": log},
	}
}

func TestBackgroundProcsTrack(t *testing.T) {
	tests := []struct {
		name string
		uses string
		ret  map[string]any
		want []backgroundProc
	}{
		{
			name: "background shell step",
			uses: "shell",
			ret:  backgroundResult(123, "/tmp/a.log"),
			want: []backgroundProc{{pid: 123, log: "/tmp/a.log"}},
		},
		{
			name: "foreground shell step",
			uses: "shell",
			ret: map[string]any{
				"req": map[string]any{"cmd": "true", "background": false},
				"res": map[string]any{"code": int64(0), "pid": int64(123)},
			},
		},
		{
			name: "other action",
			uses: "http",
			ret:  backgroundResult(123, "/tmp/a.log"),
		},
		{
			name: "no pid",
			uses: "shell",
			ret: map[string]any{
				"req": map[string]any{"background": true},
				"res": map[string]any{},
			},
		},
		{
			name: "empty result",
			uses: "shell",
			ret:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := newBackgroundProcs()
			b.track(tt.uses, tt.ret)
			if len(b.procs) != len(tt.want) {
				t.Fatalf("procs = %v, want %v", b.procs, tt.want)
			}
			for i := range tt.want {
				if b.procs[i] != tt.want[i] {
					t.Errorf("procs[%d] = %v, want %v", i, b.procs[i], tt.want[i])
				}
			}
		})
	}
}

func TestBackgroundProcsNil(t *testing.T) {
	// A JobContext built without a tracker must not panic.
	var b *backgroundProcs
	b.track("shell", backgroundResult(123, ""))
	b.stop()
}

func TestBackgroundProcsStop(t *testing.T) {
	dir := t.TempDir()
	log := filepath.Join(dir, "probe-shell-action.1.log")
	if err := os.WriteFile(log, []byte("output\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	// The shell stays as the group leader and sleep runs as its child: both
	// have to go.
	pid := startGroup(t, "sleep 60 & wait")

	b := newBackgroundProcs()
	b.track("shell", backgroundResult(pid, log))
	b.stop()

	if !waitGroupGone(pid) {
		t.Errorf("process group %d is still running", pid)
	}
	if _, err := os.Stat(log); !os.IsNotExist(err) {
		t.Errorf("log %s was not removed: %v", log, err)
	}
	if len(b.procs) != 0 {
		t.Errorf("procs = %v, want none left", b.procs)
	}
}

func TestBackgroundProcsStopIgnoringTerm(t *testing.T) {
	if testing.Short() {
		t.Skip("waits for the grace period")
	}
	pid := startGroup(t, "trap '' TERM; sleep 60 & wait")
	// Give the shell time to install the trap.
	time.Sleep(200 * time.Millisecond)

	b := newBackgroundProcs()
	b.track("shell", backgroundResult(pid, ""))

	start := time.Now()
	b.stop()
	elapsed := time.Since(start)

	if !waitGroupGone(pid) {
		t.Errorf("process group %d survived SIGKILL", pid)
	}
	if elapsed < backgroundStopGrace {
		t.Errorf("stopped after %v, want SIGKILL only after the %v grace", elapsed, backgroundStopGrace)
	}
}

func TestBackgroundProcsStopExited(t *testing.T) {
	// A process that already finished is skipped without waiting.
	pid := startGroup(t, "true")
	if !waitGroupGone(pid) {
		t.Fatalf("process group %d did not exit", pid)
	}

	b := newBackgroundProcs()
	b.track("shell", backgroundResult(pid, ""))

	start := time.Now()
	b.stop()
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("stop took %v for a process that had exited", elapsed)
	}
}

func TestProcessStartTime(t *testing.T) {
	a, err := processStartTime(os.Getpid())
	if err != nil {
		t.Fatalf("processStartTime(self) error: %v", err)
	}
	b, _ := processStartTime(os.Getpid())
	if a == 0 || a != b {
		t.Errorf("processStartTime(self) = %d then %d, want the same non-zero value", a, b)
	}

	pid := startGroup(t, "true")
	if !waitGroupGone(pid) {
		t.Fatalf("process %d did not exit", pid)
	}
	if _, err := processStartTime(pid); err == nil {
		t.Errorf("processStartTime(%d) succeeded for a process that has exited", pid)
	}
}

func TestBackgroundProcsStopReusedPid(t *testing.T) {
	// A process with the recorded pid that started at another time is not the
	// one the step started, and must be left alone.
	pid := startGroup(t, "sleep 60")
	started, err := processStartTime(pid)
	if err != nil {
		t.Fatal(err)
	}

	b := newBackgroundProcs()
	b.procs = []backgroundProc{{pid: pid, started: started + 1}}
	b.stop()

	if !groupAlive(pid) {
		t.Errorf("process group %d was signalled although its leader is not the recorded one", pid)
	}
}

func TestBackgroundProcsStopLeaderGone(t *testing.T) {
	// The shell exits at once and leaves sleep running in its group. The pid
	// cannot have been reused while the group lives, so the group is stopped.
	pid := startGroup(t, "sleep 60 & exit 0")
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := processStartTime(pid); err != nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !groupAlive(pid) {
		t.Fatalf("process group %d is gone before the test", pid)
	}

	b := newBackgroundProcs()
	b.track("shell", backgroundResult(pid, ""))
	b.stop()

	if !waitGroupGone(pid) {
		t.Errorf("process group %d is still running", pid)
	}
}

func TestBackgroundProcsTrackAfterStop(t *testing.T) {
	// A step that timed out can report its process after the workflow has
	// stopped the others; it is stopped right away.
	b := newBackgroundProcs()
	b.stop()

	pid := startGroup(t, "sleep 60")
	b.track("shell", backgroundResult(pid, ""))

	if !waitGroupGone(pid) {
		t.Errorf("process group %d reported after stop is still running", pid)
	}
	if len(b.procs) != 0 {
		t.Errorf("procs = %v, want none kept after stop", b.procs)
	}
}

func TestBackgroundProcsStopWaitsForPending(t *testing.T) {
	// An action still running when the workflow ends gets a moment to report
	// the process it started.
	b := newBackgroundProcs()
	done := b.begin("shell")
	pid := startGroup(t, "sleep 60")
	go func() {
		time.Sleep(200 * time.Millisecond)
		b.track("shell", backgroundResult(pid, ""))
		done()
	}()

	b.stop()

	if !waitGroupGone(pid) {
		t.Errorf("process group %d started by a pending action is still running", pid)
	}
}

func TestBackgroundProcsBeginOtherAction(t *testing.T) {
	// Only shell actions start background processes, so nothing else is
	// waited for.
	b := newBackgroundProcs()
	done := b.begin("http")
	defer done()
	if n := b.pendingCount(); n != 0 {
		t.Errorf("pending = %d, want 0", n)
	}
}

// lateBackgroundRunner starts a process the way a background shell step does,
// but only after delay, so that the step has already timed out.
type lateBackgroundRunner struct {
	t     *testing.T
	delay time.Duration
	pid   chan int
}

func (r *lateBackgroundRunner) RunActions(name string, with map[string]any, opts RunOptions) (map[string]any, error) {
	time.Sleep(r.delay)
	pid := startGroup(r.t, "sleep 60")
	r.pid <- pid
	return backgroundResult(pid, ""), nil
}

func TestStepTimeoutStillTracksBackground(t *testing.T) {
	step := &Step{
		Uses:    "shell",
		Timeout: Interval{Duration: 100 * time.Millisecond},
		Expr:    &Expr{},
	}
	runner := &lateBackgroundRunner{t: t, delay: 300 * time.Millisecond, pid: make(chan int, 1)}
	jCtx := &JobContext{background: newBackgroundProcs()}

	if _, err := step.executeSingleAction(runner, map[string]any{}, jCtx, false); err == nil {
		t.Fatal("expected the step to time out")
	}

	// The workflow ends while the action is still running.
	jCtx.background.stop()

	pid := <-runner.pid
	if !waitGroupGone(pid) {
		t.Errorf("process group %d started after the step timed out is still running", pid)
	}
}
