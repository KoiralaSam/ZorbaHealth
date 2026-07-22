from phone import canonical_phone_digits, normalize_phone_digits


def test_canonical_phone_digits_nanp() -> None:
    assert canonical_phone_digits("3185125670") == "13185125670"
    assert canonical_phone_digits("+13185125670") == "13185125670"
    assert canonical_phone_digits("13185125670") == "13185125670"
    assert canonical_phone_digits("447911123456") == "447911123456"


def test_normalize_phone_digits() -> None:
    assert normalize_phone_digits("+1 (318) 512-5670") == "13185125670"
