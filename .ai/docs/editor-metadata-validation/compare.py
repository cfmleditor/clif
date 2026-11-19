"""Interleave editor sessions against isolated baseline and candidate binaries."""
from pathlib import Path
import json
import os
import re
import statistics
import subprocess

out = Path(__file__).resolve().parent / 'clean'
out.mkdir(exist_ok=True)
env = dict(os.environ, GOMAXPROCS='2',
    CLIF_EDITOR_SESSION=os.environ.get('CLIF_EDITOR_ROOT', '/Users/garethedwards/corpus/ColdBox_coldbox-platform'),
    CLIF_EDITOR_FILE='system/web/Controller.cfc', CLIF_EDITOR_IN='102',
    CLIF_EDITOR_OUT='88', CLIF_EDITOR_HOVER='128:33', CLIF_EDITOR_EDITS='200')
rows = {'before': [], 'after': []}
def micros(value):
    m = re.fullmatch(r'([\d.]+)(µs|ms|s)', value)
    return float(m[1]) * {'µs': 1, 'ms': 1000, 's': 1000000}[m[2]]
for sample in range(5):
    for label in (['before', 'after'] if sample % 2 == 0 else ['after', 'before']):
        print(label, sample, flush=True)
        binary = os.environ.get('CLIF_' + label.upper() + '_SERVER', '/tmp/clif-metadata-' + label + '-server.test')
        result = subprocess.run([binary, '-test.run=^TestEditorSession$', '-test.v', '-test.timeout=3m'],
            env=env, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True, check=True, timeout=200)
        (out / (label + '-' + str(sample) + '.log')).write_text(result.stdout)
        measurements = {}
        for line in result.stdout.splitlines():
            if 'didChange p50' not in line:
                continue
            values = re.search(r'didChange p50 (\S+) p95 (\S+) \| hover after p50 (\S+) p95 (\S+)', line)
            action = 'inside' if 'inside a function' in line else 'outside'
            measurements[action] = dict(zip(['change_p50_us', 'change_p95_us', 'hover_p50_us', 'hover_p95_us'], map(micros, values.groups())))
        rows[label].append(measurements)
        (out / 'samples.json').write_text(json.dumps(rows, indent=2) + '\n')
        summary = {label: {action: {metric: statistics.median(s[action][metric] for s in samples)
            for metric in samples[0][action]} for action in ['inside', 'outside']}
            for label, samples in rows.items() if samples}
        (out / 'summary.json').write_text(json.dumps(summary, indent=2) + '\n')
        print(measurements, flush=True)
