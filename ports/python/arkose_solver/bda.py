"""BDA assembly: the 91-field enhanced_fp payload plus fe/f/ife_hash.

Mirrors the Go implementation field for field and order for order.
"""

import base64
import hashlib
import os
import random
import time
import uuid

from .devices import CHROME_COLOR_DEPTH, generate_device, js_heap_size_limit
from .hashes import compute_f, compute_ife_hash
from .webgl import (
    SCREEN_PIXEL_DEPTH_FACTOR,
    WEBGL_EXTENSIONS_HASH,
    build_webgl_fields,
    compute_rtt_type,
    compute_webgl_hash,
)

MATH_FINGERPRINT = "e00752d866abf7fdf93bd4111bcfeb7b"
SUPPORTED_MATH_FUNCS = "3f7aaba900fde542f258166cd2b71ef5"
BROWSER_OBJECT_CHECKS = "554838a8451ac36cb977e719e9d6623c"
F58835F = "6681f2144d3d1ceb383fe87f7aad9600"
HASH_29S83IH9 = "68934a3e9455fa72420237eb05902327"
AUDIO_CODECS_EXT_HASH = "805036349642e2569ec299baed02315b"
VIDEO_CODECS_EXT_HASH = "5648501a58d24ea22ce3773cc234e563"
DEFAULT_ENFORCEMENT_HASH = "2d1c8a89671586563cb793ae2399b954"
DEFAULT_DOCUMENT_REFERRER = "https://www.google.com/"

WEBGL_VERSION = "WebGL 1.0 (OpenGL ES 2.0 Chromium)"
WEBGL_SHADING_LANGUAGE = "WebGL GLSL ES 1.0 (OpenGL ES GLSL ES 1.0 Chromium)"

AUDIO_FINGERPRINTS = [
    "124.04347776696522",
    "124.04347527516074",
    "124.04347527840094",
    "124.04347541281574",
    "124.0434806260746",
    "124.04347527516074",
    "124.0434752046262",
    "124.04344968475198",
    "35.7383295930922",
    "124.0434802826973",
]

TZ_POOL = [-180, -180, -180, -120, -120, -60, -60, 0, 0, 60,
           300, 300, 360, 420, 420, 480, 480, -480, -540, -330, -240]

LANG_CHOICES = [
    ("en-US", "en-US,en", 55), ("en-GB", "en-GB,en", 12), ("en", "en", 8),
    ("es-ES", "es-ES,es", 4), ("pt-BR", "pt-BR,pt", 4), ("de-DE", "de-DE,de", 4),
    ("fr-FR", "fr-FR,fr", 4), ("it-IT", "it-IT,it", 3), ("ru-RU", "ru-RU,ru", 3),
    ("tr-TR", "tr-TR,tr", 3),
]

VOICES = {
    "en-US": "Microsoft David - English (United States)",
    "en-GB": "Microsoft George - English (United Kingdom)",
    "en": "Microsoft David - English (United States)",
    "es-ES": "Microsoft Helena - Spanish (Spain)",
    "pt-BR": "Microsoft Daniel - Portuguese (Brazil)",
    "de-DE": "Microsoft Hedda - German (Germany)",
    "fr-FR": "Microsoft Hortense - French (France)",
    "it-IT": "Microsoft Elsa - Italian (Italy)",
    "ru-RU": "Microsoft Irina - Russian (Russia)",
    "tr-TR": "Microsoft Tolga - Turkish (Turkiye)",
}

FONT_SETS = [
    "Arial,Arial Black,Arial Narrow,Calibri,Cambria,Cambria Math,Comic Sans MS,Consolas,Courier,Courier New,Georgia,Helvetica,Impact,Lucida Console,Lucida Sans Unicode,Microsoft Sans Serif,MS Gothic,MS PGothic,MS Sans Serif,MS Serif,Palatino Linotype,Segoe Print,Segoe Script,Segoe UI,Segoe UI Light,Segoe UI Semibold,Segoe UI Symbol,Tahoma,Times,Times New Roman,Trebuchet MS,Verdana,Wingdings",
    "Arial,Arial Black,Arial Narrow,Calibri,Comic Sans MS,Consolas,Courier,Courier New,Georgia,Impact,Lucida Console,Lucida Sans Unicode,Microsoft Sans Serif,MS Gothic,MS PGothic,MS Sans Serif,MS Serif,Palatino Linotype,Segoe UI,Segoe UI Light,Segoe UI Symbol,Tahoma,Times,Times New Roman,Trebuchet MS,Verdana",
    "Arial,Arial Black,Arial Narrow,Calibri,Cambria,Cambria Math,Comic Sans MS,Consolas,Courier,Courier New,Georgia,Impact,Lucida Console,Lucida Sans Unicode,Marlett,Microsoft Sans Serif,MS Gothic,MS PGothic,MS Sans Serif,MS Serif,Palatino Linotype,Segoe Print,Segoe Script,Segoe UI,Segoe UI Light,Segoe UI Semibold,Segoe UI Symbol,Sylfaen,Tahoma,Times,Times New Roman,Trebuchet MS,Verdana,Webdings,Wingdings",
]

