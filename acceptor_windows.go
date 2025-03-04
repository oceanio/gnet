// Copyright (c) 2019 Andy Pan
// Copyright (c) 2018 Joshua J Baker
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package gnet

import (
	"runtime"
	"time"
)

func (svr *server) listenerRun(lockOSThread bool) {
	if lockOSThread {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
	}

	var err error
	defer func() { svr.signalShutdown(err) }()
	var packet [0x10000]byte
	for {
		if svr.ln.pconn != nil {
			// Read data from UDP socket.
			n, addr, e := svr.ln.pconn.ReadFrom(packet[:])
			if e != nil {
				err = e
				return
			}

			el := svr.subEventLoopSet.next(addr)
			c := newUDPConn(el, svr.ln.lnaddr, addr)
			el.ch <- packUDPConn(c, packet[:n])
		} else {
			// Accept TCP socket.
			conn, e := svr.ln.ln.Accept()
			if e != nil {
				err = e
				return
			}
			el := svr.subEventLoopSet.next(conn.RemoteAddr())
			c := newTCPConn(conn, el)
			el.ch <- c
			go func() {
				var packet [0x10000]byte
				for {
					n, err := c.conn.Read(packet[:])
					if err != nil {
						_ = c.conn.SetReadDeadline(time.Time{})
						el.ch <- &stderr{c, err}
						return
					}
					el.ch <- packTCPConn(c, packet[:n])
				}
			}()
		}
	}
}
