terraform {
  required_version = ">= 1.7, < 2.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.region
}

resource "aws_sqs_queue" "dead_letter" {
  name                      = "${var.name_prefix}-dead-letter"
  message_retention_seconds = 1209600
  sqs_managed_sse_enabled   = true
  lifecycle {
    prevent_destroy = true
  }
}

resource "aws_sqs_queue" "results" {
  name                       = "${var.name_prefix}-results"
  visibility_timeout_seconds = 180
  max_message_size           = 262144
  message_retention_seconds  = 345600
  sqs_managed_sse_enabled    = true
  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.dead_letter.arn
    maxReceiveCount     = 5
  })
  lifecycle {
    prevent_destroy = true
  }
}

resource "aws_cloudwatch_metric_alarm" "queue" {
  for_each = {
    dead_letter = {
      metric = "ApproximateNumberOfMessagesVisible"
      queue  = aws_sqs_queue.dead_letter.name
      limit  = 0
    }
    source_age = {
      metric = "ApproximateAgeOfOldestMessage"
      queue  = aws_sqs_queue.results.name
      limit  = 259200
    }
  }
  alarm_name          = "${var.name_prefix}-${each.key}"
  namespace           = "AWS/SQS"
  metric_name         = each.value.metric
  dimensions          = { QueueName = each.value.queue }
  statistic           = "Maximum"
  period              = 300
  evaluation_periods  = 1
  threshold           = each.value.limit
  comparison_operator = "GreaterThanThreshold"
  treat_missing_data  = "notBreaching"
  alarm_actions       = var.alarm_action_arns
}

resource "aws_iam_role" "submission" {
  name = "${var.name_prefix}-submission"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow", Action = "sts:AssumeRole"
      Principal = { Service = "apigateway.amazonaws.com" }
    }]
  })
}

resource "aws_iam_role_policy" "submission" {
  name = "SendResults"
  role = aws_iam_role.submission.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow", Action = ["sqs:SendMessage"], Resource = aws_sqs_queue.results.arn
    }]
  })
}

resource "aws_iam_role" "reader" {
  name = "${var.name_prefix}-reader"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow", Action = "sts:AssumeRole"
      Principal = { Service = "lambda.amazonaws.com" }
    }]
  })
}

resource "aws_cloudwatch_log_group" "reader" {
  name              = "/aws/lambda/${var.name_prefix}-reader"
  retention_in_days = var.log_retention_days
}

resource "aws_iam_role_policy" "reader" {
  name = "ReceiveAcknowledgeAndLog"
  role = aws_iam_role.reader.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow", Action = ["sqs:ReceiveMessage", "sqs:DeleteMessage"]
        Resource = aws_sqs_queue.results.arn
      },
      {
        Effect   = "Allow", Action = ["logs:CreateLogStream", "logs:PutLogEvents"]
        Resource = "${aws_cloudwatch_log_group.reader.arn}:*"
      }
    ]
  })
}

resource "aws_lambda_function" "reader" {
  function_name    = "${var.name_prefix}-reader"
  role             = aws_iam_role.reader.arn
  runtime          = "nodejs22.x"
  handler          = "index.handler"
  timeout          = 10
  filename         = "${path.module}/build/reader.zip"
  source_code_hash = filebase64sha256("${path.module}/build/reader.zip")
  environment {
    variables = { QUEUE_URL = aws_sqs_queue.results.url }
  }
  depends_on = [aws_iam_role_policy.reader]
}

resource "aws_apigatewayv2_api" "results" {
  name          = var.name_prefix
  protocol_type = "HTTP"
  cors_configuration {
    allow_origins = [var.allowed_origin]
    allow_methods = ["GET", "POST"]
    allow_headers = ["content-type"]
    max_age       = 3600
  }
}

resource "aws_apigatewayv2_integration" "submission" {
  api_id                 = aws_apigatewayv2_api.results.id
  integration_type       = "AWS_PROXY"
  integration_subtype    = "SQS-SendMessage"
  credentials_arn        = aws_iam_role.submission.arn
  payload_format_version = "1.0"
  timeout_milliseconds   = 10000
  request_parameters = {
    QueueUrl    = aws_sqs_queue.results.url
    MessageBody = "$request.body"
  }
  depends_on = [aws_iam_role_policy.submission]
}

resource "aws_apigatewayv2_integration" "reader" {
  api_id                 = aws_apigatewayv2_api.results.id
  integration_type       = "AWS_PROXY"
  integration_uri        = aws_lambda_function.reader.invoke_arn
  payload_format_version = "2.0"
  timeout_milliseconds   = 10000
}

resource "aws_apigatewayv2_route" "results" {
  for_each = {
    "POST /results"     = aws_apigatewayv2_integration.submission.id
    "GET /results"      = aws_apigatewayv2_integration.reader.id
    "POST /results/ack" = aws_apigatewayv2_integration.reader.id
  }
  api_id             = aws_apigatewayv2_api.results.id
  route_key          = each.key
  authorization_type = "NONE"
  target             = "integrations/${each.value}"
}

resource "aws_lambda_permission" "reader" {
  for_each = {
    Read = "GET/results"
    Ack  = "POST/results/ack"
  }
  statement_id  = each.key
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.reader.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.results.execution_arn}/*/${each.value}"
}

resource "aws_apigatewayv2_stage" "default" {
  api_id      = aws_apigatewayv2_api.results.id
  name        = "$default"
  auto_deploy = true
  default_route_settings {
    throttling_burst_limit = 5
    throttling_rate_limit  = 2
  }
}
