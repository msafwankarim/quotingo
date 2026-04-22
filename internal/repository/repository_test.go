package repository

import "testing"

func TestToJokeItem_SingleWithText(t *testing.T) {
	raw := jokeResponse{Type: "single", Joke: "why did the chicken cross the road"}
	item, ok := toJokeItem(raw)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if item.Setup != "why did the chicken cross the road" {
		t.Errorf("unexpected Setup: %q", item.Setup)
	}
	if item.TwoPart {
		t.Error("single joke should not set TwoPart")
	}
}

func TestToJokeItem_SingleEmpty(t *testing.T) {
	raw := jokeResponse{Type: "single", Joke: "   "}
	_, ok := toJokeItem(raw)
	if ok {
		t.Fatal("expected ok=false for whitespace-only joke")
	}
}

func TestToJokeItem_TwoPart(t *testing.T) {
	raw := jokeResponse{Type: "twopart", Setup: "knock knock", Delivery: "who's there"}
	item, ok := toJokeItem(raw)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if !item.TwoPart {
		t.Error("twopart joke must set TwoPart=true")
	}
	if item.Setup != "knock knock" || item.Delivery != "who's there" {
		t.Errorf("unexpected fields: %+v", item)
	}
}

func TestToJokeItem_TwoPartMissingDelivery(t *testing.T) {
	raw := jokeResponse{Type: "twopart", Setup: "setup only"}
	_, ok := toJokeItem(raw)
	if ok {
		t.Fatal("expected ok=false when delivery is missing")
	}
}

func TestToJokeItem_UnknownType(t *testing.T) {
	raw := jokeResponse{Type: "novel", Joke: "x"}
	_, ok := toJokeItem(raw)
	if ok {
		t.Fatal("expected ok=false for unknown type")
	}
}
