"""WebGL fields and the two values derived from them.

webgl_hash_webgl is K(Object.entries(t).join(",")) where `t` already contains
an empty webgl_hash_webgl, so the hashed string ends "...,webgl_hash_webgl,".

network_info_rtt_type is rewritten by the VM at serialization time as
webgl_extensions_hash[:3] + webgl_hash_webgl[:3] (see specimen/vm/decompiled.js);
the familiar "730" prefix is just 7300c23f...[:3], not a constant.
"""

from .hashes import khash

WEBGL_EXTENSIONS = (
    "ANGLE_instanced_arrays;EXT_blend_minmax;EXT_clip_control;"
    "EXT_color_buffer_half_float;EXT_depth_clamp;EXT_disjoint_timer_query;"
    "EXT_float_blend;EXT_frag_depth;EXT_polygon_offset_clamp;"
    "EXT_shader_texture_lod;EXT_texture_compression_bptc;"
    "EXT_texture_compression_rgtc;EXT_texture_filter_anisotropic;"
    "EXT_texture_mirror_clamp_to_edge;EXT_sRGB;KHR_parallel_shader_compile;"
    "OES_element_index_uint;OES_fbo_render_mipmap;OES_standard_derivatives;"
    "OES_texture_float;OES_texture_float_linear;OES_texture_half_float;"
    "OES_texture_half_float_linear;OES_vertex_array_object;"
    "WEBGL_blend_func_extended;WEBGL_color_buffer_float;"
    "WEBGL_compressed_texture_s3tc;WEBGL_compressed_texture_s3tc_srgb;"
    "WEBGL_debug_renderer_info;WEBGL_debug_shaders;WEBGL_depth_texture;"
    "WEBGL_draw_buffers;WEBGL_lose_context;WEBGL_multi_draw;WEBGL_polygon_mode"
)

WEBGL_EXTENSIONS_HASH = khash(WEBGL_EXTENSIONS, 0)
WEBGL_VERSION = "WebGL 1.0 (OpenGL ES 2.0 Chromium)"
WEBGL_SHADING_LANGUAGE = "WebGL GLSL ES 1.0 (OpenGL ES GLSL ES 1.0 Chromium)"

RTT_TYPE_FALLBACK = "abcdef"
SCREEN_PIXEL_DEPTH_FACTOR = 3


def build_webgl_fields(vendor, renderer):
    """The 18 webgl_* entries preceding webgl_hash_webgl, in hash order."""
    return [
        ("webgl_extensions", WEBGL_EXTENSIONS),
        ("webgl_extensions_hash", WEBGL_EXTENSIONS_HASH),
        ("webgl_renderer", "WebKit WebGL"),
        ("webgl_vendor", "WebKit"),
        ("webgl_version", WEBGL_VERSION),
        ("webgl_shading_language_version", WEBGL_SHADING_LANGUAGE),
        ("webgl_aliased_line_width_range", "[1, 1]"),
        ("webgl_aliased_point_size_range", "[1, 1024]"),
        ("webgl_antialiasing", "yes"),
        ("webgl_bits", "8,8,24,8,8,0"),
        ("webgl_max_params", "16,32,16384,1024,16384,16,16384,30,16,16,4096"),
        ("webgl_max_viewport_dims", "[32767, 32767]"),
        ("webgl_unmasked_vendor", vendor),
        ("webgl_unmasked_renderer", renderer),
        ("webgl_vsf_params", "23,127,127,23,127,127,23,127,127"),
        ("webgl_vsi_params", "0,31,30,0,31,30,0,31,30"),
        ("webgl_fsf_params", "23,127,127,23,127,127,23,127,127"),
        ("webgl_fsi_params", "0,31,30,0,31,30,0,31,30"),
    ]


def webgl_hash_input(fields):
    parts = []
    for key, value in fields:
        parts.append(key)
        parts.append(value if isinstance(value, str) else "")
    return ",".join(parts) + ",webgl_hash_webgl,"


def compute_webgl_hash(fields):
    return khash(webgl_hash_input(fields), 0)


def compute_rtt_type(extensions_hash, webgl_hash):
    if len(extensions_hash) > 12 and len(webgl_hash) > 12:
        return extensions_hash[:3] + webgl_hash[:3]
    return RTT_TYPE_FALLBACK
