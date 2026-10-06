const { readFileSync } = require('node:fs');
const assert = require('node:assert/strict');
const { test } = require('node:test');
const template = readFileSync(`${__dirname}/../template.yaml`, 'utf8');
const resource = name => template.split(`  ${name}:\n`)[1]?.split(/\n  \w+:\n/)[0] ?? '';

test('queue has bounded retries into encrypted retained DLQ', () => {
  assert.match(resource('ResultsQueue'), /VisibilityTimeout: 180/);
  assert.match(resource('ResultsQueue'), /MaximumMessageSize: 262144/);
  assert.match(resource('ResultsQueue'), /deadLetterTargetArn: !GetAtt ResultsDeadLetterQueue.Arn/);
  assert.match(resource('ResultsQueue'), /maxReceiveCount: 5/);
  assert.match(resource('ResultsDeadLetterQueue'), /MessageRetentionPeriod: 1209600/);
  assert.match(resource('ResultsDeadLetterQueue'), /SqsManagedSseEnabled: true/);
  assert.match(resource('ResultsDeadLetterQueue'), /DeletionPolicy: Retain/);
});

test('ack is a separate route with scoped permission and actual origin', () => {
  assert.match(template, /Default: https:\/\/play.autodarts.com/);
  assert.match(resource('ResultsAckRoute'), /RouteKey: POST \/results\/ack/);
  assert.match(resource('ResultsAckPermission'), /\*\/POST\/results\/ack/);
  assert.match(resource('ResultsReaderIntegration'), /TimeoutInMillis: 10000/);
  assert.match(resource('ResultsStage'), /ThrottlingRateLimit: 2/);
});

test('queue age and quarantine have observable alarms', () => {
  assert.match(resource('ResultsDeadLetterAlarm'), /MetricName: ApproximateNumberOfMessagesVisible/);
  assert.match(resource('ResultsQueueAgeAlarm'), /MetricName: ApproximateAgeOfOldestMessage/);
});
