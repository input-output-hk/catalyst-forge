import mixins
import specs

import base.schemas

schema AppInstance(schemas.Instance):
    mixin [
        mixins.DeploymentMixin
    ]

    app: specs.App