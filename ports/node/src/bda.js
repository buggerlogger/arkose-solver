'use strict';

// BDA assembly: the 91-field enhanced_fp payload plus fe/f/ife_hash.
// Mirrors the Go implementation field for field and order for order.

const crypto = require('crypto');

const {
  audioCodecsExtendedHash, computeBrowserObjectChecks, computeF58835f,
  computeSpeech, md5, pickVoiceSet, videoCodecsExtendedHash,
} = require('./derived');
const { generateDevice, jsHeapSizeLimit } = require('./devices');
const { computeF, computeIfeHash } = require('./hashes');
const {
  SCREEN_PIXEL_DEPTH_FACTOR, WEBGL_EXTENSIONS_HASH,
  buildWebglFields, computeRttType, computeWebglHash,
} = require('./webgl');

const MATH_FINGERPRINT = 'e00752d866abf7fdf93bd4111bcfeb7b';
const SUPPORTED_MATH_FUNCS = '3f7aaba900fde542f258166cd2b71ef5';
const DEFAULT_ENFORCEMENT_HASH = '2d1c8a89671586563cb793ae2399b954';
const DEFAULT_DOCUMENT_REFERRER = 'https://www.google.com/';

const AUDIO_FINGERPRINTS = [
  '124.04347776696522', '124.04347527516074', '124.04347527840094',
  '124.04347541281574', '124.0434806260746', '124.04347527516074',
  '124.0434752046262', '124.04344968475198', '35.7383295930922',
  '124.0434802826973',
];

const TZ_POOL = [-180, -180, -180, -120, -120, -60, -60, 0, 0, 60,
  300, 300, 360, 420, 420, 480, 480, -480, -540, -330, -240];

const LANG_CHOICES = [
  ['en-US', 'en-US,en', 55], ['en-GB', 'en-GB,en', 12], ['en', 'en', 8],
  ['es-ES', 'es-ES,es', 4], ['pt-BR', 'pt-BR,pt', 4], ['de-DE', 'de-DE,de', 4],
  ['fr-FR', 'fr-FR,fr', 4], ['it-IT', 'it-IT,it', 3], ['ru-RU', 'ru-RU,ru', 3],
  ['tr-TR', 'tr-TR,tr', 3],
];

const FONT_SETS = [
  'Arial,Arial Black,Arial Narrow,Calibri,Cambria,Cambria Math,Comic Sans MS,Consolas,Courier,Courier New,Georgia,Helvetica,Impact,Lucida Console,Lucida Sans Unicode,Microsoft Sans Serif,MS Gothic,MS PGothic,MS Sans Serif,MS Serif,Palatino Linotype,Segoe Print,Segoe Script,Segoe UI,Segoe UI Light,Segoe UI Semibold,Segoe UI Symbol,Tahoma,Times,Times New Roman,Trebuchet MS,Verdana,Wingdings',
  'Arial,Arial Black,Arial Narrow,Calibri,Comic Sans MS,Consolas,Courier,Courier New,Georgia,Impact,Lucida Console,Lucida Sans Unicode,Microsoft Sans Serif,MS Gothic,MS PGothic,MS Sans Serif,MS Serif,Palatino Linotype,Segoe UI,Segoe UI Light,Segoe UI Symbol,Tahoma,Times,Times New Roman,Trebuchet MS,Verdana',
  'Arial,Arial Black,Arial Narrow,Calibri,Cambria,Cambria Math,Comic Sans MS,Consolas,Courier,Courier New,Georgia,Impact,Lucida Console,Lucida Sans Unicode,Marlett,Microsoft Sans Serif,MS Gothic,MS PGothic,MS Sans Serif,MS Serif,Palatino Linotype,Segoe Print,Segoe Script,Segoe UI,Segoe UI Light,Segoe UI Semibold,Segoe UI Symbol,Sylfaen,Tahoma,Times,Times New Roman,Trebuchet MS,Verdana,Webdings,Wingdings',
];

const TREE_STRUCTURES = ['[[[],[]],[[]],[],[]]', '[[],[],[],[[]],[]]', '[[],[],[],[[]],[[]],[],[]]'];
const GREASE_BRANDS = { 152: 'Not?A_Brand', 151: 'Not_A Brand', 150: 'Not)A;Brand', 149: 'Not-A.Brand' };
const CHROME_MAJORS = [148, 149, 150, 151, 152];

const choice = (a) => a[Math.floor(Math.random() * a.length)];
const hashField = (salt, field) => md5(`${salt}|${field}`);

