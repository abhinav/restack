//! exec adds an exit_ok method to process::ExitStatus.
//! This method returns an Error if the status code is non-zero.
//!
//! This is our own stable copy of the exit_status_error unstable feature.

use std::process;

// In lieu of exit_status_error stabilization,
// extend ExitStatus with our own exit_ok method.

/// Error returned if a process exits with a non-zero status code.
#[derive(Debug)]
pub struct Error {
    /// The exit code of the process.
    ///
    /// Guaranteed to be non-zero.
    pub code: i32,
}

impl std::error::Error for Error {
    fn description(&self) -> &str {
        "exited with a non-zero status code"
    }
}

impl std::fmt::Display for Error {
    fn fmt(&self, f: &mut std::fmt::Formatter) -> std::fmt::Result {
        write!(f, "exited with status code: {}", self.code)
    }
}

/// Extend ExitStatus with an exit_ok method.
pub trait ExitStatusExt {
    /// Require that the process exited successfully.
    /// If the process did not exit successfully, return an Error.
    fn exit_ok(self) -> Result<(), Error>;
}

impl ExitStatusExt for process::ExitStatus {
    fn exit_ok(self) -> Result<(), Error> {
        if self.success() {
            Ok(())
        } else {
            Err(Error {
                code: self.code().unwrap_or(1),
            })
        }
    }
}

#[cfg(test)]
mod tests {
    use pretty_assertions::assert_eq;

    use super::*;

    #[test]
    fn test_exit_ok() -> anyhow::Result<()> {
        let status = process::Command::new("true").status()?;
        status.exit_ok()?;

        Ok(())
    }

    #[test]
    fn test_exit_err() -> anyhow::Result<()> {
        let status = process::Command::new("false").status()?;
        let got_err = status.exit_ok().expect_err("expected error");

        assert_eq!(1, got_err.code);
        assert_eq!("exited with status code: 1", got_err.to_string());

        Ok(())
    }
}
