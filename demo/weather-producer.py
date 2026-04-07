#!/usr/bin/env python3
"""
weather-producer.py

Fetches live weather data from the Open-Meteo API (no API key required)
and produces structured log lines to per-city Kafka topics every 30 seconds.

Topics: new-york, chicago, sf, la, indy

Log line format:
  [2026-03-28T15:30:00Z] WEATHER location="new-york" lat=40.71 lon=-74.01
      temp=8.3 wind_speed=14.2 wind_dir=270 weather_code=3 is_day=1
"""

import os
import time
import json
import urllib.request
import urllib.error
from typing import Optional
from datetime import datetime, timezone
from kafka import KafkaProducer

BOOTSTRAP = os.environ.get("BOOTSTRAP", "localhost:9092")

CITIES = [
    ("new-york", 40.71, -74.01),
    ("chicago",  41.88, -87.63),
    ("sf",       37.77, -122.42),
    ("la",       34.05, -118.24),
    ("indy",     39.77, -86.16),
]


def log(msg: str) -> None:
    print(f"[weather-producer] {msg}", flush=True)


def fetch_weather(city: str, lat: float, lon: float) -> Optional[str]:
    url = (
        f"https://api.open-meteo.com/v1/forecast"
        f"?latitude={lat}&longitude={lon}&current_weather=true"
    )
    try:
        with urllib.request.urlopen(url, timeout=10) as resp:
            data = json.loads(resp.read())
    except (urllib.error.URLError, json.JSONDecodeError) as exc:
        log(f"Failed to fetch weather for {city}: {exc}")
        return None

    cw = data.get("current_weather", {})
    ts = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
    return (
        f'[{ts}] WEATHER location="{city}" lat={lat} lon={lon}'
        f' temp={cw.get("temperature")} wind_speed={cw.get("windspeed")}'
        f' wind_dir={cw.get("winddirection")} weather_code={cw.get("weathercode")}'
        f' is_day={cw.get("is_day")}'
    )


def main() -> None:
    log(f"Connecting to Kafka at {BOOTSTRAP}...")
    producer = KafkaProducer(
        bootstrap_servers=BOOTSTRAP,
        value_serializer=lambda v: v.encode("utf-8"),
    )
    log(f"Connected. Producing to topics: {[c[0] for c in CITIES]}")

    while True:
        batch_start = time.monotonic()
        for city, lat, lon in CITIES:
            line = fetch_weather(city, lat, lon)
            if line is None:
                continue
            producer.send(city, value=line)
            log(f"Produced to {city}: {line}")
        producer.flush()
        elapsed = time.monotonic() - batch_start
        sleep_for = max(0.0, 30.0 - elapsed)
        time.sleep(sleep_for)


if __name__ == "__main__":
    main()
