"""Verify standalone migration preserves resource identity, package and site binding."""
import copy
import importlib.util
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]
SCRIPT = ROOT / 'scripts/prepare-builder-runtime-template.py'
SPEC = importlib.util.spec_from_file_location('builder_migration', SCRIPT)
migration = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(migration)


def deployed_template(processed=False):
    properties = {
        'Runtime': 'python3.9', 'Handler': 'app.lambda_handler',
        'Architectures': ['x86_64'],
    }
    if processed:
        properties.update(Code={'S3Bucket': 'existing-source', 'S3Key': 'immutable-package'},
                          Role={'Fn::GetAtt': ['HelloWorldFunctionRole', 'Arn']}, Timeout=3)
    else:
        properties.update(CodeUri='s3://existing-source/immutable-package',
                          Policies=[{'Statement': [{'Effect': 'Allow', 'Action': ['amplify:*'],
                                                    'Resource': '*'}]}])
    template = {
        'AWSTemplateFormatVersion': '2010-09-09',
        'Description': 'Lambda function for running Amplify job',
        'Resources': {'HelloWorldFunction': {
            'Type': 'AWS::Lambda::Function' if processed else 'AWS::Serverless::Function',
            'Properties': properties,
        }},
    }
    if processed:
        template['Resources']['HelloWorldFunctionRole'] = {
            'Type': 'AWS::IAM::Role', 'Properties': {'Policies': [{'PolicyName': 'existing'}]},
        }
    else:
        template['Transform'] = 'AWS::Serverless-2016-10-31'
    return template


class BuilderMigrationTests(unittest.TestCase):
    def test_original_and_processed_preserve_everything_except_runtime(self):
        for processed in (False, True):
            with self.subTest(processed=processed):
                original = deployed_template(processed)
                before = copy.deepcopy(original)
                candidate = migration.prepare(original)
                self.assertEqual(original, before)
                self.assertEqual(candidate['Resources']['HelloWorldFunction']['Properties']['Runtime'],
                                 'python3.14')
                candidate['Resources']['HelloWorldFunction']['Properties']['Runtime'] = 'python3.9'
                self.assertEqual(candidate, before)

    def test_accepts_get_template_json_body_and_is_idempotent(self):
        for body in (deployed_template(), json.dumps(deployed_template())):
            candidate = migration.prepare({'TemplateBody': body, 'StagesAvailable': ['Original']})
            self.assertEqual(migration.prepare(candidate), candidate)

    def test_refuses_wrong_resource_handler_runtime_or_missing_package(self):
        for field, value in [('Type', 'AWS::S3::Bucket'), ('Handler', 'index.lambda_handler'),
                             ('Runtime', 'python3.12'), ('CodeUri', '')]:
            with self.subTest(field=field):
                template = deployed_template()
                function = template['Resources']['HelloWorldFunction']
                target = function if field == 'Type' else function['Properties']
                target[field] = value
                with self.assertRaises(ValueError):
                    migration.prepare(template)
        with self.assertRaises(ValueError):
            migration.prepare({'Resources': {'TriggerFunction': {}}})
        with self.assertRaises(ValueError):
            migration.prepare({'TemplateBody': 'Resources: {}'})

    def test_cli_writes_candidate_but_never_overwrites_files(self):
        with tempfile.TemporaryDirectory() as temp:
            source, output = Path(temp) / 'deployed.json', Path(temp) / 'candidate.json'
            source.write_text(json.dumps(deployed_template(True)))
            before = source.read_bytes()
            command = [sys.executable, str(SCRIPT), str(source), str(output)]
            result = subprocess.run(command, capture_output=True, text=True)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertIn('compatibility is unverified', result.stderr)
            candidate = output.read_bytes()
            result = subprocess.run(command, capture_output=True, text=True)
            self.assertEqual(result.returncode, 2)
            self.assertEqual(output.read_bytes(), candidate)
            self.assertEqual(source.read_bytes(), before)
            result = subprocess.run([*command[:-1], str(source)], capture_output=True, text=True)
            self.assertEqual(result.returncode, 2)
            self.assertEqual(source.read_bytes(), before)


if __name__ == '__main__':
    unittest.main()
