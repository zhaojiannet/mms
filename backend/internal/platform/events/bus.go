// Package events 提供极简事件总线
//
// 设计（遵循 CLAUDE.md 第 9 条）：
//   - 自写 ~80 行，不引入 Watermill / asynq / Kafka
//   - 内存 pub-sub，订阅者跑在 goroutine 里
//   - 事件类型用字符串（如 "appointment.created"），payload 是 any
//   - 订阅者崩溃不影响其他订阅者（recover）
//
// 使用：
//   events.DefaultBus.Subscribe("appointment.created", func(p any) { ... })
//   events.DefaultBus.Publish("appointment.created", appt)
package events

import (
	"log/slog"
	"sync"
)

type Handler func(payload any)

type Bus struct {
	mu   sync.RWMutex
	subs map[string][]Handler
}

func NewBus() *Bus {
	return &Bus{subs: map[string][]Handler{}}
}

func (b *Bus) Subscribe(topic string, h Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subs[topic] = append(b.subs[topic], h)
}

// Publish 把事件分发给所有订阅者（每个订阅者在独立 goroutine 执行）
// 订阅者崩溃不影响其他订阅者，也不阻塞业务事务
func (b *Bus) Publish(topic string, payload any) {
	b.mu.RLock()
	hs := b.subs[topic]
	b.mu.RUnlock()
	for _, h := range hs {
		h := h
		go func() {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("event handler panic", "topic", topic, "err", r)
				}
			}()
			h(payload)
		}()
	}
}

// ---- 约定好的事件常量 ----
const (
	TopicAppointmentCreated = "appointment.created"
	TopicTransactionCreated = "transaction.created"
	TopicTransactionVoided  = "transaction.voided"
	TopicMemberRegistered   = "member.registered"
)

// DefaultBus 全局单例（main 启动时注册订阅者，业务代码 Publish）
var DefaultBus = NewBus()
