#!/usr/bin/env python3
"""Build npm tarball from already built native manager artifacts. No publishing."""
import argparse, hashlib, json, pathlib, tarfile, tempfile, shutil
p=argparse.ArgumentParser()
p.add_argument("--version",required=True)
p.add_argument("--windows",type=pathlib.Path)
p.add_argument("--macos",type=pathlib.Path)
p.add_argument("--out",required=True,type=pathlib.Path)
a=p.parse_args()
with tempfile.TemporaryDirectory() as d:
 root=pathlib.Path(d)/"package";root.mkdir()
 platforms={}
 for target,src in [("win32-x64",a.windows),("darwin-arm64",a.macos)]:
  if src is None:continue
  dest=root/"bin"/target;dest.mkdir(parents=True)
  if target=="win32-x64":
   entry=dest/"Routilet.exe";shutil.copy2(src,entry)
  else:
   app=dest/"Routilet.app";shutil.copytree(src,app,symlinks=True)
   entry=app/"Contents"/"MacOS"/"Routilet"
  platforms[target]={"executable":entry.relative_to(root).as_posix(),"sha256":hashlib.sha256(entry.read_bytes()).hexdigest()}
  notices=src.parent/"NOTICES.txt"
  if notices.is_file():shutil.copy2(notices,dest/"THIRD_PARTY_NOTICES.txt")
 if not platforms:raise SystemExit("no native artifacts")
 (root/"package.json").write_text(json.dumps({"name":"@routilet/routilet","version":a.version,"license":"MIT","description":"Routilet native application launcher","repository":{"type":"git","url":"git+https://github.com/Routilet/routilet.git"},"homepage":"https://github.com/Routilet/routilet","files":["bin","manifest.json","LICENSE"],"publishConfig":{"access":"public"}},indent=2),encoding="utf-8")
 (root/"manifest.json").write_text(json.dumps({"schemaVersion":1,"version":a.version,"platforms":platforms},indent=2),encoding="utf-8")
 shutil.copy2(pathlib.Path(__file__).resolve().parents[2]/"LICENSE",root/"LICENSE")
 a.out.parent.mkdir(parents=True,exist_ok=True)
 with tarfile.open(a.out,"w:gz") as tar:tar.add(root,arcname="package")
