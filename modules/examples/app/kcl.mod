[package]
name = "module"
edition = "v0.11.0"
version = "0.0.1"

[dependencies]
k8s = "1.31.2"
base = { oci = "oci://ghcr.io/input-output-hk/catalyst-forge/base", tag = "0.2.0", version = "0.2.0" }
