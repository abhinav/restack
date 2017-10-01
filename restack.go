package restack

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strings"
)

// Restacker reads the todo list of an interactive rebase and writes a new
// version of it with the provided configuration.
type Restacker struct {
	// Name of a git remote. If non-empty, an opt-in section that pushes
	// restacked branches to this remote is also generated.
	RemoteName string

	// FS controls how Restacker accesses the filesystem. Defaults to
	// DefaultFilesystem.
	FS FS
}

// Matches the list of refs at the end of the "pick" instruction.
//
//   pick 12345678 Do stuff (origin/foo, foo)
//
// This requires that the rebase.instructionFormat ends with "%d"
var _refList = regexp.MustCompile(`\(([^)]+)\)$`)

const _pushSectionPrefix = "\n# Uncomment this section to push the changes.\n"

// Run reads rebase instructions from src and writes them to dst based on the
// Restacker configuration.
func (r Restacker) Run(dst io.Writer, src io.Reader) error {
	if r.FS == nil {
		r.FS = DefaultFilesystem
	}

	var branches []string

	scanner := bufio.NewScanner(src)
	for scanner.Scan() {
		line := scanner.Text()

		// If we found an empty line, the instructions section is over. We
		// will add our push instructions here.
		if len(line) == 0 && len(branches) > 0 && len(r.RemoteName) > 0 {
			if _, err := io.WriteString(dst, _pushSectionPrefix); err != nil {
				return err
			}

			for _, b := range branches {
				if _, err := fmt.Fprintf(dst, "# exec git push -f %s %s\n", r.RemoteName, b); err != nil {
					return err
				}
			}

			if _, err := fmt.Fprintln(dst); err != nil {
				return err
			}
		}

		// Most lines go in as-is.
		if _, err := fmt.Fprintln(dst, line); err != nil {
			return err
		}

		if !strings.HasPrefix(line, "pick ") {
			continue
		}

		// TODO(abg): An alternative method we could use here is to parse the
		// hash and match against 'git show-ref --heads --abbrev'. This will
		// allow us to leave the git config alone.

		matches := _refList.FindStringSubmatch(line)
		if len(matches) < 2 {
			continue
		}

		refs := strings.Split(matches[1], ",")
		for _, ref := range refs {
			ref = strings.TrimSpace(ref)
			if r.FS.FileExists(fmt.Sprintf(".git/refs/heads/%v", ref)) {
				if _, err := fmt.Fprintf(dst, "exec git branch -f %v\n", ref); err != nil {
					return err
				}
				branches = append(branches, ref)
			}
		}
	}

	return scanner.Err()
}
