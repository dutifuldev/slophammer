from dataclasses import dataclass


@dataclass(frozen=True)
class GreetingInput:
    name: str


def create_greeting(input_data: GreetingInput) -> str:
    name = input_data.name.strip()

    if not name:
        raise ValueError("name must not be empty")

    return f"Hello, {name}."
