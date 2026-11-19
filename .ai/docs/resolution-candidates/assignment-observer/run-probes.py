"""Run the archived assignment experiments through disposable Go overlays."""

import argparse
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("probe", choices=("assignments", "tokens", "benchmark"))
    parser.add_argument("paths", nargs="+", type=Path)
    args = parser.parse_args()

    paths = [path.resolve() for path in args.paths]
    if args.probe == "benchmark":
        if len(paths) != 1 or not paths[0].is_file():
            parser.error("benchmark expects the ColdBox controller file")
    elif not all(path.is_dir() for path in paths):
        parser.error("the corpus paths must be directories")

    here = Path(__file__).resolve().parent
    repo = here.parents[3]
    patch = here / ("assignment-offsets.patch" if args.probe == "assignments" else "checked-tokens.patch")
    files = [Path(line.removeprefix("+++ b/")) for line in patch.read_text().splitlines() if line.startswith("+++ b/")]
    env = dict(os.environ)
    env.setdefault("GOMAXPROCS", "2")
    if args.probe == "benchmark":
        env["CLIF_ASSIGN_PROBE_CONTROLLER"] = str(paths[0])
        flags = ["-run", "^$", "-bench", "^BenchmarkCollectionTokenReuseProbe$", "-benchtime=350ms", "-count=3"]
    else:
        env["CLIF_ASSIGN_PROBE_ROOTS"] = os.pathsep.join(map(str, paths))
        test = "TestAssignmentObserverCorpusProbe" if args.probe == "assignments" else "TestMainParseTokenTapeProbe"
        flags = ["-run", f"^{test}$", "-count=1", "-v"]

    with tempfile.TemporaryDirectory(prefix="clif-assignment-probe-") as directory:
        scratch = Path(directory)
        for path in files:
            source = repo / path
            if source.is_file():
                destination = scratch / path
                destination.parent.mkdir(parents=True, exist_ok=True)
                shutil.copyfile(source, destination)

        subprocess.run(["git", "apply", str(patch)], cwd=scratch, check=True)
        overlay = scratch / "overlay.json"
        overlay.write_text(json.dumps({"Replace": {str(repo / path): str(scratch / path) for path in files}}))
        subprocess.run(["go", "test", f"-overlay={overlay}", *flags, "./internal/parser"], cwd=repo, env=env, check=True)


if __name__ == "__main__":
    main()
