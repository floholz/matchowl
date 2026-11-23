#!/usr/bin/env python3
"""Keep design/matchowl.figcraft and its committed source folder in sync.

The .figcraft file is a zip (document.json + images/ + thumbnail.png). Git
can't diff or merge a zip, so the repo tracks the unpacked folder instead and
the .figcraft is a local, gitignored build of it.

    python3 design/tools/figcraft_sync.py status   # what changed since the last sync
    python3 design/tools/figcraft_sync.py unpack   # .figcraft -> folder (after editing in FigCraft)
    python3 design/tools/figcraft_sync.py pack     # folder -> .figcraft (after clone / pull)

Both directions refuse to overwrite work that hasn't been synced yet: a
sidecar stamp (design/.matchowl.figcraft.sync, gitignored) records the hash
of both sides at the last sync. --force skips that check.

Stdlib only, so it runs anywhere python3 does.
"""

import argparse
import hashlib
import io
import json
import sys
import zipfile
from pathlib import Path

DESIGN = Path(__file__).resolve().parent.parent
FIGCRAFT = DESIGN / "matchowl.figcraft"
SRC = DESIGN / "matchowl"
STAMP = DESIGN / ".matchowl.figcraft.sync"

DOC = "document.json"
THUMB = "thumbnail.png"
IMAGES = "images"

# Keys whose values are long machine data nobody reads line by line (vector
# path networks). They stay on one line so the file stays small.
INLINE_KEYS = {"network"}
# Containers whose compact form fits in this many columns stay on one line.
INLINE_WIDTH = 100


# ---------------------------------------------------------------- formatting

def _compact(v):
    return json.dumps(v, ensure_ascii=False, separators=(",", ":"))


def _dump(v, ind, out, inline=False):
    if not isinstance(v, (dict, list)):
        out.append(json.dumps(v, ensure_ascii=False))
        return
    c = _compact(v)
    if inline or not v or len(c) + ind <= INLINE_WIDTH:
        out.append(c)
        return
    pad = " " * (ind + 1)
    if isinstance(v, dict):
        out.append("{\n")
        for i, (k, x) in enumerate(v.items()):
            if i:
                out.append(",\n")
            out.append(pad + json.dumps(k, ensure_ascii=False) + ": ")
            _dump(x, ind + 1, out, inline=k in INLINE_KEYS)
        out.append("\n" + " " * ind + "}")
    else:
        out.append("[\n")
        for i, x in enumerate(v):
            if i:
                out.append(",\n")
            out.append(pad)
            _dump(x, ind + 1, out)
        out.append("\n" + " " * ind + "]")


def format_doc(doc):
    """Readable, diff-friendly and deterministic. Key order is FigCraft's own."""
    out = []
    _dump(doc, 0, out)
    return ("".join(out) + "\n").encode("utf-8")


def normalize(doc):
    """Drop editor view state so just looking around doesn't produce a diff."""
    if doc.get("pages"):
        doc["current_page"] = doc["pages"][0]
    return doc


# ---------------------------------------------------------------- the two sides

def read_folder():
    """{zip path: bytes} from the committed folder, or None if it's missing."""
    if not (SRC / DOC).exists():
        return None
    files = {DOC: (SRC / DOC).read_bytes()}
    for p in sorted((SRC / IMAGES).glob("*")):
        if p.is_file():
            files[f"{IMAGES}/{p.name}"] = p.read_bytes()
    if (SRC / THUMB).exists():
        files[THUMB] = (SRC / THUMB).read_bytes()
    return files


def read_figcraft():
    """{zip path: bytes} from the .figcraft, document normalized and formatted."""
    with zipfile.ZipFile(FIGCRAFT) as z:
        files = {}
        for name in z.namelist():
            if name.endswith("/"):
                continue
            data = z.read(name)
            if name == DOC:
                data = format_doc(normalize(json.loads(data)))
            files[name] = data
    return files


def folder_hash(files):
    h = hashlib.sha256()
    for name in sorted(files):
        h.update(name.encode() + b"\0" + hashlib.sha256(files[name]).digest())
    return h.hexdigest()


def file_hash(path):
    return hashlib.sha256(path.read_bytes()).hexdigest() if path.exists() else None


