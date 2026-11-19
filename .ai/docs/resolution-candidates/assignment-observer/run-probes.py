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
    buffered_probes = {"buffered-benchmark": "selective-benchmark",
                       "buffered-editor": "editor", "pooled-editor": "editor", "buffered-selective": "selective"}
    parser.add_argument("probe", choices=("assignments", "tokens", "benchmark", "selective",
                                          "selective-benchmark", "editor", *buffered_probes))
    parser.add_argument("paths", nargs="+", type=Path)
    args = parser.parse_args()
    pooled_editor = args.probe == "pooled-editor"
    buffered = args.probe in buffered_probes
    if buffered:
        args.probe = buffered_probes[args.probe]

    paths = [path.resolve() for path in args.paths]
    if args.probe.endswith("benchmark"):
        if len(paths) != 1 or not paths[0].is_file():
            parser.error("benchmark expects the ColdBox controller file")
    elif not all(path.is_dir() for path in paths):
        parser.error("the corpus paths must be directories")

    here = Path(__file__).resolve().parent
    repo = here.parents[3]
    patch = here / ("assignment-offsets.patch" if args.probe == "assignments" else "checked-tokens.patch")
    if args.probe.startswith("selective") or args.probe == "editor":
        patch = here / "selective-tokens.patch"
    patches = [patch]
    if buffered:
        patches.append(here / "buffered-capture.patch")
    if args.probe == "editor":
        patches.append(here / "editor-workload.patch")
    if pooled_editor:
        patches.append(here / "pooled-editor-control.patch")
    files = [Path(line.removeprefix("+++ b/")) for source in patches
             for line in source.read_text().splitlines() if line.startswith("+++ b/")]
    env = dict(os.environ)
    env.setdefault("GOMAXPROCS", "2")
    if args.probe.endswith("benchmark"):
        env["CLIF_ASSIGN_PROBE_CONTROLLER"] = str(paths[0])
        flags = ["-run", "^$", "-bench", "^BenchmarkCollectionTokenReuseProbe$", "-benchtime=350ms", "-count=3"]
    else:
        env["CLIF_ASSIGN_PROBE_ROOTS"] = os.pathsep.join(map(str, paths))
        test = "TestAssignmentObserverCorpusProbe" if args.probe == "assignments" else "TestMainParseTokenTapeProbe"
        if args.probe == "selective":
            tests = [test, "TestCaptureDemandAfterFullReplace"]
            if buffered:
                tests.append("TestBufferedCaptureOwnsPublishedTokens")
            test = "(" + "|".join(tests) + ")"
        flags = ["-run", f"^{test}$", "-count=1", "-v"]

    with tempfile.TemporaryDirectory(prefix="clif-assignment-probe-") as directory:
        scratch = Path(directory)
        for path in files:
            source = repo / path
            if source.is_file():
                destination = scratch / path
                destination.parent.mkdir(parents=True, exist_ok=True)
                shutil.copyfile(source, destination)

        for source in patches:
            subprocess.run(["git", "apply", str(source)], cwd=scratch, check=True)
        replacements = {str(repo / path): str(scratch / path) for path in files}
        package = "./internal/parser"
        if args.probe == "editor":
            package = "./internal/server"
            flags = ["-run", "^TestCorpusEditorTokenReuse$", "-count=1", "-v", "-timeout=30m"]
        overlay = scratch / "overlay.json"
        overlay.write_text(json.dumps({"Replace": replacements}))
        subprocess.run(["go", "test", f"-overlay={overlay}", *flags, package], cwd=repo, env=env, check=True)


if __name__ == "__main__":
    main()
