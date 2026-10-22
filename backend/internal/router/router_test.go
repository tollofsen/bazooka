package router

import (
	"testing"

	pb "github.com/tollofsen/bazooka/gen/bazookapb"
)

func TestPublishSubscribe(t *testing.T) {
	r := New()
	ch, cancel := r.Subscribe("s1")
	other, cancelOther := r.Subscribe("s2")
	defer cancelOther()
	r.Publish(&pb.Output{SessionId: "s1", Text: "hi"})
	if got := (<-ch).Text; got != "hi" {
		t.Fatalf("got %q", got)
	}
	select {
	case <-other:
		t.Fatal("s2 should not receive s1 output")
	default:
	}
	cancel()
	cancel() // idempotent
	r.Publish(&pb.Output{SessionId: "s1", Text: "x"})
}
