# Standard SQS queues replace RabbitMQ for Canvas import, grade sync, SMS, and grading agent.
# First 1M requests/month are free (AWS Always Free for SQS).

resource "aws_sqs_queue" "dlq" {
  for_each = local.sqs_queues

  name                      = "${local.name_prefix}-${each.value}-dlq"
  message_retention_seconds = 1209600 # 14 days

  tags = {
    Name = "${local.name_prefix}-${each.value}-dlq"
  }
}

resource "aws_sqs_queue" "main" {
  for_each = local.sqs_queues

  name                       = "${local.name_prefix}-${each.value}"
  visibility_timeout_seconds = 900    # 15m — covers long Canvas imports / grading
  message_retention_seconds  = 345600 # 4 days
  receive_wait_time_seconds  = 20     # long polling (cheaper + lower latency noise)

  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.dlq[each.key].arn
    maxReceiveCount     = 5
  })

  tags = {
    Name = "${local.name_prefix}-${each.value}"
  }
}

resource "aws_sqs_queue_redrive_allow_policy" "dlq" {
  for_each = local.sqs_queues

  queue_url = aws_sqs_queue.dlq[each.key].id

  redrive_allow_policy = jsonencode({
    redrivePermission = "byQueue"
    sourceQueueArns   = [aws_sqs_queue.main[each.key].arn]
  })
}
