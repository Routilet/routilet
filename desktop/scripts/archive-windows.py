#!/usr/bin/env python3
"""Use ZIP UTF-8 names, including the single top-level folder."""
import sys, pathlib, zipfile
root=pathlib.Path(sys.argv[1])
with zipfile.ZipFile(sys.argv[2],"w",zipfile.ZIP_DEFLATED) as z:
 for p in sorted(root.rglob("*")):
  if p.is_file(): z.write(p,p.relative_to(root.parent).as_posix())
