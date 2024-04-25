package events

import (
	"context"
	"encoding/json"
	"github.com/IBM/sarama"
)

const topicShareGradeEvent = "share_grade_event"

type ShareGradeEvent struct {
	Uid       int64
	StudentId string
	Password  string
}

type Producer interface {
	ProduceShareGradeEvent(ctx context.Context, evt ShareGradeEvent) error
}

type SaramaProducer struct {
	producer sarama.SyncProducer
}

func NewSaramaProducer(producer sarama.SyncProducer) *SaramaProducer {
	return &SaramaProducer{producer: producer}
}

func (s *SaramaProducer) ProduceShareGradeEvent(ctx context.Context, evt ShareGradeEvent) error {
	data, err := json.Marshal(evt)
	if err != nil {
		return err
	}
	_, _, err = s.producer.SendMessage(&sarama.ProducerMessage{
		Topic: topicShareGradeEvent,
		Value: sarama.ByteEncoder(data),
	})
	return err
}
