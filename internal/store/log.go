package store

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

type logRecord struct {
	TransactionID string           `json:"transaction_id"`
	Phase         string           `json:"phase"`
	ProductionID  string           `json:"production_id"`
	Operation     string           `json:"operation"`
	Envelope      SnapshotEnvelope `json:"envelope,omitempty"`
	RecordedAt    time.Time        `json:"recorded_at"`
}

func (s *DiskStore) commitLocked(id string, env SnapshotEnvelope, operation string) error {
	txID := fmt.Sprintf("%s-%d", id, time.Now().UnixNano())
	prepared := logRecord{TransactionID: txID, Phase: "PREPARED", ProductionID: id, Operation: operation, Envelope: env, RecordedAt: time.Now().UTC()}
	if err := appendSyncedRecord(s.logPath, prepared); err != nil {
		return err
	}
	path := filepath.Join(s.snapDir, id+".json")
	if err := writeAtomicJSON(path, env); err != nil {
		return err
	}
	committed := logRecord{TransactionID: txID, Phase: "COMMITTED", ProductionID: id, Operation: operation, Envelope: env, RecordedAt: time.Now().UTC()}
	if err := appendSyncedRecord(s.logPath, committed); err != nil {
		return err
	}
	return nil
}

func appendSyncedRecord(path string, record logRecord) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o640)
	if err != nil {
		return fmt.Errorf("打开操作日志: %w", err)
	}
	encoder := json.NewEncoder(f)
	if err := encoder.Encode(record); err != nil {
		_ = f.Close()
		return fmt.Errorf("写操作日志: %w", err)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return fmt.Errorf("同步操作日志: %w", err)
	}
	return f.Close()
}

func (s *DiskStore) replayConfirmedLog() error {
	f, err := os.Open(s.logPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("打开恢复日志: %w", err)
	}
	defer f.Close()
	reader := bufio.NewReader(f)
	confirmed := make(map[string]logRecord)
	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			var record logRecord
			if parseErr := json.Unmarshal(line, &record); parseErr != nil {
				if err == io.EOF {
					break
				}
				return fmt.Errorf("解析操作日志: %w", parseErr)
			}
			if record.Phase == "COMMITTED" {
				confirmed[record.TransactionID] = record
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("读取操作日志: %w", err)
		}
	}
	for _, record := range confirmed {
		current, ok := s.projects[record.ProductionID]
		if !ok || current.Aggregate.Production.Revision < record.Envelope.Aggregate.Production.Revision {
			s.projects[record.ProductionID] = record.Envelope
			if err := writeAtomicJSON(filepath.Join(s.snapDir, record.ProductionID+".json"), record.Envelope); err != nil {
				return err
			}
		}
	}
	return nil
}
