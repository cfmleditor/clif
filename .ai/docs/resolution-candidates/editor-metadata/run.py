"""Controlled corpus comparison for editor source metadata reuse."""
from pathlib import Path
import collections
import json
import os
import re
import subprocess

out = Path(__file__).resolve().parent / 'runs'
out.mkdir(exist_ok=True)
corpus = Path(os.environ.get('CLIF_CORPUS_ROOT', '/Users/garethedwards/corpus'))
projects = {'wheels': 'cfwheels_cfwheels', 'coldbox': 'ColdBox_coldbox-platform',
            'contentbox': 'Ortus-Solutions_ContentBox', 'testbox': 'Ortus-Solutions_TestBox',
            'fw1': 'framework-one_fw1', 'mura': 'MSU-NatSci_MuraCMS', 'lucee': 'lucee_Lucee'}
summary = {}
env = dict(os.environ, GOMAXPROCS='1')
for name, directory in projects.items():
    path = corpus / directory
    roots = [str(path)]
    if name == 'wheels':
        roots.append(str(path / 'vendor/wheels'))
    reports = {}
    summary[name] = {}
    for label, binary in [('before', os.environ['CLIF_BEFORE']), ('after', os.environ['CLIF_AFTER'])]:
        print(label + ': ' + name, flush=True)
        with (out / f'{label}-{name}.json').open('w') as data, (out / f'{label}-{name}.log').open('w') as log:
            subprocess.run([binary, 'unresolved', '--json', '--candidates', '--verbose'] + roots,
                           cwd=corpus, env=env, stdout=data, stderr=log, check=True)
        reports[label] = json.loads((out / f'{label}-{name}.json').read_text())
        text = (out / f'{label}-{name}.log').read_text()
        summary[name][label] = dict(re.findall(r'(Index|Scan):\s+\S+ \((\d+) files\)', text))
        summary[name][label]['findings'] = len(reports[label])
        summary[name][label]['resolved'] = int(re.search(r'\((\d+) resolved\)', text).group(1))
    key = lambda e: json.dumps(e, sort_keys=True)
    a, b = [collections.Counter(map(key, reports[label])) for label in ('before', 'after')]
    for label, delta in [('removed', a-b), ('added', b-a)]:
        entries = [json.loads(k) for k, count in delta.items() for _ in range(count)]
        (out / f'{label}-{name}.json').write_text(json.dumps(entries, indent=2) + '\n')
        summary[name][label] = len(entries)
    (out / 'comparison.json').write_text(json.dumps(summary, indent=2) + '\n')
    print(json.dumps({name: summary[name]}), flush=True)
