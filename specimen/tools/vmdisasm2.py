"""Definitive static disassembler for the Arkose api.js stack VM (capi 4.4.5).

Unlike vmdisasm.py (which consumed a runtime snapshot), this reads the program
straight out of the bundle. The program is the literal array `ye` in api.js;
it is byte-identical to the runtime `Te`, so nothing has to run.

Pipeline:
    node tools/deob3.js api_fresh.js api_deob.js      # resolve string tables
    node tools/clean.js api_deob.js  api_clean.js     # fold + de-bracket
    python tools/extract_program.py api_clean.js vm/program.json
    python tools/vmdisasm2.py vm/program.json vm/disasm.txt vm/opcodes.json vm/pseudo.txt

The opcode table below is COMPLETE: all 39 handlers from the VM's `Ke` map,
read out of the deobfuscated source. 27 are exercised by this program; the
other 12 are present-but-unused (defensive / other challenge programs).

VM model (interpreter `Ye`):

    while (Re < Pe.length) {
      try { const ins = Pe[Re]; ins.isAsync ? await ins.fn(ins.ops) : ins.fn(ins.ops); Re++; }
      catch (e) { if (Le.length) { Re = je[Le.pop()+"_CATCH_START"]; Me.error = e; } else throw e; }
    }

  Ce operand stack   ke frame stack {previousPosition,func,vars}   Re pc
  Ie function table {name:{start,end}}   De vars   je label->pc
  Le try stack   Me {name,error}   xe staged `this`-chain   Te raw text

Operand strings are repeating-key XOR (`me(s,key)`), key per opcode:
  LOAD_NAME 922 · PUSH_NAME 829 · SET_PROP 757 · typed-literal(be) 5
"""
import json
import re
import sys

KEY = {"name": "922", "pushname": "829", "prop": "757", "lit": "5"}


def xor(s, k):
    return ''.join(chr(ord(c) ^ ord(k[i % len(k)])) for i, c in enumerate(s))


def is_ascii(s):
    return s != '' and all(32 <= ord(c) < 127 for c in s)


def dec(s, k, strip=False):
    if strip and len(s) >= 2 and s[0] == "'" and s[-1] == "'":
        s = s[1:-1]
    out = xor(s, k)
    return out if is_ascii(out) else s


def safe(s):
    return ''.join(c if 32 <= ord(c) < 127 else '\\x%02x' % (ord(c) & 0xFF) for c in str(s))


# --- operand renderers ----------------------------------------------------
def r_name(o):   return dec(o[0], KEY["name"])                      # LOAD_NAME
def r_pushname(o):
    v = dec(o[0], KEY["pushname"], strip=True)
    deref = (o[1] == "true") if len(o) > 1 else False
    return ('deref ' if deref else '') + json.dumps(v)
def r_prop(o):   return json.dumps(dec(o[0], KEY["prop"]))          # SET_PROP
def r_lit(o):                                                       # typed literal via be()
    if not o:
        return ''
    v, ty = o[0], (o[1] if len(o) > 1 else '')
    if ty == 'string':
        return json.dumps(dec(v, KEY["lit"], strip=True))
    if ty == 'boolean':
        return 'true' if str(v) != 'false' else 'false'
    if ty == 'object' and str(v) == 'null':
        return 'null'
    return '%s:%s' % (v, ty)
def r_join(o):   return ' '.join(map(str, o))
def r_first(o):  return str(o[0]) if o else ''


