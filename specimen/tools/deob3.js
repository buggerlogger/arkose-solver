// Scope-correct string-table deobfuscator.
//
// The bundle declares several DIFFERENT decoders that share a name (4x `S`,
// 4x `l`) and several different tables that share a name (4x `d`, 2x `w`).
// Resolving by name globally silently mixes them and produces convincing
// garbage, so everything here is keyed by declaration NODE, not by identifier.
//
// Usage: node deob3.js <in.js> <out.js> [report.json]
const fs = require('fs');
const vm = require('vm');
const acorn = require('acorn');

const [, , inFile, outFile, reportFile] = process.argv;
const src = fs.readFileSync(inFile, 'utf8');
const ast = acorn.parse(src, { ecmaVersion: 2022, ranges: true });

const FN = new Set(['FunctionDeclaration', 'FunctionExpression', 'ArrowFunctionExpression']);

// ---------- pass 1: parent links + scope tree ----------
const parent = new Map();
const scopeOf = new Map();      // node -> nearest enclosing scope object
const scopes = [];

function makeScope(node, up) {
  const s = { node, up, funcs: new Map(), children: [] };
  if (up) up.children.push(s);
  scopes.push(s);
  return s;
}

const rootScope = makeScope(ast, null);

(function link(node, par, scope) {
  if (!node || typeof node.type !== 'string') return;
  parent.set(node, par);
  let sc = scope;
  if (FN.has(node.type)) sc = makeScope(node, scope);
  scopeOf.set(node, sc);
  // hoist function declarations into the scope that contains them
  if (node.type === 'FunctionDeclaration' && node.id) scope.funcs.set(node.id.name, node);
  for (const k of Object.keys(node)) {
    if (k === 'range') continue;
    const v = node[k];
    if (Array.isArray(v)) v.forEach(c => c && typeof c.type === 'string' && link(c, node, sc));
    else if (v && typeof v.type === 'string') link(v, node, sc);
  }
})(ast, null, rootScope);

function resolveFn(name, scope) {
  for (let s = scope; s; s = s.up) if (s.funcs.has(name)) return s.funcs.get(name);
  return null;
}

// ---------- pass 2: identify decoder + table declarations ----------
// decoder shape:  function DEC(t,e){ var n = TBL(); return (DEC=function(t,e){return n[t-=OFF]}), DEC(t,e) }
const decoders = new Map();   // decl node -> {tableName, offset}
const tables = new Map();     // decl node -> string[] (raw, pre-rotation)

for (const s of scopes) {
  for (const [, node] of s.funcs) {
    const text = src.slice(node.range[0], node.range[1]);
    if (text.length < 60 || text.length > 400) continue;
    // Shape: function DEC(a,b){ var V = TBL(); return DEC=function(a,b){ return V[a -= OFF] }, DEC(a,b) }
    // Variable and parameter names differ between bundles, so bind them.
    const m = /var\s+([A-Za-z_$][\w$]*)\s*=\s*([A-Za-z_$][\w$]*)\(\)\s*;[\s\S]*?return\s+\1\[\(?\s*[A-Za-z_$][\w$]*\s*-=\s*(\d+)\s*\)?\]/.exec(text);
    if (m) decoders.set(node, { tableName: m[2], offset: parseInt(m[3], 10) });
  }
}

for (const s of scopes) {
  for (const [, node] of s.funcs) {
    if (node.params.length !== 0) continue;
    const body = node.body.body || [];
    const first = body[0];
    if (!first || first.type !== 'VariableDeclaration') continue;
    const d0 = first.declarations[0];
    if (!d0 || !d0.init || d0.init.type !== 'ArrayExpression') continue;
    const els = d0.init.elements;
    if (els.length < 10) continue;
    if (!els.every(e => e && e.type === 'Literal' && typeof e.value === 'string')) continue;
    tables.set(node, els.map(e => e.value));
  }
}

// ---------- pass 3: locate each table's rotation IIFE ----------
const rotFor = new Map(); // table decl node -> source text of the rotation IIFE
(function findRot(node) {
  if (!node || typeof node.type !== 'string') return;
  if (node.type === 'CallExpression' && node.callee.type === 'FunctionExpression' &&
      node.arguments.length >= 1 && node.arguments[0].type === 'Identifier') {
    const text = src.slice(node.range[0], node.range[1]);
    if (/push\(\s*\w+\.shift\(\)\s*\)/.test(text)) {
      const tblNode = resolveFn(node.arguments[0].name, scopeOf.get(node) || rootScope);
      if (tblNode && tables.has(tblNode) && !rotFor.has(tblNode)) rotFor.set(tblNode, text);
    }
  }
  for (const k of Object.keys(node)) {
    if (k === 'range') continue;
    const v = node[k];
    if (Array.isArray(v)) v.forEach(c => c && typeof c.type === 'string' && findRot(c));
    else if (v && typeof v.type === 'string') findRot(v);
  }
})(ast);

