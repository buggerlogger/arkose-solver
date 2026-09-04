'use strict';

// Per-solve device generation from the shared weighted population.
// devices.json is the same file the Go implementation embeds.

const fs = require('fs');
const path = require('path');

const CHROME_COLOR_DEPTH = 24;
const INNER_WIDTH_INSET = 16;
const INNER_HEIGHT_INSET = 95;

let _pool = null;
function pool() {
  if (_pool === null) {
    _pool = JSON.parse(fs.readFileSync(path.join(__dirname, 'devices.json'), 'utf8'));
    for (const name of ['gpus', 'screens', 'cores', 'memory', 'network']) {
      if (!_pool[name] || !_pool[name].length) {
        throw new Error(`devices.json is missing pool ${name}`);
      }
    }
  }
  return _pool;
}

function pick(entries) {
  let total = 0;
  for (const e of entries) if (e.weight > 0) total += e.weight;
  if (total <= 0) return entries[0];
  let r = Math.floor(Math.random() * total);
  for (const e of entries) {
    if (!(e.weight > 0)) continue;
    r -= e.weight;
    if (r < 0) return e;
  }
  return entries[0];
}

// enhanced_fp["basfas"] carries performance.memory.jsHeapSizeLimit.
function jsHeapSizeLimit(deviceMemoryGB) {
  if (deviceMemoryGB >= 8) return 4395630592;
  if (deviceMemoryGB >= 4) return 4294705152;
  return 2197815296;
}

function windowOn(screen) {
  const availW = screen.availWidth || screen.width;
  const availH = screen.availHeight || screen.height;
  if (Math.floor(Math.random() * 100) < 35) return [availW, availH];
  const pct = () => 60 + Math.floor(Math.random() * 41);
  let w = Math.floor((availW * pct()) / 100);
  let h = Math.floor((availH * pct()) / 100);
  w = Math.max(w, INNER_WIDTH_INSET + 200);
  h = Math.max(h, INNER_HEIGHT_INSET + 200);
  return [w, h];
}

function generateDevice() {
  const p = pool();
  const gpu = pick(p.gpus);
  const screen = pick(p.screens);
  const net = pick(p.network);
  const [outerW, outerH] = windowOn(screen);

  return {
    gpuVendor: gpu.vendor,
    gpuModel: gpu.model,
    glVendor: gpu.glVendor,
    glRenderer: gpu.glRenderer,
    screenWidth: screen.width,
    screenHeight: screen.height,
    availWidth: screen.availWidth || screen.width,
    availHeight: screen.availHeight || screen.height,
    colorDepth: CHROME_COLOR_DEPTH,
    devicePixelRatio: screen.dpr === undefined ? 1 : screen.dpr,
    outerWidth: outerW,
    outerHeight: outerH,
    innerWidth: outerW - INNER_WIDTH_INSET,
    innerHeight: outerH - INNER_HEIGHT_INSET,
    deviceMemory: pick(p.memory).value,
    hardwareConcurrency: pick(p.cores).value,
    downlink: net.downlink,
    rtt: net.rtt,
    saveData: !!net.saveData,
  };
}

function poolSizes() {
  const p = pool();
  const out = {};
  for (const k of ['gpus', 'screens', 'cores', 'memory', 'network']) out[k] = p[k].length;
  return out;
}

module.exports = {
  CHROME_COLOR_DEPTH, INNER_WIDTH_INSET, INNER_HEIGHT_INSET,
  generateDevice, jsHeapSizeLimit, poolSizes,
};
