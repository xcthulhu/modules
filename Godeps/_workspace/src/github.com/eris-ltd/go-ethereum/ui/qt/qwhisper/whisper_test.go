package qwhisper

import (
	"testing"

	"github.com/eris-ltd/modules/Godeps/_workspace/src/github.com/eris-ltd/go-ethereum/whisper"
)

func TestHasIdentity(t *testing.T) {
	qw := New(whisper.New())
	id := qw.NewIdentity()
	if !qw.HasIdentity(id) {
		t.Error("expected to have identity")
	}
}
