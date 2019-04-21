package cond

import "testing"

func TestCondRegisterBeforeNotifyShouldNotBroadcast(t *testing.T) {
	var c Cond
	ch := make(chan int, 1)
	c.Register(ch, 0)
	select {
	case <-ch:
		t.Fatal("ch was notified before broadcast")
	default:
	}
}

func TestCondRegisterAfterNotifyShouldBroadcast(t *testing.T) {
	var c Cond
	ch := make(chan int, 1)
	c.Notify()
	c.Register(ch, 0)
	select {
	case v := <-ch:
		if v != 1 {
			t.Fatal("ch was notified with the wrong sequence number", v)
		}
	default:
		t.Fatal("ch was not notified on registration")
	}
}

func TestCondRegisterAfterNotifyWithCorrectSequenceShouldNotBroadcast(t *testing.T) {
	var c Cond
	ch := make(chan int, 1)
	c.Notify()
	c.Register(ch, 0)
	seq := <-ch

	c.Register(ch, seq)
	select {
	case v := <-ch:
		t.Fatal("ch was notified immediately with seq", v)
	default:
	}
}
