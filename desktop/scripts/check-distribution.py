#!/usr/bin/env python3
"""Run the actual extracted GUI executable in CLI mode with inherited pipes."""
import pathlib, subprocess, sys
exe=pathlib.Path(sys.argv[1]).resolve()
version=sys.argv[2]
def run(args):
 p=subprocess.run([str(exe),*args],cwd=str(exe.parent),stdout=subprocess.PIPE,stderr=subprocess.STDOUT,timeout=600)
 # CI on Windows may redirect stdout using CP1252. Emit UTF-8 bytes directly.
 sys.stdout.buffer.write(p.stdout)
 sys.stdout.buffer.flush()
 p.check_returncode()
 return p.stdout.decode("utf-8",errors="replace").strip()
assert run(["--manager-probe"])=="routilet-manager/1 "+version
if len(sys.argv)>3:
 run(["--self-update-test",str(pathlib.Path(sys.argv[3]).resolve()),"9999.0.0"])
else:
 run(["--self-test"])
