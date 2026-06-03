package mq

import (
	"encoding/json"

	"github.com/google/uuid"
)

// GenerationRequestedMessage 定义发送到 RabbitMQ 的生成任务标准消息体。
type GenerationRequestedMessage struct {
	TaskID  uuid.UUID       `json:"task_id"`
	Kind    string          `json:"kind"`
	UserID  uuid.UUID       `json:"user_id"`
	Payload json.RawMessage `json:"payload"`
}

// RoutingKey 返回生成任务创建事件使用的固定路由键，确保生产者与消费者契约稳定。
func (GenerationRequestedMessage) RoutingKey() string {
	return "generation.requested"
}
