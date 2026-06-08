pub fn pointer_is_null() -> bool {
    let value = unsafe { core::ptr::null::<u8>().as_ref() };
    value.is_none()
}

