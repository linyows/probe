package probe

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
)

// yamlBlock matches a fenced yaml code block in a page, capturing its body.
var yamlBlock = regexp.MustCompile("(?s)```ya?ml[^\n]*\n(.*?)```")

// TestDocsYAMLBlocks parses every yaml code block in the docs, so that an
// example a reader copies is not one Probe cannot load: a plain value holding
// ": ", as an unquoted a ? b : c does, or one that starts with a quote and
// goes on after it, is a YAML error.
//
// A block may repeat a key, as one that shows several ways to write a test
// does, and Probe itself loads a workflow that does. An alias whose anchor is
// defined in another file is an example of merging files, and is skipped.
func TestDocsYAMLBlocks(t *testing.T) {
	root := filepath.Join("docs", "content")
	count := 0
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".md") && !strings.HasSuffix(path, ".mdx") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, m := range yamlBlock.FindAllSubmatchIndex(data, -1) {
			count++
			var v any
			err := yaml.UnmarshalWithOptions(data[m[2]:m[3]], &v, yaml.AllowDuplicateMapKey())
			if err != nil && !strings.Contains(err.Error(), "alias") {
				line := strings.Count(string(data[:m[2]]), "\n") + 1
				t.Errorf("%s:%d: %s", path, line, strings.SplitN(err.Error(), "\n", 2)[0])
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if count == 0 {
		t.Fatalf("no yaml block found under %s", root)
	}
}
