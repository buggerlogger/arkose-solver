// Settle every rotated string table without depending on outer module scope.
// For each (decoder DEC, table TBL, offset OFF) we synthesise:
//     var ARR = <literal array extracted statically>;
//     function TBL(){ return ARR; }
//     function DEC(t,e){ return ARR[t - OFF]; }
//     <the original rotation IIFE source, invoked with TBL>
// then read ARR back in its settled order.
// Usage: node settle2.js <in.js> <rawtables.json> <out.json>
const fs = require('fs');
const vm = require('vm');
const acorn = require('acorn');

const [, , inFile, rawFile, outFile] = process.argv;
const src = fs.readFileSync(inFile, 'utf8');
const raw = JSON.parse(fs.readFileSync(rawFile, 'utf8')); // {decoders:{DEC:{table,offset}}, tables:{TBL:[...]}}
const ast = acorn.parse(src, { ecmaVersion: 2022, ranges: true });

// collect rotation IIFEs keyed by the table identifier they are invoked with
const rotByTable = new Map();
(function findRot(node) {
  if (!node || typeof node.type !== 'string') return;
  if (node.type === 'CallExpression' && node.callee.type === 'FunctionExpression' &&
      node.arguments.length >= 1 && node.arguments[0].type === 'Identifier') {
    const text = src.slice(node.range[0], node.range[1]);
    if (/push\(\s*\w+\.shift\(\)\s*\)/.test(text)) {
      const tbl = node.arguments[0].name;
      if (!rotByTable.has(tbl)) rotByTable.set(tbl, text);
    }
  }
  for (const k of Object.keys(node)) {
    if (k === 'range') continue;
    const v = node[k];
    if (Array.isArray(v)) v.forEach(c => c && typeof c.type === 'string' && findRot(c));
    else if (v && typeof v.type === 'string') findRot(v);
  }
})(ast);

const out = { decoders: raw.decoders, tables: {} };
for (const [dec, info] of Object.entries(raw.decoders)) {
  const tbl = info.table, off = info.offset;
  const arr = raw.tables[tbl];
  if (!arr) { console.error('no raw array for', tbl); continue; }
  const rot = rotByTable.get(tbl);
  const code = [
    `var ARR = ${JSON.stringify(arr)};`,
    `function ${tbl}(){ return ARR; }`,
    `function ${dec}(t,e){ return ARR[t - ${off}]; }`,
    rot ? rot + ';' : '// no rotation',
    `__RESULT = ARR;`,
  ].join('\n');
  const ctx = { __RESULT: null, parseInt, String, Number, Math, isNaN };
  vm.createContext(ctx);
  try {
    vm.runInContext(code, ctx, { timeout: 20000 });
    out.tables[tbl] = ctx.__RESULT;
    console.error(`ok ${tbl} (${ctx.__RESULT.length}) via ${dec}${rot ? '' : ' [NO ROT]'}`);
  } catch (e) {
    console.error(`FAIL ${tbl} via ${dec}: ${e.message}`);
    out.tables[tbl] = arr;
  }
}
fs.writeFileSync(outFile, JSON.stringify(out));
console.error('wrote', outFile);
