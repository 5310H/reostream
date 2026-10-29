#!/usr/bin/env python3
"""Exercise the actual image without cameras; no credential/log dumps on failure."""
import hashlib
import re
import socket
import subprocess
import sys
import time
import urllib.parse
import urllib.request
import uuid

image = sys.argv[1]
name = 'reostream-smoke-' + uuid.uuid4().hex[:10]
volume = name + '-data'


def docker(*args):
    return subprocess.check_output(['docker', *args], stderr=subprocess.PIPE).decode()


def start():
    docker('run', '-d', '--name', name, '-v', volume + ':/data',
           '--network', 'host', '--health-interval=1s',
           '--health-start-period=1s', image)
    for _ in range(60):
        try:
            with urllib.request.urlopen('http://127.0.0.1:8560/api/status', timeout=1) as r:
                assert r.status == 200
            break
        except OSError:
            time.sleep(1)
    else:
        raise AssertionError('HTTP health endpoint did not become ready')
    for _ in range(30):
        if docker('inspect', '-f', '{{.State.Health.Status}}', name).strip() == 'healthy':
            break
        time.sleep(1)
    else:
        raise AssertionError('Docker health check did not become healthy')
    with socket.create_connection(('127.0.0.1', 8561), timeout=2) as s:
        s.sendall(b'OPTIONS rtsp://localhost:8561/ RTSP/1.0\r\nCSeq: 1\r\n\r\n')
        assert b'RTSP/1.0 200' in s.recv(4096)
    assert docker('exec', name, 'sh', '-c', "awk '/^Uid:/ {print $2}' /proc/1/status").strip() == '65532'


def stop():
    docker('stop', '-t', '30', name)
    assert docker('inspect', '-f', '{{.State.ExitCode}}', name).strip() == '0'
    # Go's log uses stderr; capture both without printing the first-run token.
    logs = subprocess.run(['docker', 'logs', name], capture_output=True, check=True)
    assert b'shutting down' in logs.stdout + logs.stderr
    docker('rm', name)


try:
    docker('volume', 'create', volume)
    start()
    logs = subprocess.run(['docker', 'logs', name], capture_output=True, check=True)
    token = re.search(rb'\b[A-Z0-9]{4}(?:-[A-Z0-9]{4}){3}\b', logs.stdout + logs.stderr)[0].decode()
    with urllib.request.urlopen('http://127.0.0.1:8562/claim', timeout=3) as r:
        assert r.status == 200
    form = urllib.parse.urlencode({'token': token, 'password': uuid.uuid4().hex}).encode()
    with urllib.request.urlopen('http://127.0.0.1:8562/claim', data=form, timeout=5) as r:
        assert r.status == 200  # follows redirect to login
    path = '/data/reostream/config.toml'
    before = hashlib.sha256(docker('exec', name, 'cat', path).encode()).digest()
    assert docker('exec', name, 'stat', '-c', '%a:%u', path).strip() == '600:65532'
    docker('exec', '-u', '65532:65532', name, 'sh', '-c',
           'mkdir -p /data/reostream/clips; echo retained > /data/reostream/clips/smoke')
    stop()
    start()
    after = hashlib.sha256(docker('exec', name, 'cat', path).encode()).digest()
    assert before == after, 'Configuration changed across recreation'
    assert docker('exec', name, 'cat', '/data/reostream/clips/smoke').strip() == 'retained'
    with urllib.request.urlopen('http://127.0.0.1:8562/', timeout=3) as r:
        assert r.geturl().endswith('/login'), 'Claimed UI must require login'
    stop()
    print('Image startup, Docker health, RTSP, claim, permissions, persistence and SIGTERM: OK')
finally:
    subprocess.run(['docker', 'rm', '-f', name], capture_output=True)
    subprocess.run(['docker', 'volume', 'rm', volume], capture_output=True)
