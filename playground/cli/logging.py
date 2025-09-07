"""
Central logging configuration for the Playground CLI.

This module provides the logging system that supports:
- Console output for user-facing messages
- File output for detailed logs
- Task-specific log files
- Command execution logging
- Configurable log levels and output destinations

Usage:
    from .logging import configure_logging, get_logger, get_task_logger

    # Configure at application startup
    configure_logging(log_dir=Path("logs/latest"))

    # Get loggers for different purposes
    logger = get_logger("console")  # For user messages
    task_logger = get_task_logger("k3d")  # For task-specific logging
    cmd_logger = get_logger("command")  # For command execution
"""

import logging
import logging.handlers
import sys
from pathlib import Path
from typing import Optional, Dict


class PlaygroundLogger:
    """Central logging configuration for the Playground CLI."""

    def __init__(self):
        self._configured = False
        self._log_dir: Optional[Path] = None
        self._task_loggers: Dict[str, logging.Logger] = {}

    def configure(
        self,
        log_dir: Optional[Path] = None,
        console_level: int = logging.INFO,
        file_level: int = logging.DEBUG,
        enable_debug: bool = False,
        json_format: bool = False,
    ) -> None:
        """Configure the global logging system.

        Args:
            log_dir: Directory for log files
            console_level: Minimum level for console output
            file_level: Minimum level for file output
            enable_debug: Whether to show debug messages on console
            json_format: Whether to use JSON format for structured logging
        """
        if self._configured:
            return

        self._log_dir = log_dir
        self._configured = True

        # Clear any existing handlers from root logger
        root_logger = logging.getLogger()
        root_logger.handlers.clear()
        root_logger.setLevel(logging.DEBUG)

        # Console handler for user-facing messages
        console_handler = logging.StreamHandler(sys.stdout)
        console_handler.setLevel(console_level if not enable_debug else logging.DEBUG)

        if json_format:
            # JSON format for structured logging
            console_formatter = logging.Formatter(
                '{"timestamp": "%(asctime)s", "level": "%(levelname)s", "message": "%(message)s"}'
            )
        else:
            # Human-readable format
            console_formatter = logging.Formatter("%(levelname)s: %(message)s")

        console_handler.setFormatter(console_formatter)
        root_logger.addHandler(console_handler)

        # General file handler for all logs
        if log_dir:
            log_dir.mkdir(parents=True, exist_ok=True)

            # Main log file with rotation
            file_handler = logging.handlers.RotatingFileHandler(
                log_dir / "playground.log",
                maxBytes=10 * 1024 * 1024,  # 10MB
                backupCount=5,
            )
            file_handler.setLevel(file_level)

            if json_format:
                file_formatter = logging.Formatter(
                    '{"timestamp": "%(asctime)s", "logger": "%(name)s", '
                    '"level": "%(levelname)s", "message": "%(message)s"}'
                )
            else:
                file_formatter = logging.Formatter(
                    "%(asctime)s %(name)s %(levelname)s: %(message)s"
                )

            file_handler.setFormatter(file_formatter)
            root_logger.addHandler(file_handler)

    def get_logger(self, name: str) -> logging.Logger:
        """Get a named logger.

        Args:
            name: Logger name (e.g., 'console', 'command', 'debug')

        Returns:
            Configured logger instance
        """
        return logging.getLogger(name)

    def get_task_logger(self, task_name: str) -> logging.Logger:
        """Get or create a task-specific logger with file output.

        Args:
            task_name: Name of the task

        Returns:
            Logger that writes to both general log and task-specific file
        """
        if task_name in self._task_loggers:
            return self._task_loggers[task_name]

        logger = logging.getLogger(f"task.{task_name}")
        logger.setLevel(logging.DEBUG)

        # Add task-specific file handler if log directory is configured
        if self._log_dir:
            task_log_file = self._log_dir / f"{task_name}.log"
            task_handler = logging.FileHandler(task_log_file)
            task_handler.setLevel(logging.DEBUG)
            task_formatter = logging.Formatter("%(asctime)s: %(message)s")
            task_handler.setFormatter(task_formatter)
            logger.addHandler(task_handler)

        self._task_loggers[task_name] = logger
        return logger

    def get_command_logger(self) -> logging.Logger:
        """Get logger for command execution details."""
        return self.get_logger("command")

    def is_configured(self) -> bool:
        """Check if logging has been configured."""
        return self._configured

    def get_log_dir(self) -> Optional[Path]:
        """Get the current log directory."""
        return self._log_dir


# Global instance
_logger = PlaygroundLogger()


def configure_logging(**kwargs) -> None:
    """Configure the global logging system.

    Args:
        log_dir: Directory for log files
        console_level: Minimum level for console output
        file_level: Minimum level for file output
        enable_debug: Whether to show debug messages on console
        json_format: Whether to use JSON format
    """
    _logger.configure(**kwargs)


def get_logger(name: str) -> logging.Logger:
    """Get a named logger.

    Args:
        name: Logger name

    Returns:
        Logger instance
    """
    return _logger.get_logger(name)


def get_task_logger(task_name: str) -> logging.Logger:
    """Get a task-specific logger.

    Args:
        task_name: Name of the task

    Returns:
        Task logger instance
    """
    return _logger.get_task_logger(task_name)


def get_command_logger() -> logging.Logger:
    """Get logger for command execution details.

    Returns:
        Command logger instance
    """
    return _logger.get_command_logger()


def is_logging_configured() -> bool:
    """Check if logging has been configured.

    Returns:
        True if logging is configured
    """
    return _logger.is_configured()


def get_log_directory() -> Optional[Path]:
    """Get the current log directory.

    Returns:
        Path to log directory or None if not configured
    """
    return _logger.get_log_dir()


# Convenience functions for common logging operations
def info(message: str, logger_name: str = "console") -> None:
    """Log an info message.

    Args:
        message: Message to log
        logger_name: Name of logger to use
    """
    get_logger(logger_name).info(message)


def debug(message: str, logger_name: str = "console") -> None:
    """Log a debug message.

    Args:
        message: Message to log
        logger_name: Name of logger to use
    """
    get_logger(logger_name).debug(message)


def warning(message: str, logger_name: str = "console") -> None:
    """Log a warning message.

    Args:
        message: Message to log
        logger_name: Name of logger to use
    """
    get_logger(logger_name).warning(message)


def error(message: str, logger_name: str = "console") -> None:
    """Log an error message.

    Args:
        message: Message to log
        logger_name: Name of logger to use
    """
    get_logger(logger_name).error(message)
