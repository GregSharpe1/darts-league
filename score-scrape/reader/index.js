const {
  DeleteMessageCommand,
  ReceiveMessageCommand,
  SQSClient,
} = require('@aws-sdk/client-sqs');

const client = new SQSClient({});

exports.handler = async () => {
  const queueUrl = process.env.QUEUE_URL;
  const response = await client.send(new ReceiveMessageCommand({
    QueueUrl: queueUrl,
    MaxNumberOfMessages: 10,
    WaitTimeSeconds: 0,
    MessageAttributeNames: ['All'],
  }));

  const messages = response.Messages ?? [];

  await Promise.all(messages.map((message) => client.send(new DeleteMessageCommand({
    QueueUrl: queueUrl,
    ReceiptHandle: message.ReceiptHandle,
  }))));

  return {
    statusCode: 200,
    headers: {
      'Content-Type': 'application/json',
      'Access-Control-Allow-Origin': '*',
    },
    body: JSON.stringify({
      messages: messages.map((message) => ({
        body: message.Body ?? '',
        messageId: message.MessageId ?? null,
        attributes: message.Attributes ?? {},
        messageAttributes: message.MessageAttributes ?? {},
      })),
    }),
  };
};