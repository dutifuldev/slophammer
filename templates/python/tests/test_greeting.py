import pytest

from guardrails_template import GreetingInput, create_greeting


def test_create_greeting_trims_name() -> None:
    assert create_greeting(GreetingInput(name=" Ada ")) == "Hello, Ada."


def test_create_greeting_rejects_empty_name() -> None:
    with pytest.raises(ValueError, match="name must not be empty"):
        create_greeting(GreetingInput(name=" "))
