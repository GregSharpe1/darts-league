const { DeleteMessageBatchCommand, ReceiveMessageCommand, SQSClient } = require('@aws-sdk/client-sqs');

const MAX_BODY = 256 * 1024;
const MAX_RESPONSE = 2 * 1024 * 1024;
const MAX_ACK = 32 * 1024;
const validString = (value, max) => typeof value === 'string' && value.trim().length > 0 && value.length <= max;

function reply(statusCode, payload) {
  return {
    statusCode,
    headers: { 'Content-Type': 'application/json', 'Cache-Control': 'no-store' },
    body: JSON.stringify(payload),
  };
}

function createHandler(client) {
  return async (event) => {
    const method = event.requestContext?.http?.method;
    const QueueUrl = process.env.QUEUE_URL;
    const options = { abortSignal: AbortSignal.timeout(8000) };
    try {
      if (method === 'GET') {
        const response = await client.send(new ReceiveMessageCommand({
          QueueUrl, MaxNumberOfMessages: 10, WaitTimeSeconds: 0,
          MessageSystemAttributeNames: ['SentTimestamp', 'ApproximateReceiveCount'],
        }), options);
        let bytes = 32;
        let oldestAgeMs = 0;
        const messages = (response.Messages ?? []).map(message => {
          const entry = {
            messageId: message.MessageId, receiptHandle: message.ReceiptHandle,
            body: message.Body ?? '', attributes: message.Attributes ?? {},
          };
          if (Buffer.byteLength(entry.body) > MAX_BODY) {
            entry.body = '';
            entry.rejection = 'payload_too_large';
          }
          // Reserve space for all remaining receipt-only entries.
          if (bytes + Buffer.byteLength(JSON.stringify(entry)) > MAX_RESPONSE - MAX_ACK) {
            entry.body = '';
            entry.rejection = 'response_limit';
          }
          bytes += Buffer.byteLength(JSON.stringify(entry)) + 1;
          const sent = Number(entry.attributes.SentTimestamp);
          if (Number.isFinite(sent) && sent > 0) oldestAgeMs = Math.max(oldestAgeMs, Date.now() - sent);
          return entry;
        });
        console.info(JSON.stringify({ event: 'relay.receive', received: messages.length,
          deferred: messages.filter(m => m.rejection).length, oldestAgeMs }));
        return reply(200, { messages });
      }
      if (method !== 'POST') return reply(405, { error: 'method_not_allowed' });
      if (typeof event.body !== 'string') return reply(400, { error: 'invalid_ack' });
      if (Buffer.byteLength(event.body) > MAX_ACK * (event.isBase64Encoded ? 2 : 1)) {
        return reply(413, { error: 'ack_too_large' });
      }
      const body = event.isBase64Encoded ? Buffer.from(event.body, 'base64').toString('utf8') : event.body;
      if (Buffer.byteLength(body) > MAX_ACK) return reply(413, { error: 'ack_too_large' });
      let payload;
      try { payload = JSON.parse(body); } catch { return reply(400, { error: 'invalid_ack' }); }
      const messages = payload?.messages;
      if (!Array.isArray(messages) || messages.length < 1 || messages.length > 10 ||
        messages.some(m => !m || !validString(m.messageId, 128) || !validString(m.receiptHandle, 2048)) ||
        new Set(messages.map(m => m.messageId)).size !== messages.length ||
        new Set(messages.map(m => m.receiptHandle)).size !== messages.length) {
        return reply(400, { error: 'invalid_ack' });
      }
      const result = await client.send(new DeleteMessageBatchCommand({ QueueUrl,
        Entries: messages.map((m, i) => ({ Id: String(i), ReceiptHandle: m.receiptHandle })),
      }), options);
      const successful = new Set((result.Successful ?? []).map(m => m.Id));
      const failures = new Set((result.Failed ?? []).map(m => m.Id));
      const acknowledged = [];
      const failed = [];
      messages.forEach((m, i) => {
        (successful.has(String(i)) && !failures.has(String(i)) ? acknowledged : failed).push(m.messageId);
      });
      console.info(JSON.stringify({ event: 'relay.ack', acknowledged: acknowledged.length, failed: failed.length }));
      return reply(failed.length ? 503 : 200, { acknowledged, failed });
    } catch {
      console.error(JSON.stringify({ event: 'relay.sqs_failure', operation: method === 'GET' ? 'receive' : 'ack' }));
      return reply(503, { error: 'queue_unavailable' });
    }
  };
}

exports.createHandler = createHandler;
exports.handler = createHandler(new SQSClient({ maxAttempts: 2 }));
