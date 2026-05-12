import { describe, expect, it } from "vitest";

import { createGreeting } from "../src/greeting.js";

describe("createGreeting", () => {
  it("greets a trimmed name", () => {
    expect(createGreeting({ name: " Ada " })).toBe("Hello, Ada.");
  });

  it("rejects an empty name", () => {
    expect(() => createGreeting({ name: " " })).toThrow("name must not be empty");
  });
});
