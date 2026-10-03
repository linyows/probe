package probe

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// checkout makes a working directory with a file outside it and a symlink
// "out" inside it pointing at the outside directory, as a checked-out
// project could carry. It returns the outside file.
func checkout(t *testing.T) (outsideFile string) {
	t.Helper()
	base := t.TempDir()
	outside := filepath.Join(base, "outside")
	project := filepath.Join(base, "project")
	for _, d := range []string{outside, project} {
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	outsideFile = filepath.Join(outside, "victim")
	if err := os.WriteFile(outsideFile, []byte("keep me"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(project, "out")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	t.Chdir(project)
	return outsideFile
}

func unchanged(t *testing.T, path string) {
	t.Helper()
	if data, _ := os.ReadFile(path); string(data) != "keep me" {
		t.Errorf("%s was changed to %.40q", path, data)
	}
}

func writeText(s string) func(io.Writer) error {
	return func(w io.Writer) error {
		_, err := io.WriteString(w, s)
		return err
	}
}

// umaskMode is the mode os.Create gives a new file under the current umask.
func umaskMode(t *testing.T) os.FileMode {
	t.Helper()
	ref := filepath.Join(t.TempDir(), "ref")
	f, err := os.Create(ref)
	if err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	info, err := os.Stat(ref)
	if err != nil {
		t.Fatal(err)
	}
	return info.Mode().Perm()
}

func TestReplaceFile_RefusesLinkedParentDirectory(t *testing.T) {
	victim := checkout(t)

	err := replaceFile("out/victim", writeText("report"))
	if err == nil || !strings.Contains(err.Error(), "path escapes") {
		t.Errorf("error = %v, want the linked directory refused", err)
	}
	unchanged(t, victim)

	// A deeper directory created under the link is refused the same way.
	if err := replaceFile("out/new/report.json", writeText("report")); err == nil {
		t.Error("creating a directory through the link should be refused")
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(victim), "new")); err == nil {
		t.Error("nothing should have been created outside the project")
	}
}

func TestReplaceFile_ReplacesLinkToDirectory(t *testing.T) {
	checkout(t)

	// "out" itself is a link to a directory: as the final component it is
	// replaced by the file, not refused as a directory.
	if err := replaceFile("out", writeText("report")); err != nil {
		t.Fatalf("a link at the path should be replaced: %v", err)
	}
	info, err := os.Lstat("out")
	if err != nil {
		t.Fatal(err)
	}
	if !info.Mode().IsRegular() {
		t.Errorf("out is %v, want a regular file", info.Mode())
	}
}

func TestReplaceFile_RefusesRealDirectory(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.Mkdir("dir", 0o755); err != nil {
		t.Fatal(err)
	}
	err := replaceFile("dir", writeText("report"))
	if err == nil || !strings.Contains(err.Error(), "is a directory") {
		t.Errorf("error = %v, want a directory refused", err)
	}
}

func TestReplaceFile_Modes(t *testing.T) {
	t.Chdir(t.TempDir())

	if err := replaceFile("new.json", writeText("a")); err != nil {
		t.Fatal(err)
	}
	if info, _ := os.Stat("new.json"); info.Mode().Perm() != umaskMode(t) {
		t.Errorf("new file mode = %v, want %v as the umask allows", info.Mode().Perm(), umaskMode(t))
	}

	if err := os.WriteFile("private.json", []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod("private.json", 0o600); err != nil {
		t.Fatal(err)
	}
	if err := replaceFile("private.json", writeText("new")); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat("private.json")
	if info.Mode().Perm() != 0o600 {
		t.Errorf("existing file mode = %v, want its 0600 kept", info.Mode().Perm())
	}
	if data, _ := os.ReadFile("private.json"); string(data) != "new" {
		t.Errorf("content = %q", data)
	}
}

func TestReplaceFile_OutsideTheWorkingDirectory(t *testing.T) {
	t.Chdir(t.TempDir())
	outside := filepath.Join(t.TempDir(), "nested", "report.json")

	if err := replaceFile(outside, writeText("report")); err != nil {
		t.Fatalf("an explicit path outside the working directory should be written: %v", err)
	}
	if data, _ := os.ReadFile(outside); string(data) != "report" {
		t.Errorf("content = %q", data)
	}
}

func TestOpenForAppend_RefusesLinkedParentDirectory(t *testing.T) {
	victim := checkout(t)

	if _, _, err := openForAppend("out/victim", true); err == nil {
		t.Error("appending through a linked directory should be refused")
	}
	unchanged(t, victim)

	// Without confine, as for GITHUB_STEP_SUMMARY, the path is used as given.
	f, _, err := openForAppend("out/victim", false)
	if err != nil {
		t.Fatalf("an unconfined path should be opened as given: %v", err)
	}
	_ = f.Close()
}
