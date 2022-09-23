use std::{fs, io, path};

use anyhow::{Context, Result};

/// rename is a version of fs::rename that can
/// gracefully recover from cross-device rename errors on Linux.
pub fn rename(src: &path::Path, dst: &path::Path) -> Result<()> {
    rename_impl(|src, dst| fs::rename(src, dst), src, dst)
}

fn rename_impl<RenameFn>(fs_rename: RenameFn, src: &path::Path, dst: &path::Path) -> Result<()>
where
    RenameFn: Fn(&path::Path, &path::Path) -> io::Result<()>,
{
    match (fs_rename)(src, dst) {
        Ok(_) => Ok(()),
        Err(err) => {
            // If /tmp is mounted to a different partition (it often is),
            // attempting to move the file will cause the error:
            //   invalid cross-device link
            //
            // For that case, fall back to copying the file and
            // deleting the temporary file.
            //
            // This is not the default because move is atomic.
            if err.kind() == io::ErrorKind::CrossesDevices {
                unsafe_rename(src, dst)
            } else {
                Err(anyhow::Error::new(err))
            }
        },
    }
}

/// Renames a file by copying its contents into a new file non-atomically,
/// and deleting the original file.
///
/// This is necessary because on Linux, we cannot move the file across
/// filesystem boundaries, and /tmp is often mounted on a different file system
/// than the user's working directory.
fn unsafe_rename(src: &path::Path, dst: &path::Path) -> Result<()> {
    let md = fs::metadata(src).with_context(|| format!("Failed to inspect {}", src.display()))?;

    {
        let mut r = fs::File::open(src).context("Failed to open source")?;
        let mut w = fs::File::create(dst).context("Failed to open destination")?;
        io::copy(&mut r, &mut w).context("Failed to copy file contents")?;
    }

    fs::set_permissions(dst, md.permissions())
        .context("Failed to update destination permissions")?;
    fs::remove_file(src).context("Failed to delete source file")?;

    Ok(())
}

#[cfg(test)]
mod tests {
    use anyhow::Result;
    use rstest::rstest;

    use super::*;

    #[rstest]
    #[case::simple(&rename)]
    #[case::explicit_unsafe(&unsafe_rename)]
    #[case::cross_device(&cross_device_fail)]
    fn rename_end_to_end(
        #[case] rename_fn: &dyn Fn(&path::Path, &path::Path) -> Result<()>,
    ) -> Result<()> {
        let tempdir = tempfile::tempdir()?;

        let from = tempdir.path().join("foo.txt");
        fs::write(&from, "bar").context("Failed to create starting file")?;

        let to = tempdir.path().join("bar.txt");
        rename_fn(&from, &to).context("Failed to rename")?;

        assert!(to.exists(), "destination does not exist");
        assert!(!from.exists(), "source should not exist");

        Ok(())
    }

    fn cross_device_fail(from: &path::Path, to: &path::Path) -> Result<()> {
        rename_impl(|_, _| Err(io::Error::from_raw_os_error(18)), from, to)
    }
}
