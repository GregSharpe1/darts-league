const assert = require('node:assert/strict');
const { test } = require('node:test');
const { createHandler } = require('./index');
process.env.QUEUE_URL = 'https://sqs.example.test/results';

function setup(response) {
  const calls = [];
  const handler = createHandler({ send: async (command, options) => {
    calls.push(command);
    assert.ok(options.abortSignal);
    assert.equal(command.input.QueueUrl, process.env.QUEUE_URL);
    if (response instanceof Error) throw response;
    return response;
  } });
  return { handler, calls };
}
const event = (method, body) => ({ requestContext: { http: { method } }, body });
const message = { MessageId: 'id-1', ReceiptHandle: 'receipt-1', Body: '{bad json' };

test('GET returns identity and raw malformed body without deleting', async () => {
  const { handler, calls } = setup({ Messages: [message] });
  const result = await handler(event('GET'));
  assert.equal(result.statusCode, 200);
  assert.deepEqual(JSON.parse(result.body).messages[0], {
    messageId: 'id-1', receiptHandle: 'receipt-1', body: '{bad json', attributes: {},
  });
  assert.deepEqual(calls.map(c => c.constructor.name), ['ReceiveMessageCommand']);
  assert.equal(calls[0].input.MaxNumberOfMessages, 10);
  assert.equal(result.headers['Cache-Control'], 'no-store');
  assert.notEqual(result.headers['Access-Control-Allow-Origin'], '*');
});

test('POST acknowledges only the explicitly supplied receipt', async () => {
  const { handler, calls } = setup({ Successful: [{ Id: '0' }], Failed: [] });
  const result = await handler(event('POST', JSON.stringify({ messages: [{ messageId: 'id-1', receiptHandle: 'receipt-1' }] })));
  assert.equal(result.statusCode, 200);
  assert.deepEqual(JSON.parse(result.body), { acknowledged: ['id-1'], failed: [] });
  assert.equal(calls[0].constructor.name, 'DeleteMessageBatchCommand');
  assert.deepEqual(calls[0].input.Entries, [{ Id: '0', ReceiptHandle: 'receipt-1' }]);
});

test('partial SQS deletion is an HTTP failure with per-message outcomes', async () => {
  const { handler } = setup({ Successful: [{ Id: '0' }], Failed: [{ Id: '1' }] });
  const messages = [1, 2].map(n => ({ messageId: `id-${n}`, receiptHandle: `receipt-${n}` }));
  const result = await handler(event('POST', JSON.stringify({ messages })));
  assert.equal(result.statusCode, 503);
  assert.deepEqual(JSON.parse(result.body), { acknowledged: ['id-1'], failed: ['id-2'] });
});

test('invalid and oversized acknowledgement requests never reach SQS', async () => {
  for (const body of ['{', '{}', '{"messages":[]}', JSON.stringify({ messages: Array(11).fill({}) }),
    JSON.stringify({ messages: [{ messageId: 'id' }] }), ' '.repeat(32769),
    JSON.stringify({ messages: Array(2).fill({ messageId: 'id', receiptHandle: 'receipt' }) })]) {
    const { handler, calls } = setup({});
    const result = await handler(event('POST', body));
    assert.ok([400, 413].includes(result.statusCode));
    assert.equal(calls.length, 0);
  }
});

test('oversized bodies remain unacknowledged and response size stays bounded', async () => {
  const Messages = Array.from({ length: 10 }, (_, n) => ({ ...message, MessageId: `id-${n}`, Body: '\u0000'.repeat(262144) }));
  Messages[0].Body = 'x'.repeat(262145);
  const { handler, calls } = setup({ Messages });
  const result = await handler(event('GET'));
  assert.ok(Buffer.byteLength(result.body) <= 2 * 1024 * 1024);
  assert.ok(Buffer.byteLength(JSON.stringify(result)) < 6 * 1024 * 1024);
  const received = JSON.parse(result.body).messages;
  assert.equal(received.length, 10);
  assert.equal(received[0].rejection, 'payload_too_large');
  assert.ok(received.some(m => m.rejection === 'response_limit'));
  assert.equal(calls.length, 1);
});

test('SQS failures are signalled without returning error details', async () => {
  const { handler } = setup(new Error('secret receipt payload'));
  const result = await handler(event('GET'));
  assert.equal(result.statusCode, 503);
  assert.ok(!result.body.includes('secret'));
});

test('base64 acknowledgement is decoded and unsupported methods do not receive', async () => {
  const { handler, calls } = setup({ Successful: [{ Id: '0' }] });
  const body = Buffer.from(JSON.stringify({ messages: [{ messageId: 'id', receiptHandle: 'receipt' }] })).toString('base64');
  assert.equal((await handler({ ...event('POST', body), isBase64Encoded: true })).statusCode, 200);
  assert.equal((await handler(event('DELETE'))).statusCode, 405);
  assert.equal(calls.length, 1);
});

test('missing SQS acknowledgement outcomes fail closed', async () => {
  const { handler } = setup({});
  const result = await handler(event('POST', JSON.stringify({ messages: [{ messageId: 'id', receiptHandle: 'receipt' }] })));
  assert.equal(result.statusCode, 503);
  assert.deepEqual(JSON.parse(result.body), { acknowledged: [], failed: ['id'] });
});
