package buildlog

import (
	"io"
	"testing"
)

func drain(ch <-chan Chunk) (out []Chunk) {
	for c := range ch {
		out = append(out, c)
	}
	return out
}

func TestSinkPublishesWithOffsets(t *testing.T) {
	store := Store{Dir: t.TempDir()}
	hub := NewHub()
	ch, _ := hub.Subscribe(1)

	sink, err := store.OpenSink(1, hub)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.WriteString(sink, "abc")
	_, _ = io.WriteString(sink, "de")
	_ = sink.Close()

	got := drain(ch)
	if len(got) != 2 || got[0].Offset != 0 || string(got[0].Data) != "abc" || got[1].Offset != 3 {
		t.Fatalf("chunks %+v", got)
	}
	if data, _ := store.Read(1); string(data) != "abcde" {
		t.Fatalf("persisted %q", data)
	}

	// After the log ended, new subscribers get a closed channel.
	late, _ := hub.Subscribe(1)
	if _, open := <-late; open {
		t.Fatal("late subscriber should get a closed channel")
	}

	// Reopening a log continues from the persisted size.
	hub.Reopen(1)
	ch, _ = hub.Subscribe(1)
	sink, _ = store.OpenSink(1, hub)
	_, _ = io.WriteString(sink, "f")
	_ = sink.Close()
	if got := drain(ch); len(got) != 1 || got[0].Offset != 5 {
		t.Fatalf("reopened chunks %+v", got)
	}
}

func TestUnsubscribeAndSlowSubscriber(t *testing.T) {
	hub := NewHub()
	ch, cancel := hub.Subscribe(2)
	cancel()
	cancel() // idempotent
	if _, open := <-ch; open {
		t.Fatal("cancelled subscription must be closed")
	}

	slow, _ := hub.Subscribe(3)
	for i := range subscriberBuffer + 10 {
		hub.Publish(3, Chunk{Offset: int64(i), Data: []byte("x")})
	}
	if len(slow) != subscriberBuffer {
		t.Fatalf("buffer %d", len(slow))
	}
	hub.Close(3)
}

func TestSince(t *testing.T) {
	c := Chunk{Offset: 10, Data: []byte("hello")}
	for seen, want := range map[int64]string{0: "hello", 10: "hello", 12: "llo", 15: "", 20: ""} {
		if got := string(Since(c, seen)); got != want {
			t.Errorf("seen %d: got %q want %q", seen, got, want)
		}
	}
}
