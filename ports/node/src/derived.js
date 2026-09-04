'use strict';

// Fingerprint values the enforcement derives rather than reads. Each recipe was
// recovered from the deobfuscated bundle and reproduces what a real Chrome
// sends; conformance values live in ports/testvectors.json.

const crypto = require('crypto');
const fs = require('fs');
const path = require('path');

const { khash } = require('./hashes');

const md5 = (s) => crypto.createHash('md5').update(s, 'utf8').digest('hex');

const computeWebglExtensionsHash = (extensions) => khash(extensions, 0);

function computeBrowserObjectChecks(present) {
  if (!present || !present.length) return null;
  return md5([...present].sort().join(','));
}

const CHROME_WINDOWS_FEATURES = {
  permission_status: true,
  eye_dropper: true,
  audio_data: true,
  writable_stream: true,
  css_style_rule: true,
  navigator_ua: true,
  barcode_detector: false,
  display_names: true,
  contacts_manager: false,
  svg_discard_element: false,
  usb: true,
  media_device: true,
  playback_quality: true,
};

const F58835F_ORDER = [
  'permission_status', 'eye_dropper', 'audio_data', 'writable_stream',
  'css_style_rule', 'navigator_ua', 'barcode_detector', 'display_names',
  'contacts_manager', 'svg_discard_element', 'usb', 'media_device',
  'playback_quality',
];
const F58835F_DEFINED = new Set(['usb', 'media_device']);

function computeF58835f(features) {
  const f = features || CHROME_WINDOWS_FEATURES;
  const parts = F58835F_ORDER.map((name) => {
    const v = f[name];
    if (F58835F_DEFINED.has(name)) return `${name}: ${v ? 'defined' : 'NA'}`;
    return `${name}: ${v ? 'true' : 'false'}`;
  });
  return md5(JSON.stringify(parts));
}

function computeSpeech(voices) {
  if (!voices || !voices.length) return [null, null];
  let defaultVoice = null;
  const pairs = [];
  for (const v of voices) {
    if (v.default) defaultVoice = `${v.name} || ${v.lang}`;
    pairs.push(`${v.name},${v.lang}`);
  }
  return [defaultVoice, md5(pairs.join(','))];
}

const WINDOWS_VOICE_SETS = [
  [{ name: 'Microsoft David - English (United States)', lang: 'en-US', default: true },
   { name: 'Microsoft Zira - English (United States)', lang: 'en-US' }],
  [{ name: 'Microsoft David - English (United States)', lang: 'en-US', default: true },
   { name: 'Microsoft Mark - English (United States)', lang: 'en-US' },
   { name: 'Microsoft Zira - English (United States)', lang: 'en-US' }],
  [{ name: 'Microsoft Zira - English (United States)', lang: 'en-US', default: true },
   { name: 'Microsoft David - English (United States)', lang: 'en-US' }],
  [{ name: 'Microsoft David - English (United States)', lang: 'en-US', default: true },
   { name: 'Microsoft Zira - English (United States)', lang: 'en-US' },
   { name: 'Microsoft Hazel - English (United Kingdom)', lang: 'en-GB' },
   { name: 'Microsoft George - English (United Kingdom)', lang: 'en-GB' }],
  [{ name: 'Microsoft David - English (United States)', lang: 'en-US', default: true },
   { name: 'Microsoft Zira - English (United States)', lang: 'en-US' },
   { name: 'Microsoft Helena - Spanish (Spain)', lang: 'es-ES' },
   { name: 'Microsoft Laura - Spanish (Spain)', lang: 'es-ES' },
   { name: 'Microsoft Pablo - Spanish (Spain)', lang: 'es-ES' }],
];

const LOCALE_VOICES = {
  'en-GB': 'Microsoft George - English (United Kingdom)',
  'es-ES': 'Microsoft Helena - Spanish (Spain)',
  'pt-BR': 'Microsoft Daniel - Portuguese (Brazil)',
  'de-DE': 'Microsoft Hedda - German (Germany)',
  'fr-FR': 'Microsoft Hortense - French (France)',
  'it-IT': 'Microsoft Elsa - Italian (Italy)',
  'ru-RU': 'Microsoft Irina - Russian (Russia)',
  'tr-TR': 'Microsoft Tolga - Turkish (Turkiye)',
};

function pickVoiceSet(langTag) {
  const base = WINDOWS_VOICE_SETS[Math.floor(Math.random() * WINDOWS_VOICE_SETS.length)];
  if (langTag === 'en-US' || langTag === 'en') return base;
  const name = LOCALE_VOICES[langTag];
  if (!name) return base;
  return [{ name, lang: langTag, default: true },
    ...base.map((v) => ({ name: v.name, lang: v.lang }))];
}

let _codecs = null;
function codecs() {
  if (_codecs === null) {
    _codecs = JSON.parse(fs.readFileSync(path.join(__dirname, 'codecs.json'), 'utf8'));
  }
  return _codecs;
}

// Rebuild the exact JSON.stringify output the browser hashes.
function codecProbeJson(rows) {
  const parts = rows.map((r) =>
    JSON.stringify(r.mime) + ':{"canPlay":' + JSON.stringify(r.canPlay)
    + ',"mediaSource":' + (r.mediaSource ? 'true' : 'false') + '}');
  return '{' + parts.join(',') + '}';
}

const audioCodecsExtendedHash = () => md5(codecProbeJson(codecs().audio));
const videoCodecsExtendedHash = () => md5(codecProbeJson(codecs().video));

module.exports = {
  md5, computeWebglExtensionsHash, computeBrowserObjectChecks,
  CHROME_WINDOWS_FEATURES, computeF58835f, computeSpeech,
  WINDOWS_VOICE_SETS, LOCALE_VOICES, pickVoiceSet,
  audioCodecsExtendedHash, videoCodecsExtendedHash,
};
