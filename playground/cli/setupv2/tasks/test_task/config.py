"""
Configuration models for test_task package.
"""

from pydantic import BaseModel


class TestTaskConfig(BaseModel):
    """Configuration for test tasks.

    This model defines parameters for testing the setup v2 system.
    """

    enable_delay: bool = True
    """Whether to include delays in test tasks.

    When enabled, test tasks will include sleep statements to simulate
    real-world processing time.
    Default: True
    """

    test_message: str = "Hello from test task"
    """Test message used by test tasks.

    This message is used in test task processing.
    Default: "Hello from test task"
    """

    test_value: int = 42
    """Test value used by test tasks.

    This value is used in test task processing.
    Default: 42
    """