def build_zip(files):
    """Deterministic zip: fixed timestamps, FigCraft's member order."""
    doc = json.loads(files[DOC])
    order = [DOC] + sorted(n for n in files if n.startswith(IMAGES + "/"))
    order += [n for n in files if n not in order]
    buf = io.BytesIO()
    with zipfile.ZipFile(buf, "w", zipfile.ZIP_DEFLATED) as z:
        for name in order:
            data = _compact(doc).encode("utf-8") if name == DOC else files[name]
            info = zipfile.ZipInfo(name, date_time=(1980, 1, 1, 0, 0, 0))
            info.compress_type = zipfile.ZIP_DEFLATED
            info.external_attr = 0o644 << 16
            z.writestr(info, data)
    return buf.getvalue()


# ---------------------------------------------------------------- sync state

def load_stamp():
    try:
        return json.loads(STAMP.read_text())
    except (OSError, ValueError):
        return {}


def save_stamp(folder_files):
    STAMP.write_text(json.dumps({
        "figcraft": file_hash(FIGCRAFT),
        "folder": folder_hash(folder_files),
    }) + "\n")


def state():
    """Returns (figcraft_changed, folder_changed) since the last sync."""
    folder = read_folder()
    if folder is None:
        return FIGCRAFT.exists(), False
    if not FIGCRAFT.exists():
        return False, True
    stamp = load_stamp()
    fig_changed = file_hash(FIGCRAFT) != stamp.get("figcraft")
    dir_changed = folder_hash(folder) != stamp.get("folder")
    if fig_changed and dir_changed:
        # No stamp yet (fresh setup) or both moved: maybe they still agree.
        if folder_hash(read_figcraft()) == folder_hash(folder):
            save_stamp(folder)
            return False, False
    return fig_changed, dir_changed


# ---------------------------------------------------------------- commands

def cmd_status(_):
    fig, folder = state()
    if fig and folder:
        print("conflict: both matchowl.figcraft and design/matchowl/ changed since the last sync")
        return 3
    if fig:
        print("matchowl.figcraft has edits not unpacked yet (run: make design-unpack)")
        return 1
    if folder:
        print("design/matchowl/ is newer than matchowl.figcraft (run: make design-pack)")
        return 2
    print("in sync")
    return 0


def cmd_unpack(args):
    if not FIGCRAFT.exists():
        sys.exit(f"no {FIGCRAFT.relative_to(DESIGN.parent)} to unpack")
    fig, folder = state()
    if folder and not args.force:
        sys.exit("design/matchowl/ changed since the last sync (a pull, or edits made there).\n"
                 "Unpacking would overwrite that. Pack first to load them into FigCraft,\n"
                 "or rerun with --force to let the .figcraft win.")
    files = read_figcraft()
    written = 0
    (SRC / IMAGES).mkdir(parents=True, exist_ok=True)
    for name, data in files.items():
        p = SRC / name
        if not p.exists() or p.read_bytes() != data:
            p.parent.mkdir(parents=True, exist_ok=True)
            p.write_bytes(data)
            written += 1
    removed = 0
    for p in (SRC / IMAGES).glob("*"):
        if f"{IMAGES}/{p.name}" not in files:
            p.unlink()
            removed += 1
    if THUMB not in files and (SRC / THUMB).exists():
        (SRC / THUMB).unlink()
        removed += 1
    save_stamp(files)
    print(f"unpacked: {written} file(s) updated, {removed} removed" if written or removed
          else "unpacked: no changes")


def cmd_pack(args):
    folder = read_folder()
    if folder is None:
        sys.exit("no design/matchowl/document.json to pack")
    fig, _ = state()
    if fig and FIGCRAFT.exists() and not args.force:
        sys.exit("matchowl.figcraft has edits that aren't unpacked yet.\n"
                 "Packing would overwrite them. Unpack first (and commit), or rerun with\n"
                 "--force to throw those edits away.")
    data = build_zip(folder)
    if FIGCRAFT.exists() and FIGCRAFT.read_bytes() == data:
        print("packed: already up to date")
    else:
        FIGCRAFT.write_bytes(data)
        print(f"packed: {FIGCRAFT.relative_to(DESIGN.parent)} ({len(data) / 1e6:.1f} MB)")
    save_stamp(folder)


def main():
    ap = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    sub = ap.add_subparsers(dest="cmd", required=True)
    sub.add_parser("status").set_defaults(fn=cmd_status)
    for name, fn in (("pack", cmd_pack), ("unpack", cmd_unpack)):
        p = sub.add_parser(name)
        p.add_argument("--force", action="store_true", help="skip the unsynced-work check")
        p.set_defaults(fn=fn)
    args = ap.parse_args()
    sys.exit(args.fn(args) or 0)


if __name__ == "__main__":
    main()
