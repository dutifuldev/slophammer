use serde::{Deserialize, Serialize};
use std::fmt;

pub const EXIT_OK: i32 = 0;
pub const EXIT_FINDINGS: i32 = 1;
pub const EXIT_ERROR: i32 = 2;

#[derive(Clone, Copy, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(rename_all = "lowercase")]
pub enum Severity {
    Error,
    Warn,
}

impl fmt::Display for Severity {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Error => formatter.write_str("error"),
            Self::Warn => formatter.write_str("warn"),
        }
    }
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
pub struct Finding {
    pub rule_id: String,
    pub severity: Severity,
    pub path: String,
    pub message: String,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub baselined: Option<bool>,
}

impl Finding {
    pub fn new(definition: &RuleDefinition) -> Self {
        Self {
            rule_id: definition.id.to_owned(),
            severity: definition.severity,
            path: definition.path.to_owned(),
            message: definition.message.to_owned(),
            baselined: None,
        }
    }

    pub fn at_path(definition: &RuleDefinition, path: impl Into<String>) -> Self {
        Self {
            path: path.into(),
            ..Self::new(definition)
        }
    }

    pub fn with_message(
        definition: &RuleDefinition,
        path: impl Into<String>,
        message: impl Into<String>,
    ) -> Self {
        Self {
            path: path.into(),
            message: message.into(),
            ..Self::new(definition)
        }
    }
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
pub struct Report {
    pub ok: bool,
    pub findings: Vec<Finding>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub scope: Option<ScopeCoverage>,
}

/// Coverage of configured scope over the ecosystem's production files,
/// reported so a narrowed scope is visible instead of silent.
#[derive(Clone, Copy, Debug, Deserialize, Eq, PartialEq, Serialize)]
pub struct ScopeCoverage {
    pub scanned: usize,
    pub production_files: usize,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
pub struct RuleDefinition {
    pub id: &'static str,
    pub title: &'static str,
    pub category: &'static str,
    pub severity: Severity,
    pub path: &'static str,
    pub message: &'static str,
    pub description: &'static str,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub tool: Option<&'static str>,
    pub status: &'static str,
}

pub fn find_definition<'a>(
    definitions: &'a [RuleDefinition],
    rule_id: &str,
) -> Option<&'a RuleDefinition> {
    definitions
        .iter()
        .find(|definition| definition.id == rule_id)
}
