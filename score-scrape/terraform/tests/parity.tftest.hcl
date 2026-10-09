mock_provider "aws" {}

variables {
  region      = "eu-west-1"
  name_prefix = "offline-relay"
}

run "delivery_contract" {
  command = plan

  assert {
    condition = (
      aws_sqs_queue.results.visibility_timeout_seconds == 180 &&
      aws_sqs_queue.results.max_message_size == 262144 &&
      aws_sqs_queue.results.message_retention_seconds == 345600 &&
      aws_sqs_queue.results.sqs_managed_sse_enabled &&
      aws_sqs_queue.dead_letter.message_retention_seconds == 1209600 &&
      aws_sqs_queue.dead_letter.sqs_managed_sse_enabled
    )
    error_message = "Queue retention, encryption or visibility violates the delivery contract."
  }
  assert {
    condition = (
      aws_apigatewayv2_api.results.protocol_type == "HTTP" &&
      aws_apigatewayv2_api.results.cors_configuration[0].allow_origins == toset(["https://play.autodarts.com"]) &&
      aws_apigatewayv2_api.results.cors_configuration[0].allow_methods == toset(["GET", "POST"]) &&
      aws_apigatewayv2_api.results.cors_configuration[0].allow_headers == toset(["content-type"]) &&
      aws_apigatewayv2_api.results.cors_configuration[0].max_age == 3600 &&
      aws_apigatewayv2_stage.default.name == "$default" &&
      aws_apigatewayv2_stage.default.auto_deploy &&
      aws_apigatewayv2_stage.default.default_route_settings[0].throttling_burst_limit == 5 &&
      aws_apigatewayv2_stage.default.default_route_settings[0].throttling_rate_limit == 2
    )
    error_message = "HTTP API CORS or stage throttling violates the delivery contract."
  }
  assert {
    condition = (
      toset(keys(aws_apigatewayv2_route.results)) == toset(["POST /results", "GET /results", "POST /results/ack"]) &&
      alltrue([for key, route in aws_apigatewayv2_route.results : route.route_key == key && route.authorization_type == "NONE"]) &&
      aws_apigatewayv2_integration.submission.integration_type == "AWS_PROXY" &&
      aws_apigatewayv2_integration.submission.integration_subtype == "SQS-SendMessage" &&
      aws_apigatewayv2_integration.submission.payload_format_version == "1.0" &&
      aws_apigatewayv2_integration.submission.timeout_milliseconds == 10000 &&
      aws_apigatewayv2_integration.submission.request_parameters.MessageBody == "$request.body" &&
      aws_apigatewayv2_integration.reader.integration_type == "AWS_PROXY" &&
      aws_apigatewayv2_integration.reader.payload_format_version == "2.0" &&
      aws_apigatewayv2_integration.reader.timeout_milliseconds == 10000 &&
      aws_lambda_function.reader.runtime == "nodejs22.x" &&
      aws_lambda_function.reader.handler == "index.handler" &&
      aws_lambda_function.reader.timeout == 10 &&
      aws_lambda_function.reader.source_code_hash == filebase64sha256("build/reader.zip") &&
      length(aws_lambda_permission.reader) == 2 &&
      alltrue([for permission in aws_lambda_permission.reader : permission.action == "lambda:InvokeFunction" && permission.principal == "apigateway.amazonaws.com"])
    )
    error_message = "Routes, integrations or reader settings violate the delivery contract."
  }
  assert {
    condition = (
      length(aws_cloudwatch_metric_alarm.queue) == 2 &&
      aws_cloudwatch_metric_alarm.queue["dead_letter"].metric_name == "ApproximateNumberOfMessagesVisible" &&
      aws_cloudwatch_metric_alarm.queue["dead_letter"].threshold == 0 &&
      aws_cloudwatch_metric_alarm.queue["dead_letter"].dimensions.QueueName == aws_sqs_queue.dead_letter.name &&
      aws_cloudwatch_metric_alarm.queue["source_age"].metric_name == "ApproximateAgeOfOldestMessage" &&
      aws_cloudwatch_metric_alarm.queue["source_age"].threshold == 259200 &&
      aws_cloudwatch_metric_alarm.queue["source_age"].dimensions.QueueName == aws_sqs_queue.results.name &&
      alltrue([for alarm in aws_cloudwatch_metric_alarm.queue :
        alarm.namespace == "AWS/SQS" && alarm.statistic == "Maximum" &&
        alarm.period == 300 && alarm.evaluation_periods == 1 &&
        alarm.comparison_operator == "GreaterThanThreshold" &&
        alarm.treat_missing_data == "notBreaching" && length(alarm.alarm_actions) == 0
      ]) &&
      aws_cloudwatch_log_group.reader.retention_in_days == 30
    )
    error_message = "Queue alarm metrics, thresholds, dimensions or log retention differ."
  }
}

run "reject_wildcard_origin" {
  command = plan
  variables { allowed_origin = "*" }
  expect_failures = [var.allowed_origin]
}
