/*************************************************************************
* This file is a part of msgfiber, A decentralized and distributed message
* synchronization system

* Copyright (C) 2014  Fang Li <surivlee@gmail.com> and Funplus, Inc.
*
* This program is free software; you can redistribute it and/or modify
* it under the terms of the GNU General Public License as published by
* the Free Software Foundation; either version 2 of the License, or
* (at your option) any later version.
*
* This program is distributed in the hope that it will be useful,
* but WITHOUT ANY WARRANTY; without even the implied warranty of
* MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
* GNU General Public License for more details.
*
* You should have received a copy of the GNU General Public License along
* with this program; if not, see http://www.gnu.org/licenses/gpl-2.0.html
*************************************************************************/

package clientpool

import (
	"github.com/fangli/msgfiber/structure"
	"github.com/vmihailenco/msgpack"
	"net"
	"sync"
)

type Msg struct {
	Channel    string
	Message    []byte
	sourceConn net.Conn
}

type Pool struct {
	clients  map[net.Conn][]string
	msgQueue chan Msg
	poolLock sync.Mutex
}

func Contains(list []string, elem string) bool {
	for _, t := range list {
		if t == elem {
			return true
		}
	}
	return false
}

func (p *Pool) Close(conn net.Conn) {
	p.poolLock.Lock()
	defer p.poolLock.Unlock()
	conn.Close()
	delete(p.clients, conn)
}

func (p *Pool) Subscribe(conn net.Conn, channel []string) error {
	p.poolLock.Lock()
	defer p.poolLock.Unlock()
	p.clients[conn] = channel
	return nil
}

func (p *Pool) Send(conn net.Conn, data []byte) error {
	_, err := conn.Write(data)
	return err
}

func (p *Pool) CurrentChannel(conn net.Conn) []string {
	p.poolLock.Lock()
	defer p.poolLock.Unlock()
	return p.clients[conn]
}

func (p *Pool) BroadcastMonitoring() {
	for m := range p.msgQueue {
		syncMsg := structure.NewSyncResponse()
		syncMsg.Message = m.Message
		syncMsg.Channel = m.Channel
		payload, _ := msgpack.Marshal(syncMsg)
		p.poolLock.Lock()
		for conn, channels := range p.clients {
			if conn == m.sourceConn {
				continue
			}
			if Contains(channels, m.Channel) {
				p.Send(conn, payload)
			}
		}
		p.poolLock.Unlock()
	}
}

func (p *Pool) Broadcast(channel string, msg []byte, source net.Conn) {
	p.msgQueue <- Msg{
		Channel:    channel,
		Message:    msg,
		sourceConn: source,
	}
}

func (p *Pool) PendingMsgCount() int {
	return len(p.msgQueue)
}

func (p *Pool) ClientCount() int {
	p.poolLock.Lock()
	defer p.poolLock.Unlock()
	return len(p.clients)
}

func (p *Pool) Init() {
	p.clients = make(map[net.Conn][]string)
	p.msgQueue = make(chan Msg, 1000)
	go p.BroadcastMonitoring()
}
