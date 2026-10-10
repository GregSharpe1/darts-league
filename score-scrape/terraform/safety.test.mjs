import { existsSync, readFileSync } from 'node:fs';
import assert from 'node:assert/strict';
import { test } from 'node:test';

const read = name => readFileSync(new URL(name, import.meta.url), 'utf8');
const main = read('main.tf');
const block = (type, name) => {
  const start = `resource "${type}" "${name}" {\n`;
  assert.ok(main.includes(start), start);
  return main.split(start)[1].split('\n}\n')[0];
};
const compact = text => text.replace(/\s+/g, ' ').trim();
const contains = (text, expected) => assert.ok(compact(text).includes(compact(expected)), expected);

test('Terraform replaces the obsolete SAM deployment entrypoints', () => {
  for (const name of ['../template.yaml', '../samconfig.toml']) {
    assert.equal(existsSync(new URL(name, import.meta.url)), false, `${name} must not remain a second deployment entrypoint`);
  }
});

test('queue limits, runtime, CORS and alarm defaults preserve the delivery contract', () => {
  const queue = block('aws_sqs_queue', 'results');
  for (const setting of ['visibility_timeout_seconds = 180', 'max_message_size = 262144', 'message_retention_seconds = 345600', 'sqs_managed_sse_enabled = true']) contains(queue, setting);
  contains(block('aws_sqs_queue', 'dead_letter'), 'message_retention_seconds = 1209600');
  contains(block('aws_sqs_queue', 'dead_letter'), 'sqs_managed_sse_enabled = true');
  contains(block('aws_lambda_function', 'reader'), 'runtime = "nodejs22.x"');
  contains(block('aws_lambda_function', 'reader'), 'timeout = 10');
  contains(read('variables.tf'), 'default = "https://play.autodarts.com"');
  contains(block('aws_apigatewayv2_api', 'results'), 'allow_origins = [var.allowed_origin]');
  contains(block('aws_apigatewayv2_stage', 'default'), 'throttling_burst_limit = 5 throttling_rate_limit = 2');
  const alarms = block('aws_cloudwatch_metric_alarm', 'queue');
  for (const setting of ['metric = "ApproximateNumberOfMessagesVisible"', 'metric = "ApproximateAgeOfOldestMessage"', 'limit = 259200', 'period = 300', 'evaluation_periods = 1', 'treat_missing_data = "notBreaching"']) contains(alarms, setting);
});

// Plan-time computed ARNs are unknown on Terraform 1.7-1.9; check their exact
// source bindings locally rather than weakening prevent_destroy for mock teardown.
test('queue protection, redrive and reader environment bind to intended resources', () => {
  for (const name of ['results', 'dead_letter']) {
    contains(block('aws_sqs_queue', name), 'lifecycle { prevent_destroy = true }');
  }
  contains(block('aws_sqs_queue', 'results'), 'deadLetterTargetArn = aws_sqs_queue.dead_letter.arn maxReceiveCount = 5');
  contains(block('aws_lambda_function', 'reader'), 'variables = { QUEUE_URL = aws_sqs_queue.results.url }');
  contains(block('aws_lambda_function', 'reader'), 'depends_on = [aws_iam_role_policy.reader]');
});

test('IAM actions and resources stay least privilege', () => {
  const send = block('aws_iam_role_policy', 'submission');
  const reader = block('aws_iam_role_policy', 'reader');
  assert.deepEqual([...send.matchAll(/"((?:sqs|logs):[^"]+)"/g)].map(m => m[1]), ['sqs:SendMessage']);
  assert.deepEqual([...reader.matchAll(/"((?:sqs|logs):[^"]+)"/g)].map(m => m[1]),
    ['sqs:ReceiveMessage', 'sqs:DeleteMessage', 'logs:CreateLogStream', 'logs:PutLogEvents']);
  contains(send, 'Resource = aws_sqs_queue.results.arn');
  contains(reader, 'Resource = aws_sqs_queue.results.arn');
  contains(reader, 'Resource = "${aws_cloudwatch_log_group.reader.arn}:*"');
  assert.equal((main.match(/resource "aws_iam_role_policy"/g) ?? []).length, 2);
  assert.doesNotMatch(main, /aws_iam_(?:policy_attachment|role_policy_attachment)|Action\s*=\s*"\*"|Resource\s*=\s*"\*"/);
  contains(block('aws_iam_role', 'reader'), 'Principal = { Service = "lambda.amazonaws.com" }');
  contains(block('aws_iam_role', 'submission'), 'Principal = { Service = "apigateway.amazonaws.com" }');
});

test('routes, integrations and scoped invoke permissions use paired reader', () => {
  const routes = block('aws_apigatewayv2_route', 'results');
  contains(routes, '"POST /results" = aws_apigatewayv2_integration.submission.id');
  contains(routes, '"GET /results" = aws_apigatewayv2_integration.reader.id');
  contains(routes, '"POST /results/ack" = aws_apigatewayv2_integration.reader.id');
  contains(routes, 'target = "integrations/${each.value}"');
  const submission = block('aws_apigatewayv2_integration', 'submission');
  contains(submission, 'QueueUrl = aws_sqs_queue.results.url');
  contains(submission, 'credentials_arn = aws_iam_role.submission.arn');
  contains(block('aws_apigatewayv2_integration', 'reader'), 'integration_uri = aws_lambda_function.reader.invoke_arn');
  const permission = block('aws_lambda_permission', 'reader');
  contains(permission, 'Read = "GET/results" Ack = "POST/results/ack"');
  contains(permission, 'source_arn = "${aws_apigatewayv2_api.results.execution_arn}/*/${each.value}"');
  contains(permission, 'function_name = aws_lambda_function.reader.function_name');
});

test('required outputs expose new resources, never a historical API', () => {
  const outputs = read('outputs.tf');
  const expected = {
    submission_url: '"${aws_apigatewayv2_api.results.api_endpoint}/results"',
    read_url: '"${aws_apigatewayv2_api.results.api_endpoint}/results"',
    acknowledgement_url: '"${aws_apigatewayv2_api.results.api_endpoint}/results/ack"',
    queue_url: 'aws_sqs_queue.results.url', queue_arn: 'aws_sqs_queue.results.arn',
    dead_letter_queue_url: 'aws_sqs_queue.dead_letter.url',
  };
  for (const [name, value] of Object.entries(expected)) {
    contains(outputs, `output "${name}" { value = ${value} }`);
  }
  assert.doesNotMatch(main, /local-exec|remote-exec|data\s+"aws_|profile\s*=/);
});
