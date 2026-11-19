"""Integration checks for corpus discovery and overlay isolation."""

import os
from pathlib import Path
import socket
import subprocess
import sys
import tempfile
import unittest


RUNNER = Path(__file__).with_name("run-probes.py").resolve()


class RunnerTests(unittest.TestCase):
    def run_tokens(self, root, **env):
        return subprocess.run(
            [sys.executable, RUNNER, "tokens", root],
            env={**os.environ, **env}, capture_output=True, text=True, timeout=120,
        )

    def assert_one_stream(self, result):
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn("script regions=1 accepted-exact=1 fallback=0", result.stdout)

    @unittest.skipUnless(os.name == "posix", "requires Unix-domain sockets and symlinks")
    def test_socket_is_skipped_but_missing_target_is_reported(self):
        with tempfile.TemporaryDirectory(dir="/tmp") as directory:
            root = Path(directory)
            (root / "good.cfc").write_text("component { function f(){ return 1; } }")
            entry = root / "socket.cfc"
            with socket.socket(socket.AF_UNIX, socket.SOCK_STREAM) as listener:
                listener.bind(str(entry))
                (root / "alias.cfc").symlink_to(entry)
                self.assert_one_stream(self.run_tokens(root))

            entry.unlink()
            result = self.run_tokens(root)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn("alias.cfc", result.stdout + result.stderr)

    def test_tmpdir_inside_checkout(self):
        with tempfile.TemporaryDirectory() as directory:
            checkout = Path(directory).resolve()
            subprocess.run(["git", "init", "--quiet", checkout], check=True)
            scratch = checkout / "tmp"
            scratch.mkdir()
            corpus = checkout / "corpus"
            corpus.mkdir()
            (corpus / "good.cfc").write_text("component { function f(){ return 1; } }")
            self.assert_one_stream(self.run_tokens(corpus, TMPDIR=str(scratch)))
            self.assert_one_stream(self.run_tokens(
                corpus, TMPDIR=str(scratch), GIT_DIR=str(checkout / ".git"),
                GIT_WORK_TREE=str(checkout),
            ))


if __name__ == "__main__":
    unittest.main()