function weightedLang() {
  const total = LANG_CHOICES.reduce((s, l) => s + l[2], 0);
  let r = Math.floor(Math.random() * total);
  for (const [tag, full, w] of LANG_CHOICES) {
    r -= w;
    if (r < 0) return [tag, full];
  }
  return [LANG_CHOICES[0][0], LANG_CHOICES[0][1]];
}

function newSession() {
  const salt = crypto.randomBytes(16).toString('hex');
  const [tag, full] = weightedLang();
  const cfp = Math.floor(Math.random() * 4294967296) - 2147483648;
  const [speechVoice, speechHash] = computeSpeech(pickVoiceSet(tag));
  return {
    salt,
    machineHash: hashField(salt, 'machine_hash'),
    hash7541c2s: hashField(salt, '7541c2s'),
    hash1f220c9: hashField(salt, '1f220c9'),
    hashC2d2015: hashField(salt, 'c2d2015'),
    hash05d3d24: hashField(salt, '05d3d24'),
    hash83eb055: hashField(salt, '83eb055'),
    speechDefaultVoice: speechVoice,
    speechVoicesHash: speechHash,
    tzOffset: choice(TZ_POOL),
    languageTag: tag,
    languages: full,
    fontList: choice(FONT_SETS),
    cfp,
  };
}

const formatDpr = (v) => (Number.isInteger(v) ? String(v) : String(v));

function buildFe(device, identity) {
  const fe = [
    'DNT:unknown',
    `L:${identity.languageTag}`,
    `D:${device.colorDepth}`,
    `PR:${formatDpr(device.devicePixelRatio)}`,
    `S:${device.screenWidth},${device.screenHeight}`,
    `AS:${device.availWidth},${device.availHeight}`,
    `TO:${identity.tzOffset}`,
    'SS:true', 'LS:true', 'IDB:true', 'B:false', 'ODB:false',
    'CPUC:unknown', 'PK:Win32',
    `CFP:${identity.cfp}`,
    'FR:false', 'FOS:false', 'FB:false',
    `JSF:${identity.fontList}`,
    'P:Chrome PDF Viewer,Chromium PDF Viewer,Microsoft Edge PDF Viewer,PDF Viewer,WebKit built-in PDF',
    'T:0,false,false',
    `H:${device.hardwareConcurrency}`,
    'SWF:false',
  ];
  return [fe, computeF(fe)];
}

function shuffledBrands(major) {
  const brands = ['Chromium', 'Google Chrome', GREASE_BRANDS[major] || 'Not/A)Brand'];
  for (let i = brands.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1));
    [brands[i], brands[j]] = [brands[j], brands[i]];
  }
  return brands.join(',');
}

const item = (key, value) => ({ key, value });

