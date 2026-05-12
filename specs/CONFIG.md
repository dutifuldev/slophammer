# Config

Slophammer does not load project config in the first implementation slice.

This file exists to reserve the shared contract location. When config is added,
it should define:

- config file discovery
- config file format
- rule selection and severity overrides
- language-specific defaults
- behavior for invalid config

Until then, implementations should run the default shared rule set without
project-local configuration.