TREE_STRUCTURES = ["[[[],[]],[[]],[],[]]", "[[],[],[],[[]],[]]", "[[],[],[],[[]],[[]],[],[]]"]

GREASE_BRANDS = {152: "Not?A_Brand", 151: "Not_A Brand", 150: "Not)A;Brand", 149: "Not-A.Brand"}
CHROME_MAJORS = [148, 149, 150, 151, 152]


def _md5(s):
    return hashlib.md5(s.encode("utf-8")).hexdigest()


def _hash_field(salt, field):
    return _md5(salt + "|" + field)


def _weighted_lang():
    total = sum(w for _, _, w in LANG_CHOICES)
    r = random.randrange(total)
    for tag, full, w in LANG_CHOICES:
        r -= w
        if r < 0:
            return tag, full
    return LANG_CHOICES[0][0], LANG_CHOICES[0][1]


def new_session():
    """Per-solve identity: stable-within-a-solve hashes and locale."""
    salt = os.urandom(16).hex()
    tag, full = _weighted_lang()
    cfp = random.randint(-2147483648, 2147483647)
    return {
        "salt": salt,
        "machine_hash": _hash_field(salt, "machine_hash"),
        "hash_7541c2s": _hash_field(salt, "7541c2s"),
        "hash_1f220c9": _hash_field(salt, "1f220c9"),
        "hash_6a62b2a558": _hash_field(salt, "6a62b2a558"),
        "hash_c2d2015": _hash_field(salt, "c2d2015"),
        "hash_05d3d24": _hash_field(salt, "05d3d24"),
        "hash_83eb055": _hash_field(salt, "83eb055"),
        "speech_voices_hash": _hash_field(salt, "speech_voices"),
        "speech_default_voice": "%s || %s" % (VOICES.get(tag, VOICES["en-US"]),
                                              tag if tag in VOICES else "en-US"),
        "tz_offset": random.choice(TZ_POOL),
        "language_tag": tag,
        "languages": full,
        "font_list": random.choice(FONT_SETS),
        "cfp": cfp,
    }


def _format_dpr(v):
    return str(int(v)) if float(v) == int(v) else repr(float(v))


def build_fe(device, identity):
    fe = [
        "DNT:unknown",
        "L:" + identity["language_tag"],
        "D:%d" % device["color_depth"],
        "PR:%s" % _format_dpr(device["device_pixel_ratio"]),
        "S:%d,%d" % (device["screen_width"], device["screen_height"]),
        "AS:%d,%d" % (device["avail_width"], device["avail_height"]),
        "TO:%d" % identity["tz_offset"],
        "SS:true", "LS:true", "IDB:true", "B:false", "ODB:false",
        "CPUC:unknown", "PK:Win32",
        "CFP:%d" % identity["cfp"],
        "FR:false", "FOS:false", "FB:false",
        "JSF:" + identity["font_list"],
        "P:Chrome PDF Viewer,Chromium PDF Viewer,Microsoft Edge PDF Viewer,PDF Viewer,WebKit built-in PDF",
        "T:0,false,false",
        "H:%d" % device["hardware_concurrency"],
        "SWF:false",
    ]
    return fe, compute_f(fe)


def _shuffled_brands(major):
    brands = ["Chromium", "Google Chrome", GREASE_BRANDS.get(major, "Not/A)Brand")]
    random.shuffle(brands)
    return ",".join(brands)


def _item(key, value):
    return {"key": key, "value": value}


