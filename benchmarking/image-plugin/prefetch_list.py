#!/usr/bin/env python3
# Copyright 2026 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""Prints the Nydus prefetch list for one SWE-bench image.

Reads `nydus-image check -v` output of the image's existing bootstrap on
stdin and prints, one path per line, the files a cold guest turn reads:

  - the testbed Python, the shared libraries it maps, and the stdlib
    modules loader/guest.py imports, from a trace of Python 3.6, 3.9 and
    3.11 images;
  - the first --reads *.py files under /testbed, in the guest's walk order.

Symlinks are resolved so the list names files that hold data.
"""

import argparse
import re
import sys

ENV = "/opt/miniconda3/envs/testbed"

# Files outside the env that the dynamic loader and Python open.
SYSTEM = [
    "/etc/ld.so.cache",
    "/usr/lib/locale/C.utf8/LC_CTYPE",
    "/usr/lib/x86_64-linux-gnu/gconv/gconv-modules.cache",
    "/usr/lib/x86_64-linux-gnu/ld-linux-x86-64.so.2",
    "/usr/lib/x86_64-linux-gnu/libc.so.6",
    "/usr/lib/x86_64-linux-gnu/libdl.so.2",
    "/usr/lib/x86_64-linux-gnu/libm.so.6",
    "/usr/lib/x86_64-linux-gnu/libpthread.so.0",
    "/usr/lib/x86_64-linux-gnu/libutil.so.1",
]

# Shared libraries under $ENV/lib, matched by name prefix.
ENV_LIBS = ["libbz2.so", "libcrypto.so", "libssl.so", "liblzma.so", "libz.so"]

# Extension modules under lib-dynload, by module name.
DYNLOAD = {
    "_bisect", "_bz2", "_datetime", "_heapq", "_json", "_lzma", "_random",
    "_sha512", "_socket", "_ssl", "_struct", "array", "binascii", "grp",
    "math", "select", "zlib", "_posixsubprocess", "_opcode", "fcntl",
    "_blake2", "_hashlib", "_sha3",
}

# Pure-Python stdlib modules, relative to lib/pythonX.Y, without ".py".
STDLIB = """
__future__ _bootlocale _collections_abc _compression _compat_pickle _sitebuiltins
_weakrefset abc argparse base64 bisect bz2 calendar codecs collections/__init__
collections/abc contextlib copy copyreg datetime email/__init__
email/_encoded_words email/_parseaddr email/_policybase email/base64mime
email/charset email/encoders email/errors email/feedparser email/header
email/iterators email/message email/parser email/quoprimime email/utils
encodings/__init__ encodings/aliases encodings/ascii encodings/latin_1 encodings/utf_8 enum
fnmatch functools genericpath gettext hashlib heapq html/__init__ html/entities
http/__init__ http/client http/server importlib/__init__ ipaddress
importlib/_abc importlib/_bootstrap importlib/_bootstrap_external
importlib/abc importlib/machinery importlib/util io json/__init__
json/decoder json/encoder json/scanner keyword linecache locale lzma mimetypes ntpath
operator os pathlib posixpath quopri random re re/__init__ re/_casefix
re/_compiler re/_constants re/_parser reprlib selectors shutil site socket
socketserver sre_compile sre_constants sre_parse ssl stat string struct
sysconfig textwrap threading token tokenize traceback types typing urllib/__init__ urllib/parse uu warnings weakref
zipimport
""".split()

LINE = re.compile(
    r'^inode: (\w+) "((?:[^"\\]|\\.)*)": .* link (None|Some\("((?:[^"\\]|\\.)*)"\))')


def unescape(s):
    return re.sub(r"\\(.)", r"\1", s)


def join(parent, name):
    return parent.rstrip("/") + "/" + name


def normalize(path):
    out = []
    for p in path.split("/"):
        if p in ("", "."):
            continue
        if p == "..":
            if out:
                out.pop()
        else:
            out.append(p)
    return "/" + "/".join(out)


class Tree:
    def __init__(self, lines):
        self.kind = {}
        self.link = {}
        self.children = {}
        for line in lines:
            m = LINE.match(line)
            if not m:
                continue
            path = unescape(m.group(2))
            # A hardlink is a regular file sharing an inode.
            self.kind[path] = "file" if m.group(1) == "hardlink" else m.group(1)
            if m.group(4) is not None:
                self.link[path] = unescape(m.group(4))
            if path != "/":
                parent, _, name = path.rpartition("/")
                self.children.setdefault(parent or "/", []).append(name)

    def resolve(self, path, depth=0):
        """Returns the path with all symlinks resolved, or None if missing."""
        if depth > 40:
            return None
        out = "/"
        parts = [p for p in normalize(path).split("/") if p]
        for i, part in enumerate(parts):
            cur = join(out, part)
            if cur not in self.kind:
                return None
            if self.kind[cur] == "symlink":
                target = self.link[cur]
                if not target.startswith("/"):
                    target = join(out, target)
                rest = "/".join(parts[i + 1:])
                return self.resolve(join(target, rest) if rest else target, depth + 1)
            out = cur
        return out

    def is_dir(self, path):
        r = self.resolve(path)
        return r is not None and self.kind.get(r) == "dir"

    def walk(self, top):
        """Yields (dirpath, dirnames, filenames) like os.walk(top)."""
        names = sorted(self.children.get(top, []))
        dirs = [n for n in names if self.is_dir(join(top, n))]
        files = [n for n in names if not self.is_dir(join(top, n))]
        yield top, dirs, files
        for d in dirs:
            # os.walk does not follow symlinked directories.
            if self.kind.get(join(top, d)) == "dir":
                yield from self.walk(join(top, d))


def runtime_files(tree):
    libdir = None
    for name in tree.children.get(ENV + "/lib", []):
        if re.fullmatch(r"python3\.\d+", name) and tree.kind.get(join(ENV + "/lib", name)) == "dir":
            libdir = join(ENV + "/lib", name)
    if libdir is None:
        sys.exit("no stdlib under %s/lib" % ENV)
    tag = "cpython-" + libdir.rsplit("python", 1)[1].replace(".", "")

    out = [ENV + "/bin/python"] + SYSTEM
    for name in sorted(tree.children.get(ENV + "/lib", [])):
        if any(name.startswith(p) for p in ENV_LIBS):
            out.append(join(ENV + "/lib", name))
    dyn = libdir + "/lib-dynload"
    for name in sorted(tree.children.get(dyn, [])):
        if name.split(".")[0] in DYNLOAD:
            out.append(join(dyn, name))
    for mod in STDLIB:
        pkg, _, base = mod.rpartition("/")
        d = join(libdir, pkg) if pkg else libdir
        pyc = base + "." + tag + ".pyc"
        if pyc in tree.children.get(d + "/__pycache__", []):
            # Python reads the .pyc and only stats the source.
            out.append(d + "/__pycache__/" + pyc)
        else:
            out.append(join(d, base + ".py"))
    site = libdir + "/site-packages"
    for name in sorted(tree.children.get(site, [])):
        # site.py reads every .pth file, and editable installs import a
        # finder module from theirs.
        if name.endswith(".pth") or (name.startswith("__editable__") and name.endswith(".py")):
            out.append(join(site, name))
    out.append(site + "/_distutils_hack/__init__.py")
    return out


def testbed_files(tree, root, limit):
    """Mirrors list_py in loader/guest.py."""
    found = []
    for dirpath, dirnames, filenames in tree.walk(root):
        dirnames[:] = [d for d in dirnames if not d.startswith(".")]
        for name in filenames:
            if name.endswith(".py"):
                found.append(join(dirpath, name))
                if len(found) >= limit:
                    return found
    return found


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--reads", type=int, default=1000)
    ap.add_argument("--mark-background", action="store_true",
                    help="print a #background line between the runtime files and the testbed files")
    args = ap.parse_args()
    tree = Tree(sys.stdin)
    if tree.resolve(ENV + "/bin/python") is None:
        sys.exit("no %s/bin/python" % ENV)
    seen = set()

    def emit(paths):
        for path in paths:
            real = tree.resolve(path)
            if real is None or tree.kind.get(real) != "file" or real in seen:
                continue
            seen.add(real)
            print(real)

    emit(runtime_files(tree))
    if args.mark_background:
        print("#background")
    emit(testbed_files(tree, "/testbed", args.reads))


if __name__ == "__main__":
    main()