// ---------- pass 4: settle every table against its own rotation ----------
const settled = new Map(); // table decl node -> string[]
for (const [tblNode, arr] of tables) {
  const rot = rotFor.get(tblNode);
  if (!rot) { settled.set(tblNode, arr); continue; }
  // find a decoder that uses THIS table, to reproduce the checksum expression
  let dec = null, decInfo = null;
  for (const [dNode, info] of decoders) {
    if (resolveFn(info.tableName, scopeOf.get(dNode) || rootScope) === tblNode) { dec = dNode; decInfo = info; break; }
  }
  const tblName = tblNode.id.name;
  const decName = dec ? dec.id.name : '__dec';
  const off = decInfo ? decInfo.offset : 0;
  const code = [
    `var ARR = ${JSON.stringify(arr)};`,
    `function ${tblName}(){ return ARR; }`,
    `function ${decName}(t,e){ return ARR[t - ${off}]; }`,
    `;(${rot});`,
    `__RESULT = ARR;`,
  ].join('\n');
  const ctx = { __RESULT: null, parseInt, String, Number, Math, isNaN };
  vm.createContext(ctx);
  try {
    vm.runInContext(code, ctx, { timeout: 20000 });
    settled.set(tblNode, ctx.__RESULT);
  } catch (e) {
    settled.set(tblNode, arr);
    console.error(`  ! rotation failed for ${tblName}: ${e.message}`);
  }
}

// ---------- pass 5: scope-correct constant propagation + replacement ----------
const edits = [];
const stats = { hits: 0, oob: 0, nonconst: 0, notable: 0 };

function newEnv(up) { return { up, vars: new Map() }; }
function envGet(env, n) { for (let e = env; e; e = e.up) if (e.vars.has(n)) return e.vars.get(n); }

function classify(node, env, scope) {
  if (!node) return;
  if (node.type === 'Literal' && typeof node.value === 'number') return { kind: 'num', v: node.value };
  if (node.type === 'UnaryExpression' && node.operator === '-' &&
      node.argument.type === 'Literal' && typeof node.argument.value === 'number') {
    return { kind: 'num', v: -node.argument.value };
  }
  if (node.type === 'Identifier') {
    // a local binding always wins over a same-named function elsewhere
    const b = envGet(env, node.name);
    if (b) return b;
    const fn = resolveFn(node.name, scope);
    if (fn && decoders.has(fn)) return { kind: 'dec', v: fn };
  }
}

function resolveDec(decNode, num) {
  const info = decoders.get(decNode);
  const tblNode = resolveFn(info.tableName, scopeOf.get(decNode) || rootScope);
  if (!tblNode || !settled.has(tblNode)) { stats.notable++; return; }
  const t = settled.get(tblNode);
  const i = num - info.offset;
  if (i < 0 || i >= t.length) { stats.oob++; return; }
  return t[i];
}

function walk(node, env, scope) {
  if (!node || typeof node.type !== 'string') return;
  let e = env, sc = scope;
  if (FN.has(node.type)) { e = newEnv(env); sc = scopeOf.get(node.body) || scope; }

  if (node.type === 'VariableDeclarator' && node.id.type === 'Identifier') {
    walk(node.init, e, sc);
    const c = classify(node.init, e, sc);
    if (c) e.vars.set(node.id.name, c); else e.vars.delete(node.id.name);
    return;
  }
  if (node.type === 'AssignmentExpression' && node.operator === '=' && node.left.type === 'Identifier') {
    walk(node.right, e, sc);
    const c = classify(node.right, e, sc);
    if (c) e.vars.set(node.left.name, c); else e.vars.delete(node.left.name);
    return;
  }
  if (node.type === 'CallExpression' && node.callee.type === 'Identifier' && node.arguments.length >= 1) {
    const f = classify(node.callee, e, sc);
    if (f && f.kind === 'dec') {
      const a = classify(node.arguments[0], e, sc);
      if (a && a.kind === 'num') {
        const s = resolveDec(f.v, a.v);
        if (s !== undefined) {
          edits.push([node.range[0], node.range[1], JSON.stringify(s)]);
          stats.hits++;
          return;
        }
      } else stats.nonconst++;
    }
  }
  for (const k of Object.keys(node)) {
    if (k === 'range') continue;
    const v = node[k];
    if (Array.isArray(v)) v.forEach(c => c && typeof c.type === 'string' && walk(c, e, sc));
    else if (v && typeof v.type === 'string') walk(v, e, sc);
  }
}

walk(ast, newEnv(null), rootScope);

edits.sort((a, b) => a[0] - b[0]);
let out = '', pos = 0;
for (const [s, en, t] of edits) { if (s < pos) continue; out += src.slice(pos, s) + t; pos = en; }
out += src.slice(pos);
fs.writeFileSync(outFile, out);

const report = {
  decoders: decoders.size, tables: tables.size, rotations: rotFor.size,
  ...stats, edits: edits.length,
};
console.log(JSON.stringify(report));
if (reportFile) fs.writeFileSync(reportFile, JSON.stringify(report, null, 1));