def build_enhanced_fp(cfg, identity, device, build_id):
    major = random.choice(CHROME_MAJORS)
    webgl_fields = build_webgl_fields(device["gl_vendor"], device["gl_renderer"])
    webgl_hash = compute_webgl_hash(webgl_fields)
    rtt_type = compute_rtt_type(WEBGL_EXTENSIONS_HASH, webgl_hash)

    now_ms = int(time.time() * 1000)
    surl = cfg.get("surl", "")

    out = [_item(k, v) for k, v in webgl_fields]
    out += [
        _item("webgl_hash_webgl", webgl_hash),
        _item("user_agent_data_brands", _shuffled_brands(major)),
        _item("user_agent_data_mobile", False),
        _item("navigator_connection_downlink", device["downlink"]),
        _item("navigator_connection_downlink_max", None),
        _item("network_info_rtt", device["rtt"]),
        _item("network_info_save_data", device["save_data"]),
        _item("network_info_rtt_type", rtt_type),
        _item("screen_pixel_depth", device["color_depth"] * SCREEN_PIXEL_DEPTH_FACTOR),
        _item("navigator_device_memory", device["device_memory"]),
        _item("navigator_languages", identity["languages"]),
        _item("window_inner_width", device["inner_width"]),
        _item("window_inner_height", device["inner_height"]),
        _item("window_outer_width", device["outer_width"]),
        _item("window_outer_height", device["outer_height"]),
        _item("browser_detection_firefox", False),
        _item("browser_detection_brave", False),
        _item("9f41a2c", False),
        _item("5c273b3", False),
        _item("ce4046e", False),
        _item("f58835f", F58835F),
        _item("browser_object_checks", BROWSER_OBJECT_CHECKS),
        _item("29s83ih9", HASH_29S83IH9 + "⁣"),
        _item("audio_codecs", '{"ogg":"probably","mp3":"probably","wav":"probably","m4a":"maybe","aac":"probably"}'),
        _item("audio_codecs_extended_hash", AUDIO_CODECS_EXT_HASH),
        _item("video_codecs", '{"ogg":"","h264":"probably","webm":"probably","mpeg4v":"","mpeg4a":"","theora":""}'),
        _item("video_codecs_extended_hash", VIDEO_CODECS_EXT_HASH),
        _item("media_query_dark_mode", False),
        _item("f9bf2db", '{"pc":"no-preference","ah":"hover","ap":"fine","p":"fine","h":"hover","u":"fast","prm":"no-preference","prt":"no-preference","s":"enabled","fc":"none"}'),
        _item("headless_browser_phantom", False),
        _item("headless_browser_selenium", False),
        _item("headless_browser_nightmare_js", False),
        _item("862f2c1", 4),
        _item("1l2l5234ar2", "%d⁣" % now_ms),
        _item("document__referrer", cfg.get("document_referrer") or DEFAULT_DOCUMENT_REFERRER),
        _item("window__ancestor_origins", []),
        _item("window__tree_index", []),
        _item("window__tree_structure", random.choice(TREE_STRUCTURES)),
        _item("window__location_href", cfg.get("window_location_href", "")),
        _item("client_config__sitedata_location_href", cfg.get("sitedata_location_href", "")),
        _item("client_config__language", cfg.get("language", "en-US").lower()),
        _item("client_config__surl", surl),
        _item("c8480e29a", _md5(surl) + "⁢"),
        _item("client_config__triggered_inline", False),
        _item("mobile_sdk__is_sdk", False),
        _item("z87b89t5", None),
        _item("audio_fingerprint", random.choice(AUDIO_FINGERPRINTS)),
        _item("navigator_battery_charging", True),
        _item("7541c2s", identity["hash_7541c2s"]),
        _item("1f220c9", identity["hash_1f220c9"]),
        _item("math_fingerprint", MATH_FINGERPRINT),
        _item("supported_math_functions", SUPPORTED_MATH_FUNCS),
        _item("3f76dd27", "landscape-primary"),
        _item("5dd48ca0", 5),
        _item("4b4b269e68", str(uuid.uuid4())),
        _item("6a62b2a558", cfg.get("enforcement_hash") or DEFAULT_ENFORCEMENT_HASH),
        _item("is_keyless", False),
        _item("client_config__wait_for_settings", False),
        _item("c2d2015", identity["hash_c2d2015"]),
        _item("43f2d94", []),
        _item("20c15922", True),
        _item("4f59ca8", None),
        _item("3ea7194", {"supported": True, "formats": ["HDR10", "HLG"], "isHDR": False}),
        _item("05d3d24", identity["hash_05d3d24"]),
        _item("speech_default_voice", identity["speech_default_voice"]),
        _item("speech_voices_hash", identity["speech_voices_hash"]),
        _item("83eb055", identity["hash_83eb055"]),
        _item("4ca87df3d1", "Ow=="),
        _item("867e25e5d4", "Ow=="),
        _item("d4a306884c", "Ow=="),
        _item("vsadsa", random.randint(9, 10)),
        _item("basfas", [0, js_heap_size_limit(device["device_memory"])]),
        _item("lfasdgs", build_id or str(uuid.uuid4())),
    ]
    return out


def _fe_flag_set(fe, key):
    prefix = key + ":"
    for entry in fe:
        if entry.startswith(prefix):
            return entry[len(prefix):] != "false"
    return False


def build_jsbd(cfg):
    import json as _json
    title = _json.dumps(cfg.get("title", ""))
    return '{"HL":2,"NCE":true,"DT":%s,"NWD":"false","DMTO":1,"DOTO":1}' % title


def generate_bda(cfg, build_id=""):
    identity = new_session()
    device = generate_device()
    enhanced = build_enhanced_fp(cfg, identity, device, build_id)
    fe, f_hash = build_fe(device, identity)

    now_sec = int(time.time())
    n_val = base64.b64encode(str(now_sec).encode()).decode()
    wh = uuid.uuid4().hex + "|" + identity["machine_hash"]

    items = [
        _item("api_type", "js"),
        _item("f", f_hash),
        _item("n", n_val),
        _item("wh", wh),
        _item("enhanced_fp", enhanced),
    ]
    if _fe_flag_set(fe, "FOS") or _fe_flag_set(fe, "FB") or _fe_flag_set(fe, "FR"):
        items.append(_item("fb", 1))
    items += [
        _item("fe", fe),
        _item("ife_hash", compute_ife_hash(fe)),
        _item("jsbd", build_jsbd(cfg)),
        _item("c", "Copyright (c) 2026 Arkose Labs. All Rights Reserved."),
    ]
    return items
