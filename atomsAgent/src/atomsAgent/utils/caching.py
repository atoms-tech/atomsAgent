from __future__ import annotations

import functools
import time
from collections.abc import Callable
from typing import Any, TypeVar

T = TypeVar("T")


def ttl_cache(ttl_seconds: int) -> Callable[[Callable[..., T]], Callable[..., T]]:
    """Simple TTL cache decorator for synchronous functions."""

    def decorator(func: Callable[..., T]) -> Callable[..., T]:
        cache: dict[str, tuple[float, T]] = {}

        @functools.wraps(func)
        def wrapper(*args: Any, **kwargs: Any) -> T:
            key = repr((args, kwargs))
            now = time.time()
            if key in cache:
                timestamp, value = cache[key]
                if now - timestamp < ttl_seconds:
                    return value
            value = func(*args, **kwargs)
            cache[key] = (now, value)
            return value

        return wrapper

    return decorator