function buildEnhancedFp(cfg, identity, device, buildId) {
  const major = choice(CHROME_MAJORS);
  const webglFields = buildWebglFields(device.glVendor, device.glRenderer);
  const webglHash = computeWebglHash(webglFields);
  const rttType = computeRttType(WEBGL_EXTENSIONS_HASH, webglHash);
  const nowMs = Date.now();
  const surl = cfg.surl || '';
  const href = cfg.windowLocationHref || '';

  const out = webglFields.map(([k, v]) => item(k, v));
  out.push(
    item('webgl_hash_webgl', webglHash),
    item('user_agent_data_brands', shuffledBrands(major)),
    item('user_agent_data_mobile', false),
    item('navigator_connection_downlink', device.downlink),
    item('navigator_connection_downlink_max', null),
    item('network_info_rtt', device.rtt),
    item('network_info_save_data', device.saveData),
    item('network_info_rtt_type', rttType),
    item('screen_pixel_depth', device.colorDepth * SCREEN_PIXEL_DEPTH_FACTOR),
    item('navigator_device_memory', device.deviceMemory),
    item('navigator_languages', identity.languages),
    item('window_inner_width', device.innerWidth),
    item('window_inner_height', device.innerHeight),
    item('window_outer_width', device.outerWidth),
    item('window_outer_height', device.outerHeight),
    item('browser_detection_firefox', false),
    item('browser_detection_brave', false),
    item('9f41a2c', false),
    item('5c273b3', false),
    item('ce4046e', false),
    item('f58835f', computeF58835f()),
    item('browser_object_checks', computeBrowserObjectChecks(['chrome'])),
    item('29s83ih9', `${md5('false')}⁣`),
    item('audio_codecs', '{"ogg":"probably","mp3":"probably","wav":"probably","m4a":"maybe","aac":"probably"}'),
    item('audio_codecs_extended_hash', audioCodecsExtendedHash()),
    item('video_codecs', '{"ogg":"","h264":"probably","webm":"probably","mpeg4v":"","mpeg4a":"","theora":""}'),
    item('video_codecs_extended_hash', videoCodecsExtendedHash()),
    item('media_query_dark_mode', false),
    item('f9bf2db', '{"pc":"no-preference","ah":"hover","ap":"fine","p":"fine","h":"hover","u":"fast","prm":"no-preference","prt":"no-preference","s":"enabled","fc":"none"}'),
    item('headless_browser_phantom', false),
    item('headless_browser_selenium', false),
    item('headless_browser_nightmare_js', false),
    item('862f2c1', 4),
    item('1l2l5234ar2', `${nowMs}⁣`),
    item('document__referrer', cfg.documentReferrer || DEFAULT_DOCUMENT_REFERRER),
    item('window__ancestor_origins', []),
    item('window__tree_index', []),
    item('window__tree_structure', choice(TREE_STRUCTURES)),
    item('window__location_href', href),
    item('client_config__sitedata_location_href', cfg.sitedataLocationHref || ''),
    item('client_config__language', (cfg.language || 'en-US').toLowerCase()),
    item('client_config__surl', surl),
    item('c8480e29a', `${md5(surl)}⁢`),
    item('client_config__triggered_inline', false),
    item('mobile_sdk__is_sdk', false),
    item('z87b89t5', null),
    item('audio_fingerprint', choice(AUDIO_FINGERPRINTS)),
    item('navigator_battery_charging', true),
    item('7541c2s', identity.hash7541c2s),
    item('1f220c9', identity.hash1f220c9),
    item('math_fingerprint', MATH_FINGERPRINT),
    item('supported_math_functions', SUPPORTED_MATH_FUNCS),
    item('3f76dd27', 'landscape-primary'),
    item('5dd48ca0', 5),
    item('4b4b269e68', crypto.randomUUID()),
    item('6a62b2a558', cfg.enforcementHash || DEFAULT_ENFORCEMENT_HASH),
    item('is_keyless', false),
    item('client_config__wait_for_settings', false),
    item('c2d2015', identity.hashC2d2015),
    item('43f2d94', []),
    item('20c15922', true),
    item('4f59ca8', null),
    item('3ea7194', { supported: true, formats: ['HDR10', 'HLG'], isHDR: false }),
    item('05d3d24', identity.hash05d3d24),
    item('speech_default_voice', identity.speechDefaultVoice),
    item('speech_voices_hash', identity.speechVoicesHash),
    item('83eb055', identity.hash83eb055),
    item('4ca87df3d1', 'Ow=='),
    item('867e25e5d4', 'Ow=='),
    item('d4a306884c', 'Ow=='),
    item('vsadsa', 9 + Math.floor(Math.random() * 2)),
    item('basfas', [0, jsHeapSizeLimit(device.deviceMemory)]),
    item('lfasdgs', buildId || crypto.randomUUID()),
  );
  return out;
}

function feFlagSet(fe, key) {
  const prefix = `${key}:`;
  for (const entry of fe) {
    if (entry.startsWith(prefix)) return entry.slice(prefix.length) !== 'false';
  }
  return false;
}

function buildJsbd(cfg) {
  return `{"HL":2,"NCE":true,"DT":${JSON.stringify(cfg.title || '')},"NWD":"false","DMTO":1,"DOTO":1}`;
}

function generateBda(cfg, buildId = '') {
  const identity = newSession();
  const device = generateDevice();
  const enhanced = buildEnhancedFp(cfg, identity, device, buildId);
  const [fe, fHash] = buildFe(device, identity);

  const nowSec = Math.floor(Date.now() / 1000);
  const nVal = Buffer.from(String(nowSec), 'utf8').toString('base64');
  const wh = crypto.randomUUID().replace(/-/g, '') + '|' + identity.machineHash;

  const items = [
    item('api_type', 'js'),
    item('f', fHash),
    item('n', nVal),
    item('wh', wh),
    item('enhanced_fp', enhanced),
  ];
  if (feFlagSet(fe, 'FOS') || feFlagSet(fe, 'FB') || feFlagSet(fe, 'FR')) {
    items.push(item('fb', 1));
  }
  items.push(
    item('fe', fe),
    item('ife_hash', computeIfeHash(fe)),
    item('jsbd', buildJsbd(cfg)),
    item('c', 'Copyright (c) 2026 Arkose Labs. All Rights Reserved.'),
  );
  return items;
}

module.exports = { generateBda, newSession, buildFe, buildEnhancedFp };
