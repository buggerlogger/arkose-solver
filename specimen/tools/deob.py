import re, json, sys

BS = chr(92)  # backslash

src = open(sys.argv[1], encoding='utf-8').read()

ESC = {'n': '\n', 't': '\t', 'r': '\r', 'b': '\b', 'f': '\f', 'v': '\v', '0': '\0'}


def parse_js_string(s, i):
    """s[i] is a quote char. Returns (value, index_after_closing_quote)."""
    q = s[i]
    i += 1
    buf = []
    while i < len(s):
        c = s[i]
        if c == BS:
            n = s[i + 1]
            if n == 'u':
                if s[i + 2] == '{':
                    j = s.index('}', i)
                    buf.append(chr(int(s[i + 3:j], 16)))
                    i = j + 1
                else:
                    buf.append(chr(int(s[i + 2:i + 6], 16)))
                    i += 6
            elif n == 'x':
                buf.append(chr(int(s[i + 2:i + 4], 16)))
                i += 4
            else:
                buf.append(ESC.get(n, n))
                i += 2
        elif c == q:
            return ''.join(buf), i + 1
        else:
            buf.append(c)
            i += 1
    raise ValueError('unterminated string')


def parse_js_array_of_strings(s, i):
    """s[i] == '['. Returns (list, index_after_closing_bracket)."""
    assert s[i] == '['
    i += 1
    out = []
    while i < len(s):
        c = s[i]
        if c in ' \t\r\n,':
            i += 1
        elif c == ']':
            return out, i + 1
        elif c == '"' or c == "'":
            v, i = parse_js_string(s, i)
            out.append(v)
        else:
            raise ValueError('unexpected char %r at %d' % (c, i))
    raise ValueError('unterminated array')


# 1) decoders
dec_re = re.compile(
    r'function\s+([A-Za-z_$][\w$]*)\s*\(\s*t\s*,\s*e\s*\)\s*\{\s*var\s+n\s*=\s*([A-Za-z_$][\w$]*)\(\)\s*;'
    r'.*?return\s+n\[\(?\s*t\s*-=\s*(\d+)\s*\)?\]\s*;?', re.S)
decoders = {}
for m in dec_re.finditer(src):
    decoders[m.group(1)] = (m.group(2), int(m.group(3)))

# 2) tables
tables = {}
for name in set(v[0] for v in decoders.values()):
    m = re.search(r'function\s+' + re.escape(name) + r'\(\s*\)\s*\{\s*var\s+t\s*=\s*\[', src)
    if not m:
        tables[name] = None
        continue
    i = src.index('[', m.end() - 1)
    tables[name], _ = parse_js_array_of_strings(src, i)

summary = {}
for dec, (tbl, off) in sorted(decoders.items()):
    t = tables.get(tbl)
    summary[dec] = {'table': tbl, 'offset': off, 'size': len(t) if t else 0}
print(json.dumps(summary, indent=1))

json.dump({'decoders': {k: {'table': v[0], 'offset': v[1]} for k, v in decoders.items()},
           'tables': tables}, open(sys.argv[2], 'w', encoding='utf-8'))
print('wrote', sys.argv[2])
