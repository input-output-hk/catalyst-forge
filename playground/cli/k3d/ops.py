from __future__ import annotations

import json
import os
from pathlib import Path
from typing import Sequence

import yaml

from ..runner import CommandRunner
from ..models import ClusterSummary
from ..utils import err


def cluster_exists(name: str) -> bool:
	runner = CommandRunner()
	try:
		cp = runner.run(["k3d", "cluster", "list", "-o", "json"], capture=True)
		data = json.loads(cp.stdout or "{}")
		clusters = data.get("clusters", [])
		return any(c.get("name") == name for c in clusters)
	except Exception:
		cp = runner.run(["k3d", "cluster", "list"], capture=True)
		return name in (cp.stdout or "")


def _build_k3d_create_args(
	name: str,
	servers: int,
	agents: int,
	http_port: int,
	https_port: int,
	api_port: int,
) -> list[str]:
	args = [
		"k3d",
		"cluster",
		"create",
		name,
		"--servers",
		str(servers),
		"--agents",
		str(agents),
		"--k3s-arg",
		"--disable=traefik@server:0",
		"-p",
		f"{http_port}:80@loadbalancer",
		"-p",
		f"{https_port}:443@loadbalancer",
		"-p",
		"8372:8372@loadbalancer",
		"-p",
		"5432:5432@loadbalancer",
		"--wait",
	]
	if api_port != 0:
		args.extend(["--api-port", str(api_port)])
	return args


def create_cluster(
	name: str,
	servers: int,
	agents: int,
	http_port: int,
	https_port: int,
	api_port: int,
	extra_volumes: list[str] | None = None,
) -> None:
	args = _build_k3d_create_args(name, servers, agents, http_port, https_port, api_port)
	for vol in extra_volumes or []:
		args.extend(["--volume", vol])
	CommandRunner().run(args)


def delete_cluster(name: str) -> None:
	CommandRunner().run(["k3d", "cluster", "delete", name], check=False)


def write_kubeconfig(name: str, out_path: Path, assume_yes: bool) -> None:
	cp = CommandRunner().run(["k3d", "kubeconfig", "get", name], capture=True)
	text = str(cp.stdout or "")
	out_path.parent.mkdir(parents=True, exist_ok=True)
	if out_path.exists():
		try:
			current = out_path.read_text()
			if current == text:
				return
		except Exception:
			pass
	out_path.write_text(text)


def wait_for_nodes_ready(kubeconfig: Path, timeout_s: int = 120) -> None:
	from kubernetes import client as k8s_client, config as k8s_config  # type: ignore
	import time

	try:
		k8s_config.load_kube_config(config_file=str(kubeconfig))
	except Exception:
		err(f"Unable to load kubeconfig at {kubeconfig}")
		raise

	api = k8s_client.CoreV1Api()
	start = time.time()
	while time.time() - start < timeout_s:
		try:
			nodes = api.list_node().items
			if nodes and all(
				any(
					getattr(c, "type", None) == "Ready" and getattr(c, "status", None) == "True"
					for c in (n.status.conditions or [])
				)
				for n in nodes
			):
				return
		except Exception:
			pass
		time.sleep(2)
	err("Nodes did not become Ready within the timeout")
	raise SystemExit(1)


def get_k8s_version(kubeconfig: Path) -> str:
	try:
		from kubernetes import client as k8s_client, config as k8s_config  # type: ignore

		k8s_config.load_kube_config(config_file=str(kubeconfig))
		v = k8s_client.VersionApi().get_code()
		parts: list[str] = []
		if getattr(v, "git_version", None):
			parts.append(str(v.git_version))
		elif getattr(v, "major", None) and getattr(v, "minor", None):
			parts.append(f"v{v.major}.{v.minor}")
		if getattr(v, "platform", None):
			parts.append(f"({v.platform})")
		return " ".join(parts) if parts else ""
	except Exception:
		return ""


def emit_cluster_json(
	path: Path,
	name: str,
	servers: int,
	agents: int,
	http_port: int,
	https_port: int,
	kubeconfig: Path,
) -> None:
	summary = ClusterSummary(
		name=name,
		type="k3d",
		servers=servers,
		agents=agents,
		host_ip="127.0.0.1",
		http_port=http_port,
		https_port=https_port,
		kubeconfig=str(kubeconfig),
		kubernetes_version=get_k8s_version(kubeconfig),
	)
	path.parent.mkdir(parents=True, exist_ok=True)
	path.write_text(summary.model_dump_json(by_alias=True, indent=2))


def get_mkcert_caroot() -> Path:
	cp = CommandRunner().run(["mkcert", "-CAROOT"], capture=True)
	return Path(str(cp.stdout or "").strip())


def write_registries_yaml(tmpdir: Path, registry_host: str, ca_path: Path) -> Path:
	data = {
		"mirrors": {registry_host: {"endpoint": [f"https://{registry_host}"]}},
		"configs": {registry_host: {"tls": {"ca_file": str(ca_path)}}},
	}
	path = tmpdir / "registries.yaml"
	path.write_text(yaml.safe_dump(data, sort_keys=False))
	return path
