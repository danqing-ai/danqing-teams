"""Intentionally buggy: should print 42."""


def answer() -> int:
    # BUG: off-by-one and wrong operator — should return 42
    values = [6, 7]
    return values[0] * values[1] + 1  # currently 43


if __name__ == "__main__":
    print(answer())
