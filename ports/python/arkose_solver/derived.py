"""Fingerprint values the enforcement derives rather than reads.

Each recipe was recovered from the deobfuscated bundle and reproduces what a
real Chrome sends. Conformance values live in ports/testvectors.json.
"""

import hashlib
import json
import os
import random

from .hashes import khash

_HERE = os.path.dirname(os.path.abspath(__file__))
_CODECS = None


def md5(s):
    return hashlib.md5(s.encode("utf-8")).hexdigest()


def compute_webgl_extensions_hash(extensions):
    return khash(extensions, 0)


def compute_browser_object_checks(present):
    """md5 of the known browser globals that exist, sorted and comma-joined."""
    if not present:
        return None
    return md5(",".join(sorted(present)))


CHROME_WINDOWS_FEATURES = {
    "permission_status": True,
    "eye_dropper": True,
    "audio_data": True,
    "writable_stream": True,
    "css_style_rule": True,
    "navigator_ua": True,
    "barcode_detector": False,
    "display_names": True,
    "contacts_manager": False,
    "svg_discard_element": False,
    "usb": True,
    "media_device": True,
    "playback_quality": True,
}

_F58835F_ORDER = [
    "permission_status", "eye_dropper", "audio_data", "writable_stream",
    "css_style_rule", "navigator_ua", "barcode_detector", "display_names",
    "contacts_manager", "svg_discard_element", "usb", "media_device",
    "playback_quality",
]
_F58835F_DEFINED = {"usb", "media_device"}


def compute_f58835f(features=None):
    """md5(JSON.stringify([...])) over the 13 feature-probe strings."""
    f = features or CHROME_WINDOWS_FEATURES
    parts = []
    for name in _F58835F_ORDER:
        v = f[name]
        if name in _F58835F_DEFINED:
            parts.append("%s: %s" % (name, "defined" if v else "NA"))
        else:
            parts.append("%s: %s" % (name, "true" if v else "false"))
    return md5(json.dumps(parts, separators=(",", ":")))


def compute_speech(voices):
    """Both speech fields come from one voice list, so they always agree."""
    if not voices:
        return None, None
    default_voice = None
    pairs = []
    for v in voices:
        if v.get("default"):
            default_voice = "%s || %s" % (v["name"], v["lang"])
        pairs.append("%s,%s" % (v["name"], v["lang"]))
    return default_voice, md5(",".join(pairs))


WINDOWS_VOICE_SETS = [
    [{"name": "Microsoft David - English (United States)", "lang": "en-US", "default": True},
     {"name": "Microsoft Zira - English (United States)", "lang": "en-US"}],
    [{"name": "Microsoft David - English (United States)", "lang": "en-US", "default": True},
     {"name": "Microsoft Mark - English (United States)", "lang": "en-US"},
     {"name": "Microsoft Zira - English (United States)", "lang": "en-US"}],
    [{"name": "Microsoft Zira - English (United States)", "lang": "en-US", "default": True},
     {"name": "Microsoft David - English (United States)", "lang": "en-US"}],
    [{"name": "Microsoft David - English (United States)", "lang": "en-US", "default": True},
     {"name": "Microsoft Zira - English (United States)", "lang": "en-US"},
     {"name": "Microsoft Hazel - English (United Kingdom)", "lang": "en-GB"},
     {"name": "Microsoft George - English (United Kingdom)", "lang": "en-GB"}],
    [{"name": "Microsoft David - English (United States)", "lang": "en-US", "default": True},
     {"name": "Microsoft Zira - English (United States)", "lang": "en-US"},
     {"name": "Microsoft Helena - Spanish (Spain)", "lang": "es-ES"},
     {"name": "Microsoft Laura - Spanish (Spain)", "lang": "es-ES"},
     {"name": "Microsoft Pablo - Spanish (Spain)", "lang": "es-ES"}],
]

LOCALE_VOICES = {
    "en-GB": "Microsoft George - English (United Kingdom)",
    "es-ES": "Microsoft Helena - Spanish (Spain)",
    "pt-BR": "Microsoft Daniel - Portuguese (Brazil)",
    "de-DE": "Microsoft Hedda - German (Germany)",
    "fr-FR": "Microsoft Hortense - French (France)",
    "it-IT": "Microsoft Elsa - Italian (Italy)",
    "ru-RU": "Microsoft Irina - Russian (Russia)",
    "tr-TR": "Microsoft Tolga - Turkish (Turkiye)",
}


def pick_voice_set(lang_tag):
    base = random.choice(WINDOWS_VOICE_SETS)
    if lang_tag in ("en-US", "en"):
        return base
    name = LOCALE_VOICES.get(lang_tag)
    if not name:
        return base
    out = [{"name": name, "lang": lang_tag, "default": True}]
    for v in base:
        out.append({"name": v["name"], "lang": v["lang"]})
    return out


def _codecs():
    global _CODECS
    if _CODECS is None:
        with open(os.path.join(_HERE, "codecs.json"), encoding="utf-8") as fh:
            _CODECS = json.load(fh)
    return _CODECS


def _codec_probe_json(rows):
    """Rebuild the exact JSON.stringify output the browser hashes."""
    parts = []
    for r in rows:
        parts.append(
            json.dumps(r["mime"])
            + ':{"canPlay":' + json.dumps(r["canPlay"])
            + ',"mediaSource":' + ("true" if r["mediaSource"] else "false")
            + "}"
        )
    return "{" + ",".join(parts) + "}"


def audio_codecs_extended_hash():
    return md5(_codec_probe_json(_codecs()["audio"]))


def video_codecs_extended_hash():
    return md5(_codec_probe_json(_codecs()["video"]))
