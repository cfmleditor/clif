"""Prove consolidation boundary tests reject broken helpers without editing sources."""
from pathlib import Path
import json
import subprocess
import tempfile

root = Path(__file__).resolve().parents[2]
out = root / '.ai/docs/resolution-candidates/consolidation/mutations'
out.mkdir(parents=True, exist_ok=True)
cases = [
    ('filter', 'internal/resolve/token_readers.go', 'if !slices.ContainsFunc(tokens,',
     'if true || !slices.ContainsFunc(tokens,', './internal/resolve', '^TestWithoutNewlines'),
    ('call-opening', 'internal/resolve/token_readers.go', 'case parser.TokLParen:',
     'case parser.TokLBrace:', './internal/resolve', '^TestCallArgumentScanner'),
    ('group-depth', 'internal/parser/token_groups.go', 'depth := 0', 'depth := 1',
     './internal/parser', '^TestGroupEnd'),
    ('ancestor-cycle', 'internal/resolve/ancestors.go', '!seen[path] && ', '',
     './internal/resolve', '^TestAncestorTraversal'),
]
for name, file, old, new, package, test in cases:
    text = (root / file).read_text()
    assert text.count(old) == 1, (name, 'ambiguous mutation')
    with tempfile.TemporaryDirectory(prefix='clif-consolidation-mutation-') as tmp:
        changed = Path(tmp) / 'changed.go'
        changed.write_text(text.replace(old, new))
        overlay = Path(tmp) / 'overlay.json'
        overlay.write_text(json.dumps({'Replace': {str(root / file): str(changed)}}))
        result = subprocess.run(['go', 'test', '-overlay=' + str(overlay), '-short',
            '-count=1', '-timeout=1m', '-run=' + test, package], cwd=root,
            stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True, timeout=90)
        (out / (name + '.log')).write_text(result.stdout)
        if result.returncode == 0 or '--- FAIL:' not in result.stdout:
            raise RuntimeError(name + ': mutation did not fail behaviorally\n' + result.stdout)
        print(name + ': regression test fails as expected', flush=True)
