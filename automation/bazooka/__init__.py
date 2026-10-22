"""Bazooka automation framework: write MUD bots in Python."""

from .bot import Bot, trigger
from .client import BazookaClient

__all__ = ["Bot", "BazookaClient", "trigger"]
