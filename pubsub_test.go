/*
MIT License

Copyright (c) 2018 Daniel Morandini

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package main_test

import (
	"sync"
	"testing"
	"time"

	"github.com/danielmorandini/booster/pubsub"
)

var nop = func(interface{}) error {
	return nil
}

func TestSub(t *testing.T) {
	ps := pubsub.New()
	s := "t1"
	if _, err := ps.Sub(&pubsub.Command{
		Topic: s,
		Run:   nop,
	}); err != nil {
		t.Fatal(err)
	}

	if err := ps.Close(s); err != nil {
		t.Fatal(err)
	}
}

func TestPub(t *testing.T) {
	ps := pubsub.New()
	var wg sync.WaitGroup
	timer := time.AfterFunc(time.Second, func() {
		t.Fatal("t1 not responding")
	})

	s := "t1"
	wg.Add(1)
	if _, err := ps.Sub(&pubsub.Command{
		Topic: s,
		Run: func(i interface{}) error {
			if i != "fakedata" {
				t.Fatalf("unexpected data: %v", i)
			}

			wg.Done()
			return nil
		},
	}); err != nil {
		t.Fatal(err)
	}

	ps.Pub("fakedata", s)

	wg.Wait()
	timer.Stop()
}

func TestPub_multiple(t *testing.T) {
	ps := pubsub.New()
	var wg sync.WaitGroup
	timer := time.AfterFunc(time.Second, func() {
		t.Fatal("t1/t2 not responding")
	})

	wg.Add(1)
	if _, err := ps.Sub(&pubsub.Command{
		Topic: "t1",
		Run: func(i interface{}) error {
			if i != "fakedata" {
				t.Fatalf("unexpected data: %v", i)
			}

			wg.Done()
			return nil
		},
	}); err != nil {
		t.Fatal(err)
	}

	wg.Add(1)
	if _, err := ps.Sub(&pubsub.Command{
		Topic: "t2",
		Run: func(i interface{}) error {
			if i != "fakedata" {
				t.Fatalf("unexpected data: %v", i)
			}

			wg.Done()
			return nil
		},
	}); err != nil {
		t.Fatal(err)
	}

	ps.Pub("fakedata", "t1")
	ps.Pub("fakedata", "t2")

	wg.Wait()
	timer.Stop()
}

func TestUnsub(t *testing.T) {
	ps := pubsub.New()
	cmd := &pubsub.Command{
		Topic: "t1",
		Run:   nop,
	}
	_, err := ps.Sub(cmd)
	if err != nil {
		t.Fatal(err)
	}

	if err := ps.Unsub(cmd.Ref, "t1"); err != nil {
		t.Fatal(err)
	}
}

func TestCancel(t *testing.T) {
	timer := time.AfterFunc(time.Second, func() {
		t.Fatal("t1 not closing")
	})

	ps := pubsub.New()
	var wg sync.WaitGroup
	wg.Add(1)
	cancel, err := ps.Sub(&pubsub.Command{
		Topic: "t1",
		Run:   nop,
		PostRun: func(error) {
			wg.Done()
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := cancel(); err != nil {
		t.Fatal(err)
	}

	wg.Wait()
	timer.Stop()
}

func TestMultiSub_concurrent(t *testing.T) {
	ps := pubsub.New()
	wait := make(chan struct{}, 4)
	timer := time.AfterFunc(time.Second, func() {
		t.Fatal("t1/t2 not responding")
	})

	// these two messages shuold be ignored
	ps.Pub("fake_data_t1", "t1")
	ps.Pub("fake_data_t2", "t2")

	cmd1 := &pubsub.Command{
		Topic: "t1",
		Run: func(d interface{}) error {
			if d != "foo" {
				t.Fatalf("unexpected data from t1: %v", d)
			}

			wait <- struct{}{}
			return nil
		},
	}
	cmd2 := &pubsub.Command{
		Topic: "t2",
		Run: func(d interface{}) error {
			if d != "bar" {
				t.Fatalf("unexpected data from t2: %v", d)
			}

			wait <- struct{}{}
			return nil
		},
	}

	go func() {
		if _, err := ps.Sub(cmd1); err != nil {
			t.Fatal(err)
		}

		wait <- struct{}{}
	}()

	go func() {
		if _, err := ps.Sub(cmd2); err != nil {
			t.Fatal(err)
		}

		wait <- struct{}{}
	}()

	// wait for subscription
	<-wait
	<-wait

	ps.Pub("foo", "t1")
	ps.Pub("bar", "t2")

	// wait for the functions to be called
	<-wait
	<-wait
	timer.Stop()

	if err := ps.Unsub(cmd1.Ref, "t1"); err != nil {
		t.Fatalf("unable to unsub ch1 from t1: %v", err)
	}
	if err := ps.Unsub(cmd2.Ref, "t2"); err != nil {
		t.Fatalf("unable to unsub ch1 from t2: %v", err)
	}

}

func TestMultiSub(t *testing.T) {
	ps := pubsub.New()

	timer := time.AfterFunc(time.Second, func() {
		t.Fatal("t1 not responding")
	})
	var wg sync.WaitGroup

	f := func() {
		if _, err := ps.Sub(&pubsub.Command{
			Topic: "t1",
			Run: func(d interface{}) error {
				wg.Done()
				return nil
			},
		}); err != nil {
			t.Fatal(err)
		}
	}

	wg.Add(2)
	f()
	f()

	ps.Pub("hi", "t1")

	wg.Wait()
	timer.Stop()
}
