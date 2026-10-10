output "submission_url" {
  value = "${aws_apigatewayv2_api.results.api_endpoint}/results"
}
output "read_url" {
  value = "${aws_apigatewayv2_api.results.api_endpoint}/results"
}
output "acknowledgement_url" {
  value = "${aws_apigatewayv2_api.results.api_endpoint}/results/ack"
}
output "queue_url" {
  value = aws_sqs_queue.results.url
}
output "queue_arn" {
  value = aws_sqs_queue.results.arn
}
output "dead_letter_queue_url" {
  value = aws_sqs_queue.dead_letter.url
}
output "queue_name" {
  value = aws_sqs_queue.results.name
}
output "dead_letter_queue_name" {
  value = aws_sqs_queue.dead_letter.name
}
output "reader_function_name" {
  value = aws_lambda_function.reader.function_name
}
output "alarm_names" {
  value = { for key, alarm in aws_cloudwatch_metric_alarm.queue : key => alarm.alarm_name }
}
