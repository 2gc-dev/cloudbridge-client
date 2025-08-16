package protocol

import (
	"testing"
	"time"
)

func TestProtocolEngine(t *testing.T) {
	pe := NewProtocolEngine()

	// Test initial state
	if pe.GetBestProtocol() != QUIC {
		t.Errorf("Expected QUIC as best protocol, got %s", pe.GetBestProtocol())
	}

	// Test recording success
	pe.RecordSuccess(QUIC, 100*time.Millisecond)
	stats := pe.GetStats()
	if stats["quic"].(map[string]interface{})["success_count"].(int64) != 1 {
		t.Error("Expected success count to be 1")
	}

	// Test recording failure
	pe.RecordFailure(QUIC, "test failure")
	stats = pe.GetStats()
	if stats["quic"].(map[string]interface{})["failure_count"].(int64) != 1 {
		t.Error("Expected failure count to be 1")
	}

	// Test protocol switching
	pe.RecordFailure(QUIC, "test failure")
	pe.RecordFailure(QUIC, "test failure")
	pe.RecordFailure(QUIC, "test failure")
	pe.RecordFailure(QUIC, "test failure")
	pe.RecordFailure(QUIC, "test failure")
	if pe.GetBestProtocol() != HTTP2 {
		t.Errorf("Expected HTTP2 as best protocol after QUIC failures, got %s", pe.GetBestProtocol())
	}
}

func TestProtocolEngineStats(t *testing.T) {
	pe := NewProtocolEngine()

	// Record some data
	pe.RecordSuccess(QUIC, 50*time.Millisecond)
	pe.RecordSuccess(QUIC, 75*time.Millisecond)
	pe.RecordFailure(QUIC, "test failure")
	pe.RecordSuccess(HTTP2, 100*time.Millisecond)

	stats := pe.GetStats()

	// Check QUIC stats
	quicStats := stats["quic"].(map[string]interface{})
	if quicStats["success_count"].(int64) != 2 {
		t.Errorf("Expected QUIC success count 2, got %d", quicStats["success_count"])
	}
	if quicStats["failure_count"].(int64) != 1 {
		t.Errorf("Expected QUIC failure count 1, got %d", quicStats["failure_count"])
	}

	// Check HTTP2 stats
	http2Stats := stats["http2"].(map[string]interface{})
	if http2Stats["success_count"].(int64) != 1 {
		t.Errorf("Expected HTTP2 success count 1, got %d", http2Stats["success_count"])
	}
}

func TestProtocolEngineReset(t *testing.T) {
	pe := NewProtocolEngine()

	// Record some data
	pe.RecordSuccess(QUIC, 50*time.Millisecond)
	pe.RecordFailure(QUIC, "test failure")

	// Reset stats
	pe.ResetStats()

	stats := pe.GetStats()
	for _, protocol := range []string{"quic", "http2", "http1"} {
		protoStats, ok := stats[protocol].(map[string]interface{})
		if !ok {
			t.Errorf("Expected stats for protocol %s", protocol)
			continue
		}
		if protoStats["success_count"].(int64) != 0 || protoStats["failure_count"].(int64) != 0 {
			t.Errorf("Expected zeroed stats for protocol %s after reset", protocol)
		}
	}
} 