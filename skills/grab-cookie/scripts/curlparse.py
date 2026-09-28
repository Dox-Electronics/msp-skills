# /// script
# requires-python = ">=3.12"
# dependencies = []
# ///
"""Robust Copy-as-cURL header/cookie parser.

A naive parser handles only bash-style `-H '...'` and breaks on the rest.
Chrome's "Copy as cURL" has two flavours that differ in quoting and line
continuation, and which one a user gets depends on their platform:

  - Copy as cURL (bash) : -H 'name: value'   , backslash line continuation
  - Copy as cURL (cmd)  : -H "name: value"   , caret (^) line continuation

A backtick line continuation is also accepted, since a curl command reflowed in
a PowerShell buffer uses one. We also see bash ANSI-C quoting `-H $'name: value'`
when a value carries bytes that need escaping. This module handles all of that
and returns a lowercased header map (with the cookie merged under `cookie`).
Values are never printed; this module only parses text.

NOT SUPPORTED -- Chrome's "Copy as PowerShell" MENU ITEM. That is a different
thing from a curl command with backticks in it: it emits `Invoke-WebRequest`
with the headers in a PowerShell hashtable (`-Headers @{...}`), so there is no
`-H` or `--header` token anywhere to find. Pasting it parses to an empty header
map, and the failure surfaces later as the misleading "required credential not
found". Choose "Copy as cURL (bash)".

  parse_headers_from_file(path) -> dict[str, str]
  parse_headers(text)          -> dict[str, str]
  python curlparse.py --selfcheck
"""
from __future__ import annotations

import re
import sys

# -H / --header, in the three quote styles. Order: ANSI-C first ($'...') so the
# leading `$` is consumed, then plain single, then double.
_HDR_ANSIC = re.compile(r"(?:-H|--header)\s+\$'((?:[^'\\]|\\.)*)'")
_HDR_SINGLE = re.compile(r"(?:-H|--header)\s+'([^']*)'")
_HDR_DOUBLE = re.compile(r'(?:-H|--header)\s+"((?:[^"\\]|\\.)*)"')

# -b / --cookie, same three styles.
_CK_ANSIC = re.compile(r"(?:-b|--cookie)\s+\$'((?:[^'\\]|\\.)*)'")
_CK_SINGLE = re.compile(r"(?:-b|--cookie)\s+'([^']*)'")
_CK_DOUBLE = re.compile(r'(?:-b|--cookie)\s+"((?:[^"\\]|\\.)*)"')

# Line continuations: bash `\`, cmd `^`, PowerShell backtick - each at EOL.
# Matched IN PLACE by the scanner, only in unquoted state, and joined with
# nothing, exactly as the shell does (Chrome puts a space before the backslash,
# so headers still separate). A quote-blind pre-pass was wrong twice over: a
# space substitution split `x\<newline>-H'...'` into two words, and any
# substitution at all corrupted an escaped backslash before a newline inside a
# double-quoted body (`"x\\<newline>\" -H ..."`), closing the quote early.
_CONT = re.compile(r"[\\^`]\r?\n")


def _unescape_double(s: str) -> str:
    """Undo bash/cmd double-quote backslash escaping (\\", \\\\, \\`, \\$)."""
    out = []
    i = 0
    while i < len(s):
        c = s[i]
        if c == "\\" and i + 1 < len(s) and s[i + 1] in '"\\`$':
            out.append(s[i + 1])
            i += 2
        elif c == "\\" and (m := _CONT.match(s, i)):
            i = m.end()  # backslash-newline inside double quotes: removed (bash)
        else:
            out.append(c)
            i += 1
    return "".join(out)


def _unescape_ansic(s: str) -> str:
    """Undo bash ANSI-C ($'...') escaping for the sequences Chrome emits."""
    simple = {"n": "\n", "r": "\r", "t": "\t", "\\": "\\", "'": "'", '"': '"', "0": "\0"}
    out = []
    i = 0
    while i < len(s):
        c = s[i]
        if c != "\\" or i + 1 >= len(s):
            out.append(c)
            i += 1
            continue
        nxt = s[i + 1]
        if nxt == "x" and i + 3 < len(s) + 1:
            hexpart = s[i + 2 : i + 4]
            try:
                out.append(chr(int(hexpart, 16)))
                i += 4
                continue
            except ValueError:
                pass
        if nxt == "u" and i + 5 < len(s) + 1:
            hexpart = s[i + 2 : i + 6]
            try:
                out.append(chr(int(hexpart, 16)))
                i += 6
                continue
            except ValueError:
                pass
        if nxt in simple:
            out.append(simple[nxt])
            i += 2
            continue
        out.append(nxt)  # unknown escape: drop the backslash, keep the char
        i += 2
    return "".join(out)


