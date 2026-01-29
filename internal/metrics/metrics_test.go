package metrics

import (
	"testing"
)

func TestRecordCheck(t *testing.T) {
	RecordCheck("rpc", "aptoslabs", true, 100)
	RecordCheck("dapp", "explorer", false, 0)
}

func TestSetBuildInfo(t *testing.T) {
	SetBuildInfo("0.1.0", "abc123", "2026-01-24")
}