# tag -> (MNEMONIC, renderer, arity_note, category, confidence)
# category: FN control / stack / call / branch / var / op / literal / try
OPCODES = {
    "a82bc": ("DEFUN",       r_first,  "name",         "fn",      "confirmed"),
    "b17bd": ("ENDFUN",      r_first,  "name",         "fn",      "confirmed"),
    "a0cd9": ("RETURN",      r_join,   "-",            "fn",      "confirmed"),
    "a5772": ("CALL_NAME",   r_join,   "name argc",    "call",    "confirmed"),
    "e73a3": ("CALL",        r_first,  "argc (async)", "call",    "confirmed"),
    "ac4e4": ("AWAIT",       r_join,   "-",            "call",    "confirmed"),
    "b404d": ("NEW",         r_join,   "argc ctor",    "call",    "confirmed"),
    "bb47b": ("LOAD_NAME",   r_name,   "name",         "var",     "confirmed"),
    "b1539": ("PUSH_NAME",   r_pushname, "str [deref]", "literal", "confirmed"),
    "c1888": ("PUSH_LIT",    r_lit,    "val type",     "literal", "confirmed"),
    "c2b6b": ("PUSH_LIT2",   r_lit,    "val type",     "literal", "confirmed"),
    "e186f": ("PUSH_RAW",    r_first,  "val",          "literal", "confirmed"),
    "a2b49": ("NEW_OBJECT",  r_join,   "-",            "stack",   "confirmed"),
    "e8265": ("NEW_ARRAY",   r_first,  "n",            "stack",   "confirmed"),
    "b89af": ("GET_MEMBER",  r_first,  "depth",        "stack",   "confirmed"),
    "b1724": ("INDEX_GET",   r_join,   "-",            "stack",   "confirmed"),
    "e776b": ("SET_PROP",    r_prop,   "key",          "stack",   "confirmed"),
    "e262e": ("SET_PATH",    r_join,   "-",            "stack",   "confirmed"),
    "e9293": ("JMP_LOGICAL",  r_join,  "op label",     "branch",  "confirmed"),
    "ddde1": ("POP",         r_join,   "-",            "stack",   "confirmed"),
    "b34bd": ("PUSH_HOLE",   r_join,   "-",            "stack",   "confirmed"),
    "ee884": ("ARR_APPEND",  r_join,   "-",            "stack",   "confirmed"),
    "f70a4": ("ARR_SPREAD",  r_join,   "-",            "stack",   "confirmed"),
    "e547b": ("OBJ_ASSIGN",  r_join,   "-",            "stack",   "confirmed"),
    "cea87": ("STORE",       r_first,  "name",         "var",     "confirmed"),
    "e5b15": ("DECL",        r_first,  "name",         "var",     "confirmed"),
    "aac6f": ("DELETE",      r_first,  "name",         "var",     "confirmed"),
    "ab1ed": ("SET_ERRNAME", r_first,  "name",         "var",     "confirmed"),
    "c923e": ("JMP",         r_first,  "label",        "branch",  "confirmed"),
    "d0c9a": ("JMP_FALSE",   r_first,  "label",        "branch",  "confirmed"),
    "b0280": ("LABEL",       r_first,  "label",        "branch",  "confirmed"),
    "ae498": ("BINOP",       r_first,  "op",           "op",      "confirmed"),
    "b0d77": ("LOGIC",       r_first,  "op",           "op",      "confirmed"),
    "b5558": ("UNARY",       r_first,  "op",           "op",      "confirmed"),
    "c9353": ("UPDATE",      r_join,   "op name deref", "op",     "confirmed"),
    "d1f65": ("TRY_BEGIN",   r_first,  "label",        "try",     "confirmed"),
    "b5da0": ("TRY_END",     r_first,  "label",        "try",     "confirmed"),
    "fe342": ("THROW",       r_join,   "-",            "try",     "confirmed"),
    "bec34": ("UNSUPPORTED", r_join,   "op",           "misc",    "confirmed"),
}


def parse(line):
    toks = re.findall(r"'[^']*'|\S+", line)
    return toks[0], toks[1:]


