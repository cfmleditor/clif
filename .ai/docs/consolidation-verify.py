"""Compare corpus parsing, editor passes and isolated LSP benchmark processes."""
from pathlib import Path
import hashlib
import json
import os
import re
import statistics
import subprocess

root = Path(__file__).resolve().parents[2]
out = root / '.ai/docs/consolidation-validation'
out.mkdir(exist_ok=True)
corpus = Path('/Users/garethedwards/corpus')
projects = ['cfwheels_cfwheels', 'ColdBox_coldbox-platform', 'Ortus-Solutions_ContentBox',
            'Ortus-Solutions_TestBox', 'framework-one_fw1', 'MSU-NatSci_MuraCMS', 'lucee_Lucee']
env = dict(os.environ, GOMAXPROCS='2')

def run(args, filename, extra=None):
    result = subprocess.run(args, cwd=root, env=dict(env, **(extra or {})), text=True,
                            stdout=subprocess.PIPE, stderr=subprocess.STDOUT, timeout=240)
    (out / filename).write_text(result.stdout)
    if result.returncode:
        raise RuntimeError(filename + '\n' + result.stdout[-4000:])
    return result.stdout

dump_dir = root / '.ai/docs/resolution-candidates/consolidation/parser'
dump_dir.mkdir(exist_ok=True)
for label in ['before', 'after']:
    print('parser corpus: ' + label, flush=True)
    run(['/tmp/clif-consolidation-' + label + '-parser.test', '-test.run=^TestCorpusResultDump$',
         '-test.v'], 'parser-' + label + '.log', {'CFML_CORPUS': ':'.join(str(corpus / p) for p in projects),
         'CLIF_DUMP': str(dump_dir / (label + '.txt'))})
before, after = [(dump_dir / (label + '.txt')).read_bytes() for label in ['before', 'after']]
if before != after:
    raise RuntimeError('parser corpus dump changed')
(dump_dir / 'comparison.json').write_text(json.dumps({'identical': True, 'bytes': len(before),
    'sha256': hashlib.sha256(before).hexdigest(), 'header': before.splitlines()[0].decode()}, indent=2)+'\n')
print('parser dumps identical', flush=True)

coldbox = corpus / 'ColdBox_coldbox-platform'
editor = {'CLIF_PASSES_EDITOR': str(coldbox), 'CLIF_PASSES_FILE': 'system/web/Controller.cfc',
          'CLIF_PASSES_SECOND': 'system/web/context/RequestContext.cfc',
          'CLIF_PASSES_HOVER': '128:33', 'CLIF_PASSES_IN': '102', 'CLIF_PASSES_OUT': '88'}
passes = {}
for label in ['before', 'after']:
    print('editor passes: ' + label, flush=True)
    output = run(['/tmp/clif-consolidation-' + label + '-passes.test',
                  '-test.run=^TestEditorActionPasses$', '-test.v'], 'passes-' + label + '.log', editor)
    passes[label] = re.findall(r'editor_passes_test.go:\d+: (.*?)\s+([\d.]+) passes, (\d+) parses', output)
(out / 'passes.json').write_text(json.dumps(passes, indent=2)+'\n')

cases = ['BenchmarkCompletion', 'BenchmarkCompletionWithMarshal/full',
         'BenchmarkCompletionWithMarshal/deferred', 'BenchmarkWorkspaceSymbolWithMarshal/query_m',
         'BenchmarkTyping/funcs20', 'BenchmarkTyping/funcs200',
         'BenchmarkKeystroke/funcs20', 'BenchmarkKeystroke/funcs200',
         'BenchmarkIndexBackedHandlers/hover', 'BenchmarkIndexBackedHandlers/definition']
summary = {}
for name in cases:
    print('LSP: ' + name, flush=True)
    samples = {'before': [], 'after': []}
    pattern = '/'.join('^' + re.escape(piece) + '$' for piece in name.split('/'))
    for n in range(3):
        for label in (['before', 'after'] if n % 2 == 0 else ['after', 'before']):
            binary = '/tmp/clif-server-bench.test' if label == 'before' else '/tmp/clif-consolidation-after-server.test'
            output = run([binary, '-test.run=^$', '-test.bench=' + pattern,
                          '-test.benchmem', '-test.benchtime=300ms'],
                         name.replace('/', '-') + '-' + label + '-' + str(n) + '.log')
            m = re.findall(r'^Benchmark\S+\s+\d+\s+([\d.]+) ns/op\s+(\d+) B/op\s+(\d+) allocs/op', output, re.M)
            if len(m) != 1:
                raise RuntimeError(output)
            samples[label].append([float(m[0][0]), int(m[0][1]), int(m[0][2])])
    summary[name] = {}
    for label, values in samples.items():
        summary[name][label] = {'ns': statistics.median(v[0] for v in values),
            'bytes': statistics.median(v[1] for v in values), 'allocs': statistics.median(v[2] for v in values),
            'samples': values}
    summary[name]['time_delta_percent'] = 100*(summary[name]['after']['ns']/summary[name]['before']['ns']-1)
    (out / 'benchmarks.json').write_text(json.dumps(summary, indent=2)+'\n')
    print(name + ': ' + str(round(summary[name]['time_delta_percent'], 1)) + '%', flush=True)

session = {'CLIF_EDITOR_SESSION': str(coldbox), 'CLIF_EDITOR_FILE': 'system/web/Controller.cfc',
           'CLIF_EDITOR_IN': '102', 'CLIF_EDITOR_OUT': '88', 'CLIF_EDITOR_HOVER': '128:33',
           'CLIF_EDITOR_EDITS': '200', 'GOMAXPROCS': '2'}
for label, binary in [('before', '/tmp/clif-server-bench.test'), ('after', '/tmp/clif-consolidation-after-server.test')]:
    print('editor session: ' + label, flush=True)
    run([binary, '-test.run=^TestEditorSession$', '-test.v'], 'editor-' + label + '.log', session)
print('all comparisons completed', flush=True)
