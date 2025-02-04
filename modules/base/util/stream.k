import schemas

import manifests

Stream = lambda inst: schemas.Instance -> any {
    manifests.yaml_stream([v for _, v in inst.manifests])
}