def _add_header(headers: dict[str, str], raw: str) -> None:
    """Split one `Name: value` header string and store it lowercased."""
    idx = raw.find(":")
    if idx <= 0:
        return
    name = raw[:idx].strip().lower()
    value = raw[idx + 1 :].strip()
    if name:
        headers[name] = value


_FLAGS = ("--header", "--cookie", "-H", "-b")


def _iter_flag_values(text: str):
    """Yield (flag, unescaped_value) for each -H/--header/-b/--cookie in SOURCE ORDER.

    Walks the blob once, honouring quote state, so a quoted value is consumed
    whole and its contents can never be re-parsed as another flag. Handles the
    three quote styles Chrome emits ($'...', '...', "...") plus an unquoted
    token, and skips over any other quoted run so flags inside it stay inert.
    """
    i, n = 0, len(text)
    # True only at the start of a word: at the start of the text or right
    # after UNQUOTED whitespace. An escaped space (`x\ -H`) is part of the word,
    # so looking at the raw previous character was not enough.
    word_start = True
    while i < n:
        c = text[i]
        if c in "\\^`" and (m := _CONT.match(text, i)):
            i = m.end()  # unquoted line continuation: the word continues
            continue
        if c.isspace():
            i += 1
            word_start = True
            continue
        # Skip a quoted run that is not preceded by one of our flags: whatever
        # is inside belongs to that value, not to the command line.
        # An ANSI-C run ($'...') escapes its own quote as \', so the skip must
        # step over a backslash and the character after it, exactly as the
        # flag branch below does. Without this the skip stopped at \' and
        # whatever followed inside the body (e.g. -H "authorization: ...")
        # parsed as a real flag -- and Chrome puts --data-raw last, so it won.
        # A backslash OUTSIDE any quotes escapes the next character (POSIX):
        # `'it'\''s` is how Firefox and a hand-written command put an
        # apostrophe inside a single-quoted value. Without this step, the \'
        # opened a stray quoted run and whatever followed the value's real
        # closing quote parsed as the command line -- including a -H.
        if c == "\\":
            i += 2
            word_start = False
            continue
        if c == "$" and text.startswith("$'", i):
            j = i + 2
            while j < n and text[j] != "'":
                if text[j] == "\\":
                    j += 2
                    continue
                j += 1
            i = j + 1
            word_start = False
            continue
        if c == "'":
            j = i + 1
            while j < n and text[j] != "'":
                j += 1
            i = j + 1
            word_start = False
            continue
        if c == '"':
            j = i + 1
            while j < n:
                if text[j] == "\\":
                    j += 2
                    continue
                if text[j] == '"':
                    break
                j += 1
            i = j + 1
            word_start = False
            continue
        matched = None
        # A flag starts a word. Without the left boundary, `--data-raw
        # prefix-H'...'` matched the -H in the middle of the body argument.
        for flag in _FLAGS:
            if word_start and text.startswith(flag, i):
                after = i + len(flag)
                # Require a real boundary so --headerish does not match -H. A
                # continuation glued to the flag (`-H\<newline> 'a: b'`) is one.
                if after < n and (text[after].isspace() or text[after] in "'\"$"
                                  or _CONT.match(text, after)):
                    matched = (flag, after)
                    break
        if matched is None:
            i += 1
            word_start = False
            continue
        flag, j = matched
        # Between a flag and its value: whitespace and unquoted continuations
        # alike (`-H \<newline>  'a: b'` is legal shell), or the value came
        # back empty and the header was dropped.
        while j < n:
            if text[j].isspace():
                j += 1
            elif text[j] in "\\^`" and (m := _CONT.match(text, j)):
                j = m.end()
            else:
                break
        if j >= n:
            break
        value, k = _read_word(text, j)
        yield flag, value
        i = k
        word_start = False


