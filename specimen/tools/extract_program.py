"""Extract the VM program (the `ye` literal) from a deobfuscated api.js.

The program is a plain array of instruction strings, e.g. "a82bc b5eec",
"bb47b N[\\]]E", "b1539 'VSOQUXL[VV' false". It is identical to the runtime
`Te`, so the whole disassembly is static.

Usage: python extract_program.py api_clean.js vm/program.json
"""
import json
import re
import sys

BS = chr(92)


def parse_array(src, i):
    """src[i] == '['. Returns (list_of_strings, index_after_']')."""
    assert src[i] == '['
    i += 1
    out = []
    while i < len(src):
        c = src[i]
        if c in ' \t\r\n,':
            i += 1
        elif c == ']':
            return out, i + 1
        elif c == '"' or c == "'":
            q = c
            i += 1
            buf = []
            while i < len(src) and src[i] != q:
                if src[i] == BS:
                    n = src[i + 1]
                    if n == 'u':
                        if src[i + 2] == '{':
                            j = src.index('}', i)
                            buf.append(chr(int(src[i + 3:j], 16)))
                            i = j + 1
                        else:
                            buf.append(chr(int(src[i + 2:i + 6], 16)))
                            i += 6
                    elif n == 'x':
                        buf.append(chr(int(src[i + 2:i + 4], 16)))
                        i += 4
                    else:
                        buf.append({'n': '\n', 't': '\t', 'r': '\r', 'b': '\b',
                                    'f': '\f', 'v': '\v', '0': '\0'}.get(n, n))
                        i += 2
                else:
                    buf.append(src[i])
                    i += 1
            out.append(''.join(buf))
            i += 1
        else:
            raise ValueError('unexpected %r at %d' % (c, i))
    raise ValueError('unterminated array')


def main():
    src = open(sys.argv[1], encoding='utf-8').read()
    m = re.search(r'\bye\s*=\s*\[', src)
    if not m:
        raise SystemExit("no `ye` program array found")
    prog, _ = parse_array(src, src.index('[', m.start()))
    json.dump(prog, open(sys.argv[2], 'w', encoding='utf-8'))
    print("extracted %d instructions -> %s" % (len(prog), sys.argv[2]))


if __name__ == '__main__':
    main()
