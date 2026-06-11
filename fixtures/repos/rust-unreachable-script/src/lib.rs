pub fn message() -> &'static str {
    "ok"
}

#[cfg(test)]
mod tests {
    #[test]
    fn message_is_ok() {
        assert_eq!(super::message(), "ok");
    }
}

