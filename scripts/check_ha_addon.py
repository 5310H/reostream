#!/usr/bin/env python3
"""Check release/source consistency; authoritative metadata lint runs separately."""
from pathlib import Path
import re
import subprocess
import yaml

root = Path(__file__).resolve().parents[1]
addon = root / 'ha-addon/reostream'
config = yaml.safe_load((addon / 'config.yaml').read_text())
dockerfile = (addon / 'Dockerfile').read_text()
revision = re.search(r'ENV REOSTREAM_SOURCE=([0-9a-f]{40})\n', dockerfile)[1]
assert set(config['arch']) == {'amd64', 'aarch64'}
assert f'io.hass.version="{config["version"]}"' in dockerfile
assert f'control.version=ha-{config["version"]}-${{REOSTREAM_SOURCE}}' in dockerfile
assert f'org.opencontainers.image.revision="{revision}"' in dockerfile
assert config['host_network'] is True
assert config.get('options', {}) == config.get('schema', {}) == {}
assert 'image' not in config, 'This package must build from its pinned source'
# A later application change must update the pin and add-on version deliberately.
subprocess.run(['git', 'diff', '--exit-code', revision, '--', 'go.mod', 'go.sum',
                'cmd', 'internal', 'third_party'], cwd=root, check=True)
# Check the documented Frigate example is valid YAML and names real endpoints.
docs = (addon / 'DOCS.md').read_text()
example = yaml.safe_load(re.search(r'```yaml\n(.*?)```', docs, re.S)[1])
assert example['go2rtc']['streams']['driveway'].endswith(':8561/driveway')
assert example['go2rtc']['streams']['driveway_sub'].endswith(':8561/driveway_sub')
assert 'driveway' in example['cameras']
print(f'Add-on {config["version"]}, application {revision}: source consistency OK')
