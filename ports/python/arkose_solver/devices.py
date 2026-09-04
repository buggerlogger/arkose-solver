"""Per-solve device generation from the embedded weighted population.

devices.json is the same file the Go implementation embeds: 234 GPUs,
105 screens, 13 core counts, 4 memory sizes, 62 connection profiles.
"""

import json
import os
import random

_HERE = os.path.dirname(os.path.abspath(__file__))
_POOL = None

CHROME_COLOR_DEPTH = 24
INNER_WIDTH_INSET = 16
INNER_HEIGHT_INSET = 95


def _pool():
    global _POOL
    if _POOL is None:
        with open(os.path.join(_HERE, "devices.json"), encoding="utf-8") as fh:
            _POOL = json.load(fh)
        for name in ("gpus", "screens", "cores", "memory", "network"):
            if not _POOL.get(name):
                raise RuntimeError("devices.json is missing pool %r" % name)
    return _POOL


def _pick(entries):
    total = sum(e["weight"] for e in entries if e.get("weight", 0) > 0)
    if total <= 0:
        return entries[0]
    r = random.randrange(total)
    for e in entries:
        w = e.get("weight", 0)
        if w <= 0:
            continue
        r -= w
        if r < 0:
            return e
    return entries[0]


def js_heap_size_limit(device_memory_gb):
    """enhanced_fp["basfas"] carries performance.memory.jsHeapSizeLimit."""
    if device_memory_gb >= 8:
        return 4395630592
    if device_memory_gb >= 4:
        return 4294705152
    return 2197815296


def _window_on(screen):
    avail_w = screen.get("availWidth") or screen["width"]
    avail_h = screen.get("availHeight") or screen["height"]
    if random.randrange(100) < 35:
        return avail_w, avail_h
    w = avail_w * random.randint(60, 100) // 100
    h = avail_h * random.randint(60, 100) // 100
    w = max(w, INNER_WIDTH_INSET + 200)
    h = max(h, INNER_HEIGHT_INSET + 200)
    return w, h


def generate_device():
    """A fresh, internally coherent synthetic machine."""
    pool = _pool()
    gpu = _pick(pool["gpus"])
    screen = _pick(pool["screens"])
    net = _pick(pool["network"])
    outer_w, outer_h = _window_on(screen)

    return {
        "gpu_vendor": gpu["vendor"],
        "gpu_model": gpu["model"],
        "gl_vendor": gpu["glVendor"],
        "gl_renderer": gpu["glRenderer"],
        "screen_width": screen["width"],
        "screen_height": screen["height"],
        "avail_width": screen.get("availWidth") or screen["width"],
        "avail_height": screen.get("availHeight") or screen["height"],
        "color_depth": CHROME_COLOR_DEPTH,
        "device_pixel_ratio": screen.get("dpr", 1),
        "outer_width": outer_w,
        "outer_height": outer_h,
        "inner_width": outer_w - INNER_WIDTH_INSET,
        "inner_height": outer_h - INNER_HEIGHT_INSET,
        "device_memory": _pick(pool["memory"])["value"],
        "hardware_concurrency": _pick(pool["cores"])["value"],
        "downlink": net["downlink"],
        "rtt": net["rtt"],
        "save_data": net.get("saveData", False),
    }


def pool_sizes():
    p = _pool()
    return {k: len(p[k]) for k in ("gpus", "screens", "cores", "memory", "network")}
