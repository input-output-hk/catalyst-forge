import base.schemas

schema App(schemas.App):
    image: str
    replicas: int = 1
    port: int