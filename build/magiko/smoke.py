#!/usr/bin/env python3
"""Smoke-test the built embedded web UI without real HOME, accounts or keychain."""
import http.cookiejar
import json
import os
from pathlib import Path
import re
import selectors
import signal
import subprocess
import tempfile
import time
import urllib.error
import urllib.request

root = Path(__file__).resolve().parents[2]
binary = root / 'dist/Magiko.app/Contents/MacOS/magiko'
with tempfile.TemporaryDirectory(prefix='magiko-smoke-') as home:
    tmp = Path(home)
    bin_dir = tmp / 'bin'
    bin_dir.mkdir()
    for name in ('security', 'secret-tool', 'claude', 'codex', 'cursor-agent', 'devin', 'grok', 'kiro-cli'):
        standin = bin_dir / name
        standin.write_text('#!/bin/sh\nexit 1\n')
        standin.chmod(0o700)
    env = {
        'HOME': home, 'USERPROFILE': home, 'PATH': f'{bin_dir}:/usr/bin:/bin',
        'XDG_CONFIG_HOME': str(tmp / 'config'), 'XDG_CACHE_HOME': str(tmp / 'cache'),
        'CODEX_HOME': str(tmp / '.codex'), 'TMPDIR': home,
        'MAGPIE_ADDR': '127.0.0.1:0',
        'MAGPIE_WEB_KEY': '0123456789abcdef0123456789abcdef',
        'HTTP_PROXY': 'http://127.0.0.1:9', 'HTTPS_PROXY': 'http://127.0.0.1:9',
        'ALL_PROXY': 'http://127.0.0.1:9',
    }
    with (tmp / 'stderr.log').open('w+') as log:
        proc = subprocess.Popen([str(binary), 'web', '--no-open', '--addr', '127.0.0.1:0'],
                                env=env, stdout=subprocess.PIPE, stderr=log,
                                text=True, start_new_session=True)
        try:
            sel = selectors.DefaultSelector()
            sel.register(proc.stdout, selectors.EVENT_READ)
            link = None
            deadline = time.monotonic() + 30
            while time.monotonic() < deadline:
                if sel.select(timeout=1):
                    line = proc.stdout.readline()
                    match = re.search(r'http://127\.0\.0\.1:\d+/\?k=[A-Za-z0-9]+', line)
                    if match:
                        link = match.group(0)
                        break
                if proc.poll() is not None:
                    break
            if not link:
                log.seek(0)
                raise RuntimeError('No web link: ' + log.read())
            base = link.split('/?')[0]
            cookies = http.cookiejar.CookieJar()
            opener = urllib.request.build_opener(urllib.request.ProxyHandler({}), urllib.request.HTTPCookieProcessor(cookies))
            html = opener.open(link, timeout=10).read().decode()
            assert '<title>Magiko</title>' in html
            assert 'plugins/magiko/client.js?v=' in html
            boot = opener.open(base + '/boot.js', timeout=10).read().decode()
            assert '"magiko":true' in boot
            state = json.load(opener.open(base + '/api/magiko', timeout=10))
            assert state['preferences'] == {'enabled': False, 'text': ''}
            assert Path(state['configDir']) == tmp / 'config/magiko'
            anon = urllib.request.build_opener(urllib.request.ProxyHandler({}))
            try:
                anon.open(base + '/api/magiko', timeout=10)
                raise AssertionError('Unauthenticated API access allowed')
            except urllib.error.HTTPError as error:
                assert error.code in (401, 403), error.code
            print(json.dumps({'app': 'Magiko', 'embeddedUI': 'ok', 'preferencesAPI': 'ok',
                              'webAuthentication': 'ok', 'isolatedHome': True}))
        finally:
            if proc.poll() is None:
                os.killpg(proc.pid, signal.SIGTERM)
            try:
                proc.wait(timeout=10)
            except subprocess.TimeoutExpired:
                os.killpg(proc.pid, signal.SIGKILL)
                proc.wait(timeout=10)
