#!/usr/bin/env python3
"""Prepare a runtime-only update for Fog's older standalone builder; never call AWS."""
import argparse
import copy
import json
from pathlib import Path
import sys


RUNTIME = 'python3.14'
LOGICAL_ID = 'HelloWorldFunction'


def prepare(template):
    """Preserve the deployed template except for the known standalone runtime."""
    if not isinstance(template, dict):
        raise ValueError('Expected a CloudFormation template JSON object')
    # Accept the JSON response from GetTemplate as well as its TemplateBody.
    if 'TemplateBody' in template:
        template = template['TemplateBody']
        if isinstance(template, str):
            try:
                template = json.loads(template)
            except json.JSONDecodeError as error:
                raise ValueError('TemplateBody must be JSON; export the Processed template') from error
    if not isinstance(template, dict):
        raise ValueError('TemplateBody must be a JSON object')
    resources = template.get('Resources', {})
    function = resources.get(LOGICAL_ID, {})
    properties = function.get('Properties', {})
    resource_type = function.get('Type')
    if resource_type not in ('AWS::Lambda::Function', 'AWS::Serverless::Function'):
        raise ValueError('Expected the existing HelloWorldFunction Lambda resource')
    if properties.get('Handler') != 'app.lambda_handler':
        raise ValueError('Expected the existing app.lambda_handler; refusing a different builder')
    if properties.get('Runtime') not in ('python3.9', RUNTIME):
        raise ValueError('Expected the inventoried python3.9 runtime or an already prepared template')
    code_key = 'Code' if resource_type == 'AWS::Lambda::Function' else 'CodeUri'
    if not properties.get(code_key) or 'InlineCode' in properties:
        raise ValueError('Expected the existing packaged code locator; refusing replacement code')
    candidate = copy.deepcopy(template)
    candidate['Resources'][LOGICAL_ID]['Properties']['Runtime'] = RUNTIME
    return candidate


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('input', type=Path, help='Existing deployed template or GetTemplate JSON response')
    parser.add_argument('output', type=Path, help='New candidate JSON file; must not already exist')
    args = parser.parse_args()
    try:
        candidate = prepare(json.loads(args.input.read_text()))
        # Never overwrite an export or an earlier review candidate.
        with args.output.open('x') as output:
            output.write(json.dumps(candidate, indent=2) + '\n')
    except (OSError, ValueError, TypeError, AttributeError) as error:
        parser.exit(2, f'{error}\n')
    print(f'Prepared {args.output}: only {LOGICAL_ID}.Properties.Runtime -> {RUNTIME}')
    print('No AWS call made. Existing packaged Python compatibility is unverified.', file=sys.stderr)


if __name__ == '__main__':
    main()
