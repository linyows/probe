package probe

import (
	"os"
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
}

// backgroundProcs records the processes that steps start in the background,
// so that they can be stopped when the workflow is over. An action runs in a
// plugin process that exits as soon as the step is done, and the background
// process is put in a session of its own to survive that, so nothing else
// would ever stop it.
type backgroundProcs struct {
	mu    sync.Mutex
	procs []backgroundProc
}

func newBackgroundProcs() *backgroundProcs {
	return &backgroundProcs{}
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

	b.mu.Lock()
	defer b.mu.Unlock()
	b.procs = append(b.procs, backgroundProc{pid: pid, log: log})
}

// stop terminates every recorded process and removes its log file. Each
// process leads its own process group, so the whole group is signalled and
// the commands the shell started go with it.
func (b *backgroundProcs) stop() {
	if b == nil {
		return
	}
	b.mu.Lock()
	procs := b.procs
	b.procs = nil
	b.mu.Unlock()

	if len(procs) == 0 {
		return
	}

	for _, p := range procs {
		_ = syscall.Kill(-p.pid, syscall.SIGTERM)
	}

	deadline := time.Now().Add(backgroundStopGrace)
	for _, p := range procs {
		for groupAlive(p.pid) && time.Now().Before(deadline) {
			time.Sleep(50 * time.Millisecond)
		}
		if groupAlive(p.pid) {
			_ = syscall.Kill(-p.pid, syscall.SIGKILL)
		}
	}

	for _, p := range procs {
		if p.log != "" {
			_ = os.Remove(p.log)
		}
	}
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
