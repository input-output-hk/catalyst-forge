"""
Logging utilities for setupv2 tasks.

This module provides utilities for task logging including context managers
and file-like objects for structured logging.
"""

import logging
from contextlib import contextmanager
from typing import IO, Generator, Optional
from pathlib import Path


class TaskLogWriter:
    """File-like object that writes to logger and optional file handle.

    This class provides a unified interface for task logging, routing messages
    through the logging system while optionally maintaining file output.
    """

    def __init__(self, logger: logging.Logger, file_handle: Optional[IO] = None):
        """Initialize the log writer.

        Args:
            logger: Logger instance to write to
            file_handle: Optional file handle for additional output
        """
        self.logger = logger
        self.file_handle = file_handle
        self._buffer: list[str] = []

    def write(self, text: str) -> None:
        """Write text to both logger and file handle.

        Args:
            text: Text to write
        """
        if not text.strip():
            return

        # Write to logger (this goes to all configured handlers)
        self.logger.debug(text.rstrip())

        # Write to file handle if provided
        if self.file_handle:
            self.file_handle.write(text)
            self.file_handle.flush()

        # Keep in buffer for reference
        self._buffer.append(text)

    def flush(self) -> None:
        """Flush any buffered content."""
        if self.file_handle:
            self.file_handle.flush()

    def fileno(self) -> int:
        """Expose underlying file descriptor when a real file is available.

        This allows passing TaskLogWriter to subprocess as stdout/stderr.
        """
        if self.file_handle and hasattr(self.file_handle, "fileno"):
            return self.file_handle.fileno()  # type: ignore[no-any-return]
        raise OSError("TaskLogWriter has no underlying file descriptor")

    def get_content(self) -> str:
        """Get accumulated content.

        Returns:
            Accumulated content as string
        """
        return "".join(self._buffer)

    def close(self) -> None:
        """Close the file handle if it exists."""
        if self.file_handle:
            self.file_handle.close()


@contextmanager
def task_logging_context(
    task_name: str, log_dir: Optional[Path] = None
) -> Generator[TaskLogWriter, None, None]:
    """Context manager that provides file-like logging for tasks.

    This context manager provides a file-like object to task functions that routes
    logging through the logging system while optionally maintaining file output.

    Args:
        task_name: Name of the task
        log_dir: Directory for log files (optional)

    Yields:
        File-like object for logging
    """
    from cli.logging import get_task_logger

    logger = get_task_logger(task_name)
    file_handle = None

    # Optionally create a file handle for backward compatibility
    if log_dir:
        log_dir.mkdir(parents=True, exist_ok=True)
        log_file = log_dir / f"{task_name}.log"
        file_handle = open(log_file, "a")

    try:
        writer = TaskLogWriter(logger, file_handle)
        yield writer
    finally:
        if file_handle:
            file_handle.close()


def setup_task_logging(task_name: str, log_dir: Optional[Path] = None) -> TaskLogWriter:
    """Set up logging for a task and return a file-like object.

    This function sets up logging for a task and returns a file-like object
    that can be used for structured logging output.

    Args:
        task_name: Name of the task
        log_dir: Directory for log files (optional)

    Returns:
        File-like object for logging
    """
    from cli.logging import get_task_logger

    logger = get_task_logger(task_name)

    if log_dir:
        log_dir.mkdir(parents=True, exist_ok=True)
        log_file = log_dir / f"{task_name}.log"
        file_handle = open(log_file, "a")
        return TaskLogWriter(logger, file_handle)
    else:
        return TaskLogWriter(logger)


class CommandLogger:
    """Logger for command execution with structured output."""

    def __init__(self, logger: logging.Logger):
        """Initialize command logger.

        Args:
            logger: Logger instance to use
        """
        self.logger = logger

    def log_command_start(self, command: str, **kwargs) -> None:
        """Log the start of a command execution.

        Args:
            command: Command being executed
            **kwargs: Additional context
        """
        self.logger.debug(f"Starting command: {command}", extra=kwargs)

    def log_command_end(self, command: str, exit_code: int, duration: float, **kwargs) -> None:
        """Log the end of a command execution.

        Args:
            command: Command that was executed
            exit_code: Exit code of the command
            duration: Duration of execution in seconds
            **kwargs: Additional context
        """
        level = logging.DEBUG if exit_code == 0 else logging.WARNING
        self.logger.log(
            level,
            f"Command completed: {command} (exit={exit_code}, duration={duration:.2f}s)",
            extra=kwargs,
        )

    def log_command_output(self, command: str, output: str, is_stderr: bool = False) -> None:
        """Log command output.

        Args:
            command: Command that produced the output
            output: Output text
            is_stderr: Whether this is stderr output
        """
        if output.strip():
            stream = "stderr" if is_stderr else "stdout"
            self.logger.debug(f"Command {stream}: {output.rstrip()}")


def get_command_logger() -> CommandLogger:
    """Get a command logger instance.

    Returns:
        CommandLogger instance
    """
    from cli.logging import get_command_logger

    return CommandLogger(get_command_logger())
