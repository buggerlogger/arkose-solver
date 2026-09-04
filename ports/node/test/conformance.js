'use strict';

// Verifies this port against ports/testvectors.json, which the Go
// implementation generates and asserts against captured ground truth.

const fs = require('fs');
const path = require('path');

const A = require('../src/index.js');

const vectorsPath = path.join(__dirname, '..', '..', 'testvectors.json');
const v = JSON.parse(fs.readFileSync(vectorsPath, 'utf8'));

let pass = 0;
let fail = 0;

function check(name, got, want) {
  if (got === want) {
    pass++;
  } else {
    fail++;
    console.log(`FAIL ${name}\n  got  ${got}\n  want ${want}`);
  }
}

for (const t of v.khash) {
  check(`khash(seed=${t.seed}, len=${t.input.length})`, A.khash(t.input, t.seed), t.want);
}

check('computeF', A.computeF(v.fe), v.wantF);
check('computeIfeHash', A.computeIfeHash(v.fe), v.wantIfeHash);

for (const w of v.webgl) {
  const fields = A.buildWebglFields(w.vendor, w.renderer);
  check(`${w.name} hashInput`, A.webglHashInput(fields), w.hashInput);
  check(`${w.name} webgl_hash_webgl`, A.computeWebglHash(fields), w.wantHash);
  check(`${w.name} rtt_type`,
    A.computeRttType(v.webglExtensionsHash, w.wantHash), w.wantRttType);
}

const d = v.derived;
check('webgl_extensions', A.WEBGL_EXTENSIONS, d.webglExtensions);
check('webgl_extensions_hash', A.computeWebglExtensionsHash(d.webglExtensions),
  d.wantWebglExtensionsHash);
check('f58835f', A.computeF58835f(), d.wantF58835f);
check('browser_object_checks', A.computeBrowserObjectChecks(['chrome']),
  d.wantBrowserObjectChecks);
check('29s83ih9', A.md5('false'), d.want29s83ih9);

const [sv, sh] = A.computeSpeech(d.speechVoices);
check('speech_default_voice', sv, d.wantSpeechDefaultVoice);
check('speech_voices_hash', sh, d.wantSpeechVoicesHash);
check('audio_codecs_extended_hash', A.audioCodecsExtendedHash(),
  d.wantAudioCodecsExtendedHash);
check('video_codecs_extended_hash', A.videoCodecsExtendedHash(),
  d.wantVideoCodecsExtendedHash);

// A generated payload must be internally consistent, not just individually correct.
for (let i = 0; i < 20; i++) {
  const items = A.generateBda({
    surl: 'https://verify.example.com',
    windowLocationHref: 'https://www.example.com/sign-in',
    sitedataLocationHref: 'https://www.example.com/sign-in',
    language: 'en-US',
    title: 'Sign in',
  });
  const top = Object.fromEntries(items.map((x) => [x.key, x.value]));
  const ef = Object.fromEntries(top.enhanced_fp.map((x) => [x.key, x.value]));

  check(`bda[${i}] enhanced_fp length`, top.enhanced_fp.length, 91);
  check(`bda[${i}] f self-consistent`, top.f, A.computeF(top.fe));
  check(`bda[${i}] ife_hash self-consistent`, top.ife_hash, A.computeIfeHash(top.fe));

  const idx = top.enhanced_fp.findIndex((x) => x.key === 'webgl_hash_webgl');
  check(`bda[${i}] webgl_hash index`, idx, 18);
  check(`bda[${i}] webgl_hash self-consistent`, ef.webgl_hash_webgl,
    A.computeWebglHash(top.enhanced_fp.slice(0, idx).map((x) => [x.key, x.value])));
  check(`bda[${i}] rtt_type tracks webgl hash`, ef.network_info_rtt_type,
    A.computeRttType(ef.webgl_extensions_hash, ef.webgl_hash_webgl));

  const colorDepth = parseInt(top.fe.find((e) => e.startsWith('D:')).slice(2), 10);
  check(`bda[${i}] screen_pixel_depth`, ef.screen_pixel_depth, colorDepth * 3);
}

console.log(`${pass} passed, ${fail} failed`);
process.exit(fail ? 1 : 0);
