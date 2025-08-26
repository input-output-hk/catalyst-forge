from __future__ import annotations

import json
from typing import Any

from ..utils import log, run


def _get_localstack_pod(namespace: str = "localstack") -> str:
    cp = run(["kubectl", "-n", namespace, "get", "pods", "-o", "json"], capture=True)
    try:
        data = json.loads(cp.stdout or "{}")
        items = data.get("items", [])
        for item in items:
            phase = item.get("status", {}).get("phase")
            name = item.get("metadata", {}).get("name", "")
            if phase == "Running" and name.startswith("localstack-"):
                return name
    except Exception:
        pass
    raise SystemExit("LocalStack pod not found or not running in namespace 'localstack'")


def _awslocal(pod: str, args: list[str], namespace: str = "localstack") -> None:
    run(["kubectl", "-n", namespace, "exec", pod, "--", "awslocal", *args])


def _awslocal_capture(pod: str, args: list[str], namespace: str = "localstack") -> int:
    cp = run(
        ["kubectl", "-n", namespace, "exec", pod, "--", "awslocal", *args],
        check=False,
        capture=True,
    )
    return cp.returncode


def ensure_secret_json(name: str, payload: dict[str, Any], namespace: str = "localstack") -> None:
    pod = _get_localstack_pod(namespace)
    json_str = json.dumps(payload, separators=(",", ":"))
    # describe-secret returns non-zero if missing
    exists = (
        _awslocal_capture(
            pod, ["secretsmanager", "describe-secret", "--secret-id", name], namespace
        )
        == 0
    )
    if exists:
        log(f"Updating LocalStack secret: {name}")
        _awslocal(
            pod,
            [
                "secretsmanager",
                "put-secret-value",
                "--secret-id",
                name,
                "--secret-string",
                json_str,
            ],
            namespace,
        )
    else:
        log(f"Creating LocalStack secret: {name}")
        _awslocal(
            pod,
            [
                "secretsmanager",
                "create-secret",
                "--name",
                name,
                "--secret-string",
                json_str,
            ],
            namespace,
        )