def _read_word(text: str, j: int) -> tuple[str, int]:
    """Read ONE shell word starting at `j` and return (value, index after it).

    A shell word ends at unquoted whitespace, not at the first closing quote:
    `x:"abc -H 'a: b'"`, `'it'\\''s` and `'it'"'"'s` are each one word made of
    concatenated segments. Reading segment by segment is what keeps the
    invariant the scanner rests on -- nothing inside a value is ever re-read as
    a flag -- and it is also what makes `it's` extract as `it's` rather than `it`.
    An unterminated quote swallows the rest of the text: that is the safe
    direction, since nothing after it can then pose as a flag.
    """
    n = len(text)
    buf: list[str] = []
    k = j
    while k < n:
        c = text[k]
        if c in "\\^`" and (m := _CONT.match(text, k)):
            k = m.end()  # unquoted line continuation: the word continues
            continue
        if c.isspace():
            break
        if text.startswith("$'", k):  # ANSI-C, backslash escapes
            k += 2
            seg: list[str] = []
            while k < n and text[k] != "'":
                if text[k] == "\\" and k + 1 < n:
                    seg.append(text[k:k + 2])
                    k += 2
                    continue
                seg.append(text[k])
                k += 1
            buf.append(_unescape_ansic("".join(seg)))
            k += 1
        elif c == "'":  # plain single: no escapes inside
            close = text.find("'", k + 1)
            if close == -1:
                buf.append(text[k + 1:])
                k = n
                break
            buf.append(text[k + 1:close])
            k = close + 1
        elif c == '"':  # double, backslash escapes
            k += 1
            seg = []
            while k < n and text[k] != '"':
                if text[k] == "\\" and k + 1 < n:
                    seg.append(text[k:k + 2])
                    k += 2
                    continue
                seg.append(text[k])
                k += 1
            buf.append(_unescape_double("".join(seg)))
            k += 1
        elif c == "\\" and k + 1 < n:  # unquoted escape: literal next char
            buf.append(text[k + 1])
            k += 2
        else:
            buf.append(c)
            k += 1
    return "".join(buf), k


def parse_headers(text: str) -> dict[str, str]:
    """Parse a Copy-as-cURL blob into a lowercased header map.

    The raw cookie (`-b`/`--cookie` or a `cookie:` header) is always available
    under the `cookie` key. Later occurrences overwrite earlier ones.
    """
    headers: dict[str, str] = {}
    cookie_flag: str | None = None

    # Scan ONCE, in source order, tracking quote state -- never three separate
    # passes over the whole blob. Three passes are quoting-unaware: text that
    # merely LOOKS like `-H "..."` inside somebody else's quoted value is picked
    # up as a real header, and because the double-quote pass ran last it also
    # WINS. That lets any attacker-controlled substring of the victim's own
    # request (a cookie value, a URL query parameter) choose which credential
    # gets stored and wired -- and verify still passes, because the injected
    # session is live. Source order is also what the docstring promises.
    for flag, value in _iter_flag_values(text):
        if flag in ("-H", "--header"):
            _add_header(headers, value)
        elif flag in ("-b", "--cookie"):
            # -b / --cookie takes precedence for the cookie value if present.
            cookie_flag = value

    if cookie_flag is not None:
        headers["cookie"] = cookie_flag

    return headers


def parse_headers_from_file(path: str) -> dict[str, str]:
    with open(path, "r", encoding="utf-8", errors="replace") as fh:
        return parse_headers(fh.read())


