package mq

import (
	sharedrabbitmq "inkwords-backend/shared/platform/rabbitmq"
)

// GenerationRequestedMessage 定义发送到 RabbitMQ 的生成任务标准消息体。
type GenerationRequestedMessage = sharedrabbitmq.GenerationRequestedMessage

// ParseRequestedMessage 定义发送到 RabbitMQ 的解析任务标准消息体。
type ParseRequestedMessage = sharedrabbitmq.ParseRequestedMessage

// ExportRequestedMessage 定义发送到 RabbitMQ 的导出任务标准消息体。
type ExportRequestedMessage = sharedrabbitmq.ExportRequestedMessage
