from verification import OTP_LENGTH, normalize_otp


def test_normalize_otp_accepts_six_digits() -> None:
    assert normalize_otp("123456") == "123456"
    assert normalize_otp("12-34-56") == "123456"


def test_normalize_otp_rejects_wrong_length() -> None:
    assert normalize_otp("12345") is None
    assert normalize_otp("1234567") is None
    assert normalize_otp("") is None


def test_otp_length_constant() -> None:
    assert OTP_LENGTH == 6
