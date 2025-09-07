"""
Utility functions for setupv2.
"""

import yaml
from pathlib import Path
import importlib
from typing import Dict, Any

from ...logging import get_logger


def load_tasks():
    """
    Import all task modules from tasks/ directory.

    Supports both legacy single-file format and new subpackage format:
    - Legacy: cert_manager.py
    - New: cert_manager/task.py

    This function dynamically imports all Python files in the tasks/
    directory, excluding files that start with underscore.
    """
    logger = get_logger("console")
    tasks_dir = Path(__file__).parent.parent / "tasks"

    if not tasks_dir.exists():
        return

    # Process subdirectories (new format)
    for task_subdir in sorted(tasks_dir.iterdir()):
        if task_subdir.is_dir() and not task_subdir.name.startswith("_"):
            task_py = task_subdir / "task.py"
            if task_py.exists():
                module_name = f"cli.setup.tasks.{task_subdir.name}.task"
                try:
                    importlib.import_module(module_name)
                    logger.info(f"  ✓ Loaded task module: {task_subdir.name} (subpackage)")
                except Exception as e:
                    logger.error(f"  ✗ Failed to load {task_subdir.name}: {e}")

    # Process legacy single files (for backward compatibility)
    for task_file in sorted(tasks_dir.glob("*.py")):
        if task_file.name.startswith("_"):
            continue

        # Skip if this is a legacy file that's been migrated to subpackage
        subdir = tasks_dir / task_file.stem
        if subdir.is_dir() and (subdir / "task.py").exists():
            logger.info(f"  → Skipping legacy {task_file.name} (migrated to subpackage)")
            continue

        module_name = f"cli.setup.tasks.{task_file.stem}"
        try:
            importlib.import_module(module_name)
            logger.info(f"  ✓ Loaded task module: {task_file.stem} (legacy)")
        except Exception as e:
            logger.error(f"  ✗ Failed to load {task_file.stem}: {e}")


def load_values(filename: str) -> Dict[str, Any]:
    """
    Load values from a YAML file in the values/ directory.

    Args:
        filename: Name of the YAML file (e.g., "argocd-base.yaml")

    Returns:
        Parsed YAML content as dictionary
    """
    values_dir = Path(__file__).parent.parent / "values"
    file_path = values_dir / filename

    if not file_path.exists():
        raise FileNotFoundError(f"Values file not found: {file_path}")

    with open(file_path, "r") as f:
        return yaml.safe_load(f)


def get_mkcert_caroot() -> Path:
    """Return the mkcert CA root directory.

    Returns:
        Path to the mkcert CA root directory containing rootCA.pem and rootCA-key.pem
    """
    from ...runner import CommandRunner

    cp = CommandRunner().run(["mkcert", "-CAROOT"], capture=True)
    return Path(str(cp.stdout or "").strip())
