"""Exercise the actual deployment archive and inline trigger without AWS access."""
import http.server
import json
import os
from pathlib import Path
import platform
import queue
import stat
import struct
import subprocess
import sys
import tempfile
import textwrap
import threading
import types
import unittest
from unittest.mock import Mock, patch
import zipfile

ROOT = Path(__file__).resolve().parents[2]


class LambdaPackageTests(unittest.TestCase):
    def test_archives_are_executable_static_linux_with_matching_architectures(self):
        for architecture, machine in [('x86_64', 62), ('arm64', 183)]:
            with self.subTest(architecture=architecture):
                with zipfile.ZipFile(ROOT / f'dist/lambda-{architecture}.zip') as archive:
                    self.assertEqual(archive.namelist(), ['bootstrap'])
                    info = archive.getinfo('bootstrap')
                    self.assertTrue((info.external_attr >> 16) & stat.S_IXUSR)
                    data = archive.read('bootstrap')
                self.assertEqual(data[:6], b'\x7fELF\x02\x01')
                self.assertEqual(struct.unpack_from('<H', data, 18)[0], machine)
                # A CGO-free executable must have no dynamic interpreter (PT_INTERP).
                offset = struct.unpack_from('<Q', data, 32)[0]
                size, count = struct.unpack_from('<HH', data, 54)
                self.assertNotIn(3, [struct.unpack_from('<I', data, offset + i * size)[0]
                                     for i in range(count)])

    def test_unqualified_archive_preserves_x86_64_alias(self):
        self.assertEqual((ROOT / 'dist/lambda.zip').read_bytes(),
                         (ROOT / 'dist/lambda-x86_64.zip').read_bytes())

    def test_invalid_architecture_is_rejected_before_creating_output(self):
        with tempfile.TemporaryDirectory() as temp:
            output = Path(temp) / 'invalid.zip'
            result = subprocess.run([str(ROOT / 'scripts/package-lambda.sh'), str(output), 'invalid'],
                                    capture_output=True, text=True, check=False)
            self.assertEqual(result.returncode, 2)
            self.assertIn('Unsupported Lambda architecture', result.stderr)
            self.assertFalse(output.exists())

    def test_bootstrap_reports_handler_errors_through_runtime_api(self):
        received = queue.Queue()

        class Runtime(http.server.BaseHTTPRequestHandler):
            def log_message(self, *args):
                pass

            def do_GET(self):
                self.send_response(200)
                self.send_header('Lambda-Runtime-Aws-Request-Id', 'smoke-request')
                self.send_header('Lambda-Runtime-Deadline-Ms', '4102444800000')
                self.end_headers()
                self.wfile.write(b'{"detail":{"stack-id":"test"}}')

            def do_POST(self):
                # aws-lambda-go streams error payloads with chunked encoding.
                if self.headers.get('Transfer-Encoding') == 'chunked':
                    chunks = []
                    while True:
                        size = int(self.rfile.readline().split(b';', 1)[0], 16)
                        if size == 0:
                            while self.rfile.readline().strip():
                                pass  # Consume optional trailers through the empty line.
                            break
                        chunks.append(self.rfile.read(size))
                        self.rfile.read(2)
                    body = b''.join(chunks)
                else:
                    body = self.rfile.read(int(self.headers['Content-Length']))
                received.put((self.path, json.loads(body)))
                self.send_response(202)
                self.end_headers()

        with tempfile.TemporaryDirectory() as temp:
            binary = Path(temp) / 'bootstrap'
            architecture = {'x86_64': 'x86_64', 'AMD64': 'x86_64',
                            'aarch64': 'arm64', 'arm64': 'arm64'}.get(platform.machine())
            if platform.system() == 'Linux' and architecture:
                with zipfile.ZipFile(ROOT / f'dist/lambda-{architecture}.zip') as archive:
                    binary.write_bytes(archive.read('bootstrap'))
                binary.chmod(0o755)
            else:
                # macOS cannot execute the Linux archive. CI runs the exact archive.
                subprocess.run(['go', 'build', '-tags', 'lambda.norpc', '-o', str(binary), '.'],
                               cwd=ROOT, check=True)
            with http.server.ThreadingHTTPServer(('127.0.0.1', 0), Runtime) as server:
                thread = threading.Thread(target=server.serve_forever, daemon=True)
                thread.start()
                env = os.environ.copy()
                for key in ('ReportS3Bucket', 'ReportOutputFormat'):
                    env.pop(key, None)
                env.update(AWS_LAMBDA_FUNCTION_NAME='fog-smoke',
                           AWS_LAMBDA_RUNTIME_API=f'127.0.0.1:{server.server_port}')
                with open(os.devnull, 'w') as output:
                    process = subprocess.Popen([str(binary)], env=env, stdout=output, stderr=output)
                    try:
                        path, body = received.get(timeout=20)
                        self.assertEqual(path, '/2018-06-01/runtime/invocation/smoke-request/error')
                        self.assertIn('ReportS3Bucket is not set', body['errorMessage'])
                    finally:
                        process.terminate()
                        process.wait(timeout=5)
                        server.shutdown()
                thread.join(timeout=5)


class InlineTriggerTests(unittest.TestCase):
    def setUp(self):
        template = (ROOT / 'examples/templates/fogreport-builder.yaml').read_text()
        source = textwrap.dedent(template.split('Fn::Sub: |\n', 1)[1].split('\n      Handler:', 1)[0])
        source = source.replace('${AmplifyApp.AppId}', 'test-app')
        self.client = Mock()
        module = types.ModuleType('boto3')
        module.client = Mock(return_value=self.client)
        self.module = module
        self.namespace = {}
        with patch.dict(sys.modules, boto3=module):
            exec(compile(source, 'inline-trigger', 'exec'), self.namespace)

    def test_requests_main_release(self):
        self.namespace['lambda_handler']({'Records': []}, None)
        self.module.client.assert_called_once_with('amplify')
        self.client.start_job.assert_called_once_with(appId='test-app', branchName='main', jobType='RELEASE')

    def test_service_failure_propagates(self):
        failure = RuntimeError('Amplify unavailable')
        self.client.start_job.side_effect = failure
        with self.assertRaises(RuntimeError) as caught:
            self.namespace['lambda_handler']({}, None)
        self.assertIs(caught.exception, failure)
