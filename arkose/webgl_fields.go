package arkose

func buildWebGLFields(gpu WebGLFingerprint, chrome chromeVersion) []Item {
	return []Item{
		{Key: "webgl_extensions", Value: "ANGLE_instanced_arrays;EXT_blend_minmax;EXT_clip_control;EXT_color_buffer_half_float;EXT_depth_clamp;EXT_disjoint_timer_query;EXT_float_blend;EXT_frag_depth;EXT_polygon_offset_clamp;EXT_shader_texture_lod;EXT_texture_compression_bptc;EXT_texture_compression_rgtc;EXT_texture_filter_anisotropic;EXT_texture_mirror_clamp_to_edge;EXT_sRGB;KHR_parallel_shader_compile;OES_element_index_uint;OES_fbo_render_mipmap;OES_standard_derivatives;OES_texture_float;OES_texture_float_linear;OES_texture_half_float;OES_texture_half_float_linear;OES_vertex_array_object;WEBGL_blend_func_extended;WEBGL_color_buffer_float;WEBGL_compressed_texture_s3tc;WEBGL_compressed_texture_s3tc_srgb;WEBGL_debug_renderer_info;WEBGL_debug_shaders;WEBGL_depth_texture;WEBGL_draw_buffers;WEBGL_lose_context;WEBGL_multi_draw;WEBGL_polygon_mode"},
		{Key: "webgl_extensions_hash", Value: chrome.WebGLExtensionsHash},
		{Key: "webgl_renderer", Value: "WebKit WebGL"},
		{Key: "webgl_vendor", Value: "WebKit"},
		{Key: "webgl_version", Value: chrome.WebGLVersion},
		{Key: "webgl_shading_language_version", Value: chrome.WebGLShadingLanguage},
		{Key: "webgl_aliased_line_width_range", Value: "[1, 1]"},
		{Key: "webgl_aliased_point_size_range", Value: "[1, 1024]"},
		{Key: "webgl_antialiasing", Value: "yes"},
		{Key: "webgl_bits", Value: "8,8,24,8,8,0"},
		{Key: "webgl_max_params", Value: "16,32,16384,1024,16384,16,16384,30,16,16,4096"},
		{Key: "webgl_max_viewport_dims", Value: "[32767, 32767]"},
		{Key: "webgl_unmasked_vendor", Value: gpu.UnmaskedVendor},
		{Key: "webgl_unmasked_renderer", Value: gpu.UnmaskedRenderer},
		{Key: "webgl_vsf_params", Value: "23,127,127,23,127,127,23,127,127"},
		{Key: "webgl_vsi_params", Value: "0,31,30,0,31,30,0,31,30"},
		{Key: "webgl_fsf_params", Value: "23,127,127,23,127,127,23,127,127"},
		{Key: "webgl_fsi_params", Value: "0,31,30,0,31,30,0,31,30"},
	}
}
