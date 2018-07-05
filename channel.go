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

package main

import (
	"fmt"
	"sync"
	"time"
)

type channel struct {
	sendc chan interface{}
	stopc chan interface{}

	sync.Mutex
	active bool
	ch     chan interface{}
}

func (c *channel) run() (chan interface{}, error) {
	if c.isActive() {
		return nil, fmt.Errorf("channel: unable to run: already active")
	}

	// create output channel
	c.Lock()
	c.ch = make(chan interface{})
	c.active = true
	c.Unlock()

	send := func(m interface{}) {
		defer func() {
			if err := recover(); err != nil {
				// we tried to send something trough ch and we found it closed.
				// need to remove this channel.
				c.stop()
			}
		}()

		select {
		case c.out() <- m:
		case <-time.After(time.Second):
			c.stop()
		}
	}

	go func() {
		for {
			select {
			case m := <-c.sendc:
				go send(m)
			case <-c.stopc:
				return
			}
		}
	}()

	return c.out(), nil
}

func newChannel() *channel {
	return &channel{
		sendc:  make(chan interface{}),
		stopc:  make(chan interface{}),
		active: false,
	}
}

func (c *channel) send(m interface{}) {
	if c.isActive() {
		c.sendc <- m
	}
}

func (c *channel) stop() {
	if !c.isActive() {
		return
	}

	c.stopc <- struct{}{} // stop run()
	c.setIsActive(false)  // set inactive so we don't forward messages anymore
	c.Lock()
	closeChanSafe(c.ch) // close output channel
	c.ch = nil          // and remove it
	c.Unlock()
}

func (c *channel) isActive() bool {
	c.Lock()
	defer c.Unlock()

	return c.active
}

func (c *channel) setIsActive(ia bool) {
	c.Lock()
	c.active = ia
	c.Unlock()
}

func (c *channel) out() chan interface{} {
	c.Lock()
	defer c.Unlock()

	return c.ch
}

func closeChanSafe(c chan interface{}) {
	defer func() {
		if r := recover(); r != nil {
			// tried to close c, which was already closed.
		}
	}()

	close(c)
}
