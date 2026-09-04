'use strict';

const { khash, computeF, computeIfeHash, feValues } = require('./hashes');
const {
  WEBGL_EXTENSIONS, WEBGL_EXTENSIONS_HASH,
  buildWebglFields, webglHashInput, computeWebglHash, computeRttType,
} = require('./webgl');
const {
  md5, computeWebglExtensionsHash, computeBrowserObjectChecks,
  computeF58835f, computeSpeech, pickVoiceSet,
  audioCodecsExtendedHash, videoCodecsExtendedHash,
} = require('./derived');
const { generateDevice, poolSizes, jsHeapSizeLimit } = require('./devices');
const { generateBda } = require('./bda');
const { encrypt, marshalBda } = require('./crypto');
const { extractRsaKey } = require('./vmkey');
const { Solver, SolveResult, solve } = require('./solver');

module.exports = {
  Solver, SolveResult, solve,
  extractRsaKey,
  khash, computeF, computeIfeHash, feValues,
  WEBGL_EXTENSIONS, WEBGL_EXTENSIONS_HASH,
  buildWebglFields, webglHashInput, computeWebglHash, computeRttType,
  md5, computeWebglExtensionsHash, computeBrowserObjectChecks,
  computeF58835f, computeSpeech, pickVoiceSet,
  audioCodecsExtendedHash, videoCodecsExtendedHash,
  generateDevice, poolSizes, jsHeapSizeLimit,
  generateBda, encrypt, marshalBda,
};
