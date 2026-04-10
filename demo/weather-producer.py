#!/usr/bin/env python3
"""
weather-producer.py

Fetches live weather data from the Open-Meteo API (no API key required)
and produces tab-separated records to per-city Kafka topics every 30 seconds.

Topics: new-york, chicago, sf, la, indy
"""

import os
import math
import time
import json
import urllib.request
import urllib.error
from typing import Optional
from datetime import datetime, timezone
from kafka import KafkaProducer

BOOTSTRAP = os.environ.get("BOOTSTRAP", "localhost:9092")

# (topic, lat, lon, icao_station_id)
CITIES = [
    ("new-york", 40.71, -74.01,  "KNYC"),
    ("chicago",  41.88, -87.63,  "KMDW"),
    ("sf",       37.77, -122.42, "KSFO"),
    ("la",       34.05, -118.24, "KLAX"),
    ("indy",     39.77, -86.16,  "KIND"),
]

CURRENT_FIELDS = ",".join([
    "temperature_2m",
    "apparent_temperature",
    "relative_humidity_2m",
    "precipitation",
    "weather_code",
    "cloud_cover",
    "surface_pressure",
    "wind_speed_10m",
    "wind_direction_10m",
    "wind_gusts_10m",
    "is_day",
])

HOURLY_FIELDS = ",".join([
    "uv_index",
    "visibility",
    "soil_temperature_0cm",
])


def log(msg: str) -> None:
    print(f"[weather-producer] {msg}", flush=True)


def dew_point(temp_c: float, humidity: int) -> float:
    """Magnus formula approximation for dew point (°C)."""
    a, b = 17.625, 243.04
    rh = max(1, min(100, humidity))
    alpha = (a * temp_c / (b + temp_c)) + math.log(rh / 100.0)
    return round((b * alpha) / (a - alpha), 1)


def fetch_weather(city: str, lat: float, lon: float, station: str) -> Optional[str]:
    url = (
        f"https://api.open-meteo.com/v1/forecast"
        f"?latitude={lat}&longitude={lon}"
        f"&current={CURRENT_FIELDS}"
        f"&hourly={HOURLY_FIELDS}"
        f"&timezone=UTC&forecast_days=1"
    )
    try:
        with urllib.request.urlopen(url, timeout=15) as resp:
            data = json.loads(resp.read())
    except (urllib.error.URLError, json.JSONDecodeError) as exc:
        log(f"Failed to fetch weather for {city}: {exc}")
        return None

    cur = data.get("current", {})
    hourly = data.get("hourly", {})

    # Find current UTC hour index in the hourly arrays
    now_hour = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:00")
    times = hourly.get("time", [])
    hour_idx = 0
    for i, t in enumerate(times):
        if t.startswith(now_hour):
            hour_idx = i
            break

    def hval(field: str, default=None):
        arr = hourly.get(field, [])
        if arr and hour_idx < len(arr):
            return arr[hour_idx]
        return default

    temp    = cur.get("temperature_2m")
    feels   = cur.get("apparent_temperature")
    humid   = cur.get("relative_humidity_2m")
    precip  = cur.get("precipitation", 0.0)
    wcode   = cur.get("weather_code")
    clouds  = cur.get("cloud_cover", 0)
    press   = cur.get("surface_pressure")
    wspeed  = cur.get("wind_speed_10m")
    wdir    = cur.get("wind_direction_10m")
    wgusts  = cur.get("wind_gusts_10m")
    is_day  = cur.get("is_day", 0)

    dp         = dew_point(temp, humid) if temp is not None and humid is not None else 0.0
    uv         = round(hval("uv_index", 0.0), 1)
    visibility = int(hval("visibility", 10000))
    soil_temp  = round(hval("soil_temperature_0cm", temp), 1) if temp is not None else 0.0

    ts = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")

    return "\t".join([
        station,
        city,
        str(lat),
        str(lon),
        ts,
        str(temp),
        str(feels),
        str(humid),
        str(dp),
        str(wspeed),
        str(wdir),
        str(wgusts),
        str(press),
        str(clouds),
        str(visibility),
        str(uv),
        str(precip),
        str(wcode),
        str(is_day),
        str(soil_temp),
    ])


def main() -> None:
    log(f"Connecting to Kafka at {BOOTSTRAP}...")
    producer = KafkaProducer(
        bootstrap_servers=BOOTSTRAP,
        value_serializer=lambda v: v.encode("utf-8"),
    )
    log(f"Connected. Producing to topics: {[c[0] for c in CITIES]}")

    while True:
        batch_start = time.monotonic()
        for city, lat, lon, station in CITIES:
            line = fetch_weather(city, lat, lon, station)
            if line is None:
                continue
            producer.send(city, value=line)
            log(f"Produced to {city}: {line}")
        producer.flush()
        elapsed = time.monotonic() - batch_start
        time.sleep(max(0.0, 30.0 - elapsed))


if __name__ == "__main__":
    main()
