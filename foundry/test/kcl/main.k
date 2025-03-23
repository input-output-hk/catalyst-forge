
schema Deployment:
    env: str
    instance: str
    name: str
    namespace: str
    values: any
    version: str

_deployment: any = option("deployment", default=None)
_d = Deployment { **_deployment }
_d