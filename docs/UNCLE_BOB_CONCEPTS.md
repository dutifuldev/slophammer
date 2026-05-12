# Uncle Bob Concepts

This section turns the Uncle Bob reference material into reusable engineering
concepts for agent-assisted projects.

The source transcripts stay in [`docs/uncle-bob/`](uncle-bob/). These pages are
the working wiki version: what the idea means, why it matters, and how to apply
it in a repository.

## Concept Map

| Concept                                           | Use it when                                               | Main guardrail                                                |
| ------------------------------------------------- | --------------------------------------------------------- | ------------------------------------------------------------- |
| [Rule Files](RULE_FILES.md)                       | Writing `AGENTS.md`, `CLAUDE.md`, or project instructions | Keep rules short, concrete, and testable                      |
| [Acceptance Tests](ACCEPTANCE_TESTS.md)           | Defining product behavior                                 | Put behavior in executable tests outside the model context    |
| [Unit Tests](UNIT_TESTS.md)                       | Protecting code while agents edit quickly                 | Make breakage visible immediately                             |
| [Coverage And CRAP](COVERAGE_AND_CRAP.md)         | Pushing agents toward smaller, better-tested code         | Combine coverage and complexity instead of chasing one metric |
| [Mutation Testing](MUTATION_TESTING.md)           | Checking whether tests actually assert behavior           | Kill surviving mutants                                        |
| [Structural Review](STRUCTURAL_REVIEW.md)         | Reviewing agent output without reading every token        | Inspect structure, names, size, arguments, and dependencies   |
| [Duplication Detection](DUPLICATION_DETECTION.md) | Catching copy/paste growth                                | Compare code structure, not just text                         |
| [Dependency Checks](DEPENDENCY_CHECKS.md)         | Protecting architecture boundaries                        | Make forbidden dependencies fail automatically                |
| [Power Tools](POWER_TOOLS.md)                     | Framing AI coding responsibly                             | Treat agents as amplifiers that require guards                |
| [Agentic Code Quality](AGENTIC_CODE_QUALITY.md)   | Managing AI coding at project scale                       | Surround the agent with external checks and feedback loops    |

## The Through Line

The common theme is not that AI agents should be trusted to follow advice.

The theme is that agents should work inside a system that makes quality visible:

- executable behavior specs
- fast unit tests
- high coverage
- mutation checks
- complexity limits
- dependency checks
- duplication detection
- small, focused rule files
- review of structure instead of syntax trivia

The human role changes from typing code to managing constraints. The better the
constraints, the less the project depends on the agent remembering an instruction
buried deep in a context window.

## Practical Default

For a new agent-assisted project, start with this loop:

1. Define behavior with acceptance tests.
2. Make the agent write unit tests before or with implementation.
3. Run coverage and close meaningful gaps.
4. Run CRAP or equivalent complexity-plus-coverage analysis.
5. Run mutation testing on risky code.
6. Check dependencies and module boundaries.
7. Review structure before reading implementation details.

That loop is the minimum useful shape of a guardrailed agentic workflow.
