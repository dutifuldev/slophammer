export type GreetingInput = {
  name: string;
};

export function createGreeting(input: GreetingInput): string {
  const name = input.name.trim();

  if (name.length === 0) {
    throw new Error("name must not be empty");
  }

  return `Hello, ${name}.`;
}
