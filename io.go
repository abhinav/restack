package restack

import (
	"errors"
	"io"
	"os"
	"syscall"

	"go.uber.org/multierr"
)

var _osRename = os.Rename

func renameFile(src, dst string) error {
	err := _osRename(src, dst)
	if err == nil {
		return nil
	}

	// If /tmp is mounted to a different partition (it often is),
	// attempting to move the file will cause the error:
	//   invalid cross-device link
	//
	// In that case, fall back to copying over the file
	// and deleting the temporary file manually.
	//
	// This behavior is not the default
	// because an atomic move is preferable.
	if errors.Is(err, syscall.EXDEV) {
		err = unsafeRenameFile(src, dst)
	}

	return err
}

// unsafeRenameFile is a variant of os.Rename
// that operates by manually copying over the contents
// and permissions of src to dst.
func unsafeRenameFile(src, dst string) (err error) {
	defer func() {
		if err == nil {
			// Delete src only if
			// everything else succeeded.
			err = os.Remove(src)
		}
	}()

	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	r, err := os.Open(src)
	if err != nil {
		return err
	}
	defer multierr.AppendInvoke(&err, multierr.Close(r))

	w, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer multierr.AppendInvoke(&err, multierr.Close(w))

	if _, err := io.Copy(w, r); err != nil {
		return err
	}

	return multierr.Combine(
		w.Sync(),
		w.Chmod(info.Mode()),
	)
}
