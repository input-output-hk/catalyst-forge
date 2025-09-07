# ruff: noqa: E402
from __future__ import annotations

"""CommandRunner: a robust subprocess execution wrapper for the Playground CLI.

Provides consistent command execution with optional capture, timeouts, working
directory control, environment merging, and simple argument redaction in logs.
"""

import os
import shlex
import subprocess
import time
from pathlib import Path
from typing import Mapping, Sequence

from .setup.logging import get_command_logger


class CommandRunner:
    """Execute external commands with sensible defaults and logging.

    Notes:
        - When capture=True, stdout/stderr are captured as text.
        - When check=True, non-zero exit codes raise CalledProcessError.
        - A timeout (in seconds) can be specified; exceeding it raises TimeoutExpired.
        - Redaction is a simple token replace in the logged command line.
    """

    def __init__(self, default_timeout: int | None = None) -> None:
        self.default_timeout = default_timeout
        self._cmd_logger = get_command_logger()

    def _format_cmd(self, args: Sequence[str], redact: Sequence[str] | None) -> str:
        redactions = set(redact or [])
        quoted = []
        for a in args:
            qa = shlex.quote(a)
            for token in redactions:
                if token and token in qa:
                    qa = qa.replace(token, "****")
            quoted.append(qa)
        return " ".join(quoted)

    def run(
        self,
        args: Sequence[str],
        *,
        capture: bool = False,
        check: bool = True,
        env: Mapping[str, str] | None = None,
        cwd: str | Path | None = None,
        timeout: int | None = None,
        redact: Sequence[str] | None = None,
    ) -> subprocess.CompletedProcess:
        cmd_str = self._format_cmd(args, redact)
        started = time.time()
        effective_env = {**os.environ, **(dict(env) if env else {})}

        if capture:
            cp = subprocess.run(
                list(args),
                check=check,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                text=True,
                env=effective_env,
                cwd=str(cwd) if cwd is not None else None,
                timeout=timeout or self.default_timeout,
            )
        else:
            cp = subprocess.run(
                list(args),
                check=check,
                text=True,
                env=effective_env,
                cwd=str(cwd) if cwd is not None else None,
                timeout=timeout or self.default_timeout,
            )

        duration = time.time() - started
        self._cmd_logger.log_command_start(cmd_str)
        self._cmd_logger.log_command_end(cmd_str, cp.returncode, duration)
        return cp
