// AST-safe readability pass, run AFTER deob3.js.
//   1. fold "a" + "b" (recursively) into one literal
//   2. rewrite obj["prop"] -> obj.prop, but ONLY for real member expressions
//      with an identifier-safe, non-reserved name
//
// Doing (2) with a regex corrupts single-element array literals like ["foo"],
// which is why this is AST-driven.
//
// Usage: node clean.js <in.js> <out.js>
const fs = require('fs');
const acorn = require('acorn');

const [, , inFile, outFile] = process.argv;
let src = fs.readFileSync(inFile, 'utf8');

const RESERVED = new Set(('break case catch class const continue debugger default delete do else export ' +
  'extends finally for function if import in instanceof new return super switch this throw try typeof ' +
  'var void while with yield let static enum await implements package protected interface private public ' +
  'null true false').split(' '));

const IDENT = /^[A-Za-z_$][A-Za-z0-9_$]*$/;

function pass(source) {
  const ast = acorn.parse(source, { ecmaVersion: 2022, ranges: true });
  const edits = [];
  let folds = 0, dots = 0;

  // fully evaluate a chain of string-literal concatenations
  function strValue(node) {
    if (node.type === 'Literal' && typeof node.value === 'string') return node.value;
    if (node.type === 'BinaryExpression' && node.operator === '+') {
      const l = strValue(node.left), r = strValue(node.right);
      if (l !== undefined && r !== undefined) return l + r;
    }
  }

  (function walk(node) {
    if (!node || typeof node.type !== 'string') return;

    if (node.type === 'BinaryExpression' && node.operator === '+') {
      const v = strValue(node);
      if (v !== undefined) {
        edits.push([node.range[0], node.range[1], JSON.stringify(v)]);
        folds++;
        return; // do not descend; the whole expression is replaced
      }
    }

    // `5["toString"]` must NOT become `5.toString` (parse error), and the same
    // goes for any object whose source ends in a digit or a `.`-ambiguous form.
    const objIsNumeric = node.type === 'MemberExpression' && node.object.type === 'Literal' &&
      (typeof node.object.value === 'number' || typeof node.object.value === 'bigint');

    if (node.type === 'MemberExpression' && node.computed && !objIsNumeric &&
        node.property.type === 'Literal' && typeof node.property.value === 'string' &&
        IDENT.test(node.property.value) && !RESERVED.has(node.property.value)) {
      // Anchor on the opening bracket, NOT on object.range[1]: acorn's range
      // for a parenthesised object stops before its closing paren, so
      // ("a"+"b")["concat"] would lose the ")" and stop parsing.
      const open = source.lastIndexOf('[', node.property.range[0]);
      if (open > node.object.range[1] - 1) {
        edits.push([open, node.range[1], '.' + node.property.value]);
        dots++;
        walk(node.object);
        return;
      }
    }

    for (const k of Object.keys(node)) {
      if (k === 'range') continue;
      const v = node[k];
      if (Array.isArray(v)) v.forEach(c => c && typeof c.type === 'string' && walk(c));
      else if (v && typeof v.type === 'string') walk(v);
    }
  })(ast);

  edits.sort((a, b) => a[0] - b[0]);
  let out = '', pos = 0;
  for (const [s, e, t] of edits) { if (s < pos) continue; out += source.slice(pos, s) + t; pos = e; }
  out += source.slice(pos);
  return { out, folds, dots };
}

let totalFolds = 0, totalDots = 0;
for (let i = 0; i < 5; i++) {
  const r = pass(src);
  if (r.folds === 0 && r.dots === 0) break;
  // Each pass must leave the file parseable; if it does not, keep the last
  // good source and report where it broke rather than emitting garbage.
  try {
    acorn.parse(r.out, { ecmaVersion: 2022 });
  } catch (e) {
    const p = e.pos || 0;
    console.error(`pass ${i + 1} broke the file at ${e.loc && e.loc.line}:${e.loc && e.loc.column}`);
    console.error('context: ' + JSON.stringify(r.out.slice(Math.max(0, p - 200), p + 100)));
    fs.writeFileSync(outFile + '.broken', r.out);
    break;
  }
  src = r.out;
  totalFolds += r.folds;
  totalDots += r.dots;
}

fs.writeFileSync(outFile, src);
try {
  acorn.parse(src, { ecmaVersion: 2022 });
  console.log(JSON.stringify({ folds: totalFolds, dots: totalDots, bytes: src.length, parses: true }));
} catch (e) {
  const p = e.pos || 0;
  console.error('PARSE FAILED at', e.loc, '\ncontext:\n' + JSON.stringify(src.slice(Math.max(0, p - 160), p + 80)));
  process.exit(1);
}
