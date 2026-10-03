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
