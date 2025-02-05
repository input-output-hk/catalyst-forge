# Example App Module

This directory contains a full example KCL module that uses the `base` module for ingesting input and producing the expectedo
output.
You can run it locally with:

```
forge mod template mod.cue
```

It will produce a single Kubernetes deployment that deploys an nginx instance.