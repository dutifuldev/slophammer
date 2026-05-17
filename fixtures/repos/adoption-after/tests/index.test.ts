import { formatUser } from "../src/index.js";

test("formats a user name", () => {
  expect(formatUser({ firstName: "Ada", lastName: "Lovelace" })).toBe("Ada Lovelace");
});
