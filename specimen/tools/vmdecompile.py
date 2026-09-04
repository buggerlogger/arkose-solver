"""Stack-to-expression decompiler for the Arkose api.js VM program.

vmdisasm2.py gives a faithful linear listing. This goes one step further: it
symbolically executes each VM subroutine's stack to lift the ops back into
readable JavaScript expressions/statements. It is a best-effort reconstruction
(the VM's control flow is preserved as labels/gotos), verified by eye against
the deobfuscated handlers and the runtime crypto trace.

Usage: python vmdecompile.py vm/program.json vm/decompiled.js
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


def parse(line):
    t = re.findall(r"'[^']*'|\S+", line)
    return t[0], t[1:]


def jsname(x):
    """Render a VM identifier: 5-hex ids become v_<id>, real names stay."""
    return "v_" + x if re.match(r'^[0-9a-f]{5}$', x) else x


def lit(o):
    v, ty = o[0], (o[1] if len(o) > 1 else '')
    if ty == 'string':
        return json.dumps(dec(v, KEY["lit"], strip=True))
    if ty == 'boolean':
        return 'true' if str(v) != 'false' else 'false'
    if ty == 'object' and str(v) == 'null':
        return 'null'
    if ty == 'number':
        return str(v)
    return json.dumps(v)


def decompile(prog):
    out = ["// Arkose VM program - decompiled to pseudo-JavaScript",
           "// Reconstructed by symbolic stack execution. Control flow is kept as",
           "// labels + gotos; expressions are lifted. Verified against the runtime",
           "// crypto trace (importKey/encrypt) and the deobfuscated opcode handlers.",
           "//",
           "// v_<hex> = a VM-local variable/function id.  crypto == v_e38f0.",
           ""]

    st = []          # symbolic operand stack (each entry is a JS-expression string)
    staged = []      # xe: staged `this`-chain for the next CALL
    saved = []       # (st, staged) snapshots pushed at DEFUN, popped at ENDFUN
    indent = [0]

    def pad():
        return "  " * indent[0]

    def push(e):
        st.append(e)

    def pop():
        return st.pop() if st else "?"

    labels = {}
    for pc, line in enumerate(prog):
        tag, ops = parse(line)
        if tag == "b0280" and ops:
            labels.setdefault(pc, []).append(ops[0])

    for pc, line in enumerate(prog):
        tag, ops = parse(line)
        for lb in labels.get(pc, []):
            out.append("%s%s:" % (pad(), lb))

        if tag == "a82bc":                                   # DEFUN
            # At linear (load-time) execution the VM's DEFUN handler just scans
            # Re forward to the matching ENDFUN without touching the stack, so
            # the caller's pending operands must survive the body: save them.
            out.append("%sfunction %s() {" % (pad(), jsname(ops[0])))
            indent[0] += 1
            saved.append((list(st), list(staged)))
            st.clear(); staged.clear()
        elif tag == "b17bd":                                 # ENDFUN
            indent[0] = max(0, indent[0] - 1)
            out.append("%s}" % pad())
            if saved:
                s_st, s_staged = saved.pop()
                st[:] = s_st; staged[:] = s_staged
        elif tag == "a0cd9":                                 # RETURN
            out.append("%sreturn%s;" % (pad(), (" " + pop()) if st else ""))
        elif tag == "bb47b":                                 # LOAD_NAME
            nm = ops[0]
            push(jsname(nm) if re.match(r'^[0-9a-f]{5}$', nm) else dec(nm, KEY["name"]))
        elif tag == "b1539":                                 # PUSH_NAME (property key or deref)
            v = dec(ops[0], KEY["pushname"], strip=True)
            push(v if (len(ops) > 1 and ops[1] == "true") else json.dumps(v))
        elif tag in ("c1888", "c2b6b"):                      # PUSH_LIT
            push(lit(ops))
        elif tag == "e186f":                                 # PUSH_RAW
            push(json.dumps(ops[0]) if ops else "undefined")
        elif tag == "a2b49":                                 # NEW_OBJECT
            push("{}")
        elif tag == "e8265":                                 # NEW_ARRAY n
            n = int(ops[0]); items = [pop() for _ in range(n)][::-1]
            push("[" + ", ".join(items) + "]")
        elif tag == "e776b":                                 # SET_PROP key
            key = dec(ops[0], KEY["prop"]); val = pop(); obj = pop()
            base = obj[:-1] if obj.endswith("}") else obj
            inner = base[1:].strip()
            push("{%s%s: %s}" % (inner + ", " if inner else "", key, val))
        elif tag == "b89af":                                 # GET_MEMBER depth
            depth = int(ops[0]); keys = [pop() for _ in range(depth)][::-1]; base = pop()
            expr = base
            for k in keys:
                # quoted strings and numeric indices must stay bracketed
                bracket = k.startswith('"') or k.startswith("'") or re.match(r'^-?\d', k)
                expr += ("[%s]" % k) if bracket else (".%s" % k)
            staged.append(expr)
            push(expr)
        elif tag == "b1724":                                 # INDEX_GET
            k = pop(); o = pop(); push("%s[%s]" % (o, k))
        elif tag == "e73a3":                                 # CALL argc (async)
            # Calling convention (handler Fe): callee is on TOP (staged by the
            # preceding GET_MEMBER), argc args sit below it. Pop callee first.
            argc = int(ops[0]); callee = pop(); args = [pop() for _ in range(argc)][::-1]
            if staged:
                staged.pop()
            push("await %s(%s)" % (callee, ", ".join(args)))
        elif tag == "a5772":                                 # CALL_NAME name argc
            name = jsname(ops[0]); argc = int(ops[1]) if len(ops) > 1 else 0
            args = [pop() for _ in range(argc)][::-1]
            push("%s(%s)" % (name, ", ".join(args)))
        elif tag == "ac4e4":                                 # AWAIT
            v = pop()
            push(v if v.startswith("await ") else "await " + v)
        elif tag == "b404d":                                 # NEW argc ctor
            argc = int(ops[0]); ctor = ops[1]
            args = [pop() for _ in range(argc)][::-1]
            push("new %s(%s)" % (ctor, ", ".join(args)))
        elif tag in ("cea87", "e5b15"):                      # STORE / DECL
            # An empty stack here is real VM semantics, not a lift failure:
            # handler pops `undefined` and stores it, i.e. a hoisted `var x;`.
            if st:
                out.append("%s%s = %s;" % (pad(), jsname(ops[0]), pop()))
            else:
                out.append("%svar %s;  // = undefined (hoisted declaration)" % (pad(), jsname(ops[0])))
        elif tag == "aac6f":                                 # DELETE
            out.append("%sdelete %s;" % (pad(), jsname(ops[0])))
        elif tag == "ab1ed":                                 # SET_ERRNAME
            out.append("%s// catch as %s" % (pad(), ops[0]))
        elif tag == "c923e":                                 # JMP
            out.append("%sgoto %s;" % (pad(), ops[0]))
        elif tag == "d0c9a":                                 # JMP_FALSE
            out.append("%sif (!(%s)) goto %s;" % (pad(), pop(), ops[0]))
        elif tag == "b0280":                                 # LABEL (already emitted)
            pass
        elif tag == "ae498":                                 # BINOP
            b = pop(); a = pop(); push("(%s %s %s)" % (a, ops[0], b))
        elif tag == "b0d77":                                 # LOGIC (&& ||)
            b = pop(); a = pop(); push("(%s %s %s)" % (a, ops[0], b))
        elif tag == "b5558":                                 # UNARY
            push("(%s%s)" % (ops[0], pop()))
        elif tag == "e9293":                                 # JMP_LOGICAL op label
            # handler: o = pop(); push(o); if ((op=="||" && o) || (op=="&&" && !o)) Re = je[label]
            # i.e. PEEK-and-branch: the value stays on the stack for the fallthrough.
            top = st[-1] if st else "?"
            op = ops[0] if ops else ''
            cond = top if op == "||" else "!(%s)" % top
            out.append("%sif (%s) goto %s;  // %s short-circuit, value kept" % (pad(), cond, ops[1] if len(ops) > 1 else '?', op))
        elif tag == "e262e":                                 # SET_PATH
            # handler: u = pop() -> the {target,props} path pushed by GET_MEMBER (on top);
            #          s = pop() -> the value; then target.p1.p2... = s
            lhs = pop(); rhs = pop()
            out.append("%s%s = %s;" % (pad(), lhs, rhs))
        elif tag == "c9353":                                 # UPDATE ++/--
            out.append("%s%s%s;" % (pad(), ops[0], jsname(ops[1]) if len(ops) > 1 else ''))
        elif tag == "ddde1":                                 # POP
            if st:
                out.append("%s%s;" % (pad(), pop()))
        elif tag == "d1f65":                                 # TRY_BEGIN
            out.append("%stry {  // %s" % (pad(), ops[0]))
        elif tag == "b5da0":                                 # TRY_END
            out.append("%s} // end try %s" % (pad(), ops[0]))
        elif tag == "fe342":                                 # THROW
            out.append("%sthrow %s;" % (pad(), pop()))
        else:
            out.append("%s/* %s %s */" % (pad(), tag, ' '.join(ops)))

    return '\n'.join(out) + '\n'


def main():
    prog = json.load(open(sys.argv[1], encoding='utf-8'))
    src = decompile(prog)
    open(sys.argv[2], 'w', encoding='utf-8').write(src)
    print("decompiled %d instructions -> %s (%d lines)"
          % (len(prog), sys.argv[2], src.count('\n')))


if __name__ == '__main__':
    main()