def main():
    prog = json.load(open(sys.argv[1], encoding='utf-8'))
    out_disasm = sys.argv[2] if len(sys.argv) > 2 else 'disasm.txt'
    out_ops = sys.argv[3] if len(sys.argv) > 3 else 'opcodes.json'
    out_pseudo = sys.argv[4] if len(sys.argv) > 4 else 'pseudo.txt'

    # label map: LABEL instructions record their pc
    labels = {}
    for pc, line in enumerate(prog):
        tag, ops = parse(line)
        if tag == "b0280" and ops:
            labels.setdefault(pc, []).append(ops[0])

    counts, unknown = {}, 0
    dis = ["; Arkose api.js stack VM  -  static disassembly",
           "; %d instructions, %d distinct opcodes used" % (len(prog), len({parse(l)[0] for l in prog})),
           "; operands decrypted (XOR keys name=922 pushname=829 prop=757 lit=5)",
           ""]
    for pc, line in enumerate(prog):
        tag, ops = parse(line)
        for lb in labels.get(pc, []):
            dis.append("%s:" % lb)
        spec = OPCODES.get(tag)
        if spec:
            mnem, render, _note, _cat, _conf = spec
            try:
                text = render(ops)
            except Exception:
                text = r_join(ops)
        else:
            mnem, text, unknown = "OP_" + tag, r_join(ops), unknown + 1
        counts[tag] = counts.get(tag, 0) + 1
        dis.append("%4d  %-11s %-42s ; %s" % (pc, mnem, safe(text), safe(line)))

    open(out_disasm, 'w', encoding='utf-8').write('\n'.join(dis) + '\n')

    # opcode table
    table = []
    for tag in sorted(counts, key=lambda t: -counts[t]):
        spec = OPCODES.get(tag)
        table.append({
            "tag": tag,
            "mnemonic": spec[0] if spec else "OP_" + tag,
            "operandFormat": spec[2] if spec else "?",
            "category": spec[3] if spec else "unknown",
            "count": counts[tag],
            "confidence": spec[4] if spec else "guess",
        })
    # include defined-but-unused opcodes for completeness
    for tag, spec in OPCODES.items():
        if tag not in counts:
            table.append({"tag": tag, "mnemonic": spec[0], "operandFormat": spec[2],
                          "category": spec[3], "count": 0, "confidence": spec[4]})
    json.dump(table, open(out_ops, 'w', encoding='utf-8'), indent=1)

    # coverage report
    used = len(counts)
    conf = sum(1 for t in counts if OPCODES.get(t))
    print("instructions %d | opcodes used %d | all mapped: %s | unknown-tag instrs: %d"
          % (len(prog), used, unknown == 0 and conf == used, unknown))
    print("opcode table: %d total (%d used, %d defined-unused)"
          % (len(table), used, len(OPCODES) - used))
    print("wrote %s, %s" % (out_disasm, out_ops))

    build_pseudo(prog, labels, out_pseudo)
    print("wrote %s" % out_pseudo)


# --- lightweight structured / pseudo-JS view ------------------------------
def build_pseudo(prog, labels, path):
    """Group the linear program into VM functions (DEFUN..ENDFUN) and render a
    readable, indented, operand-decrypted listing per function."""
    out = ["// Arkose VM program - function-structured view",
           "// Each DEFUN <id> ... ENDFUN <id> is one VM subroutine; the top level",
           "// is the module body. Labels are jump targets; CALL/AWAIT are async.",
           ""]
    # find DEFUN spans by name
    fn_start = {}
    for pc, line in enumerate(prog):
        tag, ops = parse(line)
        if tag == "a82bc" and ops:
            fn_start[ops[0]] = pc
    depth = 0
    for pc, line in enumerate(prog):
        tag, ops = parse(line)
        for lb in labels.get(pc, []):
            out.append("  " * depth + "%s:" % lb)
        spec = OPCODES.get(tag)
        mnem = spec[0] if spec else "OP_" + tag
        text = (spec[1](ops) if spec else r_join(ops))
        if tag == "a82bc":
            out.append("  " * depth + "function %s {   // pc %d" % (ops[0], pc))
            depth += 1
        elif tag == "b17bd":
            depth = max(0, depth - 1) if False else depth  # ENDFUN marks return target, keep flat
            out.append("  " * depth + "endfun %s" % (ops[0] if ops else ''))
        else:
            out.append("  " * depth + "%-11s %s" % (mnem, safe(text)))
    open(path, 'w', encoding='utf-8').write('\n'.join(out) + '\n')


if __name__ == '__main__':
    main()
