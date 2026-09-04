'use strict';

// webgl_hash_webgl is K(Object.entries(t).join(",")) where `t` already carries
// an empty webgl_hash_webgl, so the hashed string ends ",webgl_hash_webgl,".
//
// network_info_rtt_type is rewritten by the VM at serialization time as
// webgl_extensions_hash[:3] + webgl_hash_webgl[:3]; the familiar "730" prefix
// is just 7300c23f...[:3], not a constant.

const { khash } = require('./hashes');

const WEBGL_EXTENSIONS = [
  'ANGLE_instanced_arrays', 'EXT_blend_minmax', 'EXT_clip_control',
  'EXT_color_buffer_half_float', 'EXT_depth_clamp', 'EXT_disjoint_timer_query',
  'EXT_float_blend', 'EXT_frag_depth', 'EXT_polygon_offset_clamp',
  'EXT_shader_texture_lod', 'EXT_texture_compression_bptc',
  'EXT_texture_compression_rgtc', 'EXT_texture_filter_anisotropic',
  'EXT_texture_mirror_clamp_to_edge', 'EXT_sRGB', 'KHR_parallel_shader_compile',
  'OES_element_index_uint', 'OES_fbo_render_mipmap', 'OES_standard_derivatives',
  'OES_texture_float', 'OES_texture_float_linear', 'OES_texture_half_float',
  'OES_texture_half_float_linear', 'OES_vertex_array_object',
  'WEBGL_blend_func_extended', 'WEBGL_color_buffer_float',
  'WEBGL_compressed_texture_s3tc', 'WEBGL_compressed_texture_s3tc_srgb',
  'WEBGL_debug_renderer_info', 'WEBGL_debug_shaders', 'WEBGL_depth_texture',
  'WEBGL_draw_buffers', 'WEBGL_lose_context', 'WEBGL_multi_draw',
  'WEBGL_polygon_mode',
].join(';');

const WEBGL_EXTENSIONS_HASH = khash(WEBGL_EXTENSIONS, 0);
const WEBGL_VERSION = 'WebGL 1.0 (OpenGL ES 2.0 Chromium)';
const WEBGL_SHADING_LANGUAGE = 'WebGL GLSL ES 1.0 (OpenGL ES GLSL ES 1.0 Chromium)';
const RTT_TYPE_FALLBACK = 'abcdef';
const SCREEN_PIXEL_DEPTH_FACTOR = 3;

function buildWebglFields(vendor, renderer) {
  return [
    ['webgl_extensions', WEBGL_EXTENSIONS],
    ['webgl_extensions_hash', WEBGL_EXTENSIONS_HASH],
    ['webgl_renderer', 'WebKit WebGL'],
    ['webgl_vendor', 'WebKit'],
    ['webgl_version', WEBGL_VERSION],
    ['webgl_shading_language_version', WEBGL_SHADING_LANGUAGE],
    ['webgl_aliased_line_width_range', '[1, 1]'],
    ['webgl_aliased_point_size_range', '[1, 1024]'],
    ['webgl_antialiasing', 'yes'],
    ['webgl_bits', '8,8,24,8,8,0'],
    ['webgl_max_params', '16,32,16384,1024,16384,16,16384,30,16,16,4096'],
    ['webgl_max_viewport_dims', '[32767, 32767]'],
    ['webgl_unmasked_vendor', vendor],
    ['webgl_unmasked_renderer', renderer],
    ['webgl_vsf_params', '23,127,127,23,127,127,23,127,127'],
    ['webgl_vsi_params', '0,31,30,0,31,30,0,31,30'],
    ['webgl_fsf_params', '23,127,127,23,127,127,23,127,127'],
    ['webgl_fsi_params', '0,31,30,0,31,30,0,31,30'],
  ];
}

function webglHashInput(fields) {
  const parts = [];
  for (const [k, v] of fields) {
    parts.push(k);
    parts.push(typeof v === 'string' ? v : '');
  }
  return parts.join(',') + ',webgl_hash_webgl,';
}

const computeWebglHash = (fields) => khash(webglHashInput(fields), 0);

function computeRttType(extensionsHash, webglHash) {
  if (extensionsHash.length > 12 && webglHash.length > 12) {
    return extensionsHash.slice(0, 3) + webglHash.slice(0, 3);
  }
  return RTT_TYPE_FALLBACK;
}

module.exports = {
  WEBGL_EXTENSIONS, WEBGL_EXTENSIONS_HASH, WEBGL_VERSION,
  WEBGL_SHADING_LANGUAGE, SCREEN_PIXEL_DEPTH_FACTOR,
  buildWebglFields, webglHashInput, computeWebglHash, computeRttType,
};