def _selfcheck() -> None:
    # 1. bash single-quote
    h = parse_headers("curl 'https://x' -H 'authorization: tok123' -H 'x-realm: rrr'")
    assert h["authorization"] == "tok123" and h["x-realm"] == "rrr", h

    # 2. bash double-quote with an escaped quote in the value
    h = parse_headers('curl "https://x" -H "authorization: a\\"b"')
    assert h["authorization"] == 'a"b', h

    # 3. ANSI-C quoting with a hex escape
    h = parse_headers(r"curl 'https://x' -H $'x-test: a\x2Db'")
    assert h["x-test"] == "a-b", h

    # 4. cmd-style `^` line continuation + double quotes
    cmd = 'curl "https://x" ^\n  -H "accept: application/json" ^\n  -H "cookie: example_session=s%3Aabc; _ga=1"'
    h = parse_headers(cmd)
    assert h["accept"] == "application/json", h
    assert h["cookie"].startswith("example_session=s%3Aabc"), h

    # 5. cookie via -b flag (not a header)
    h = parse_headers("curl 'https://x' -b 'example_session=s%3Axyz; other=2'")
    assert h["cookie"].startswith("example_session=s%3Axyz"), h

    # 6. cookie via -H 'cookie: ...'
    h = parse_headers("curl 'https://x' -H 'cookie: example_session=s%3Aqqq'")
    assert h["cookie"] == "example_session=s%3Aqqq", h

    # 7. --header long form + value containing a colon (split on first only)
    h = parse_headers("curl 'https://x' --header 'authorization: Bearer a:b:c'")
    assert h["authorization"] == "Bearer a:b:c", h

    # 8. backtick continuation (a curl command reflowed in a PowerShell buffer;
    #    NOT Chrome's "Copy as PowerShell", which emits Invoke-WebRequest)
    h = parse_headers('curl "https://x" `\n  -H "x-realm: backtick"')
    assert h["x-realm"] == "backtick", h

    # 9. Chrome's "Copy as PowerShell" carries no -H tokens at all, so it must
    #    parse to nothing rather than appearing to half-work.
    pwsh = ('$session = New-Object Microsoft.PowerShell.Commands.WebRequestSession\n'
            'Invoke-WebRequest -UseBasicParsing -Uri "https://x" `\n'
            '  -Headers @{"x-realm"="nope"; "authorization"="Bearer zzz"}')
    assert parse_headers(pwsh) == {}, "Invoke-WebRequest must not parse as curl"

    # 10. A request body cannot choose the credential. Chrome's bash flavour
    #     writes a body containing ' in ANSI-C form with the quote escaped as
    #     \', and puts --data-raw last. The skip over that body must honour the
    #     escape, or the -H inside it parses as a real header and, by source
    #     order, overrides the real one. (Reported on Servosity/msp-skills#206.)
    attack = (r"""curl 'https://x' -H 'authorization: Bearer REAL' """
              r"""--data-raw $'x\' -H "authorization: Bearer ATTACKER"'""")
    h = parse_headers(attack)
    assert h.get("authorization") == "Bearer REAL", h

    # 11. Benign twin of 10: an escaped quote inside an ANSI-C body must not
    #     swallow the real header that follows it.
    benign = r"""curl 'https://x' --data-raw $'{"note":"it\'s done"}' -H 'x-realm: after'"""
    h = parse_headers(benign)
    assert h.get("x-realm") == "after", h

    # 12. Same attack through the POSIX '\'' idiom (Firefox's Copy as cURL, or a
    #     hand-written command): a backslash outside quotes escapes ONE character,
    #     so the value's real closing quote is the one after ATTACKER, and the -H
    #     inside stays inert.
    attack = (r"""curl 'https://x' -H 'authorization: Bearer REAL' """
              r"""--data-raw 'it'\''s -H "authorization: Bearer ATTACKER"'""")
    h = parse_headers(attack)
    assert h.get("authorization") == "Bearer REAL", h

    # 13. Benign twin of 12: a real header after a '\'' body must survive, and the
    #     '"'"' idiom (already handled by quote tracking) must behave the same.
    for benign in (r"""curl 'https://x' --data-raw 'it'\''s' -H 'x-realm: after'""",
                   r"""curl 'https://x' --data-raw 'it'"'"'s' -H 'x-realm: after'"""):
        h = parse_headers(benign)
        assert h.get("x-realm") == "after", h

    # 14. A value is a whole shell WORD, not its first quoted segment. A bare
    #     value with an embedded quoted run used to stop at the first space, so
    #     the -H inside the quotes parsed as real (and its benign twin dropped
    #     the header that followed).
    h = parse_headers("curl 'https://x' -H 'authorization: Bearer REAL' -H x:\"abc -H 'authorization: Bearer ATTACKER'\"")
    assert h.get("authorization") == "Bearer REAL", h
    assert h.get("x") == "abc -H 'authorization: Bearer ATTACKER'", h
    h = parse_headers("""curl 'https://x' -H x:"abc def" -H 'authorization: Bearer REAL'""")
    assert h.get("authorization") == "Bearer REAL" and h.get("x") == "abc def", h

    # 15. A flag must start a word: `prefix-H'...'` is one body argument, not a
    #     body followed by a header.
    h = parse_headers("""curl 'https://x' -H 'authorization: Bearer REAL' --data-raw prefix-H'authorization: Bearer ATTACKER'""")
    assert h.get("authorization") == "Bearer REAL", h

    # 16. Concatenated segments extract WHOLE: an apostrophe inside a credential
    #     survives in both POSIX idioms and in the ANSI-C form.
    for blob in (r"""curl 'https://x' -H 'cookie: a=it'\''s'""",
                 r"""curl 'https://x' -H 'cookie: a=it'"'"'s'""",
                 r"""curl 'https://x' -H $'cookie: a=it\'s'"""):
        h = parse_headers(blob)
        assert h.get("cookie") == "a=it's", (blob, h)

    # 17. An escaped space is part of the word, not a boundary: `x\ -H'...'` is
    #     one body argument.
    h = parse_headers("curl -H 'authorization: Bearer REAL' --data-raw x\\ -H'authorization: Bearer ATTACKER'")
    assert h.get("authorization") == "Bearer REAL", h

    # 18. A line continuation joins with NOTHING, as the shell does, so
    #     `x\<newline>-H'...'` is still one word; Chrome's ` \<newline>  -H` keeps
    #     its separating space either way.
    h = parse_headers("curl -H 'authorization: Bearer REAL' --data-raw x\\\n-H'authorization: Bearer ATTACKER'")
    assert h.get("authorization") == "Bearer REAL", h
    h = parse_headers("curl 'https://x' \\\n  -H 'authorization: Bearer REAL' \\\n  -H 'x-realm: after'")
    assert h.get("authorization") == "Bearer REAL" and h.get("x-realm") == "after", h

    # 19. Continuations are handled in quote state. An escaped backslash before
    #     a newline inside a double-quoted body is data, not a continuation; a
    #     quote-blind pre-pass ate one backslash and closed the quote early, so
    #     the -H after it parsed as real. LF and CRLF alike. Inside single
    #     quotes a backslash-newline is literal.
    for nl in ("\n", "\r\n"):
        body = 'curl -H \'authorization: Bearer REAL\' --data-raw "x\\\\' + nl + '\\" -H \'authorization: Bearer ATTACKER\' "'
        h = parse_headers(body)
        assert h.get("authorization") == "Bearer REAL", (nl, h)
        h = parse_headers("curl -H 'authorization: Bearer REAL' --data-raw 'x\\" + nl + "-H \"authorization: Bearer ATTACKER\"' -H 'x-realm: after'")
        assert h.get("authorization") == "Bearer REAL" and h.get("x-realm") == "after", (nl, h)
        h = parse_headers('curl -H "x-realm: a\\' + nl + 'b"')
        assert h.get("x-realm") == "ab", (nl, h)  # bash: backslash-newline removed inside ""

    # 20. A continuation between a flag and its value must not empty the value.
    for nl in ("\n", "\r\n"):
        h = parse_headers("curl -H \\" + nl + "  'authorization: Bearer REAL' --cookie ^" + nl + "  \"s=1\" -H 'x-realm: after'")
        assert h.get("authorization") == "Bearer REAL" and h.get("cookie") == "s=1" and h.get("x-realm") == "after", (nl, h)
        h = parse_headers("curl -H\\" + nl + " 'a: b'")  # glued to the flag, no space
        assert h.get("a") == "b", (nl, h)

    print("curlparse.py selfcheck OK (bash/cmd/backtick quoting, ANSI-C, -b + cookie header, "
          "a quoted body cannot inject a header, values read as whole shell words)")


if __name__ == "__main__":
    if "--selfcheck" in sys.argv:
        _selfcheck()
        raise SystemExit(0)
    if len(sys.argv) > 1:
        # Debug aid: print only header NAMES found in a capture file (never values).
        hdrs = parse_headers_from_file(sys.argv[1])
        print("headers found:", ", ".join(sorted(hdrs.keys())) or "(none)")
    else:
        print(__doc__)
