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

package processor

import (
	"bytes"
	"container/list"
	"errors"
	"github.com/fangli/msgfiber/structure"
	"github.com/vmihailenco/msgpack"
	"log"
	"net"
	"time"
)

type Client struct {
	Name       string
	Channels   []string
	Incoming   chan structure.Command
	Outgoing   chan interface{}
	Conn       net.Conn
	Quit       chan bool
	ClientList *list.List
}

func (c *Client) Equal(other *Client) bool {
	if bytes.Equal([]byte(c.Name), []byte(other.Name)) {
		if c.Conn == other.Conn {
			return true
		}
	}
	return false
}

func (c *Client) Remove() {
	c.Quit <- true
	for entry := c.ClientList.Front(); entry != nil; entry = entry.Next() {
		client := entry.Value.(Client)
		if c.Equal(&client) {
			c.ClientList.Remove(entry)
		}
	}
	c.Conn.Close()
}

func (c *Processor) broadcast(channel string, msg []byte, source net.Conn) {
	c.clients.Broadcast(channel, msg, source)
}

func (c *Processor) writePayload(client *Client, payload interface{}) {
	client.Outgoing <- payload
}

func (c *Processor) stats(client *Client) {
	c.writePayload(client, c.genStatsInfo())
}

func (c *Processor) clusterStats(client *Client) {
	c.writePayload(client, c.genClusterStatsInfo())
}

func (c *Processor) get(client *Client) {
	msgMap := make(map[string][]byte)
	for _, channel := range c.clients.CurrentChannel(client.Conn) {
		msgMap[channel] = c.store.Get(channel)
	}
	resp := structure.NewGetResponse()
	resp.Channel = msgMap
	c.writePayload(client, resp)
}

func (c *Processor) subscribe(client *Client, channels []string) {
	c.clients.Subscribe(client.Conn, channels)
	resp := structure.NewSubscribeResponse()
	resp.Channel = channels
	c.writePayload(client, resp)
}

func (c *Processor) set(client *Client, channel string, msg []byte) {
	c.writePayload(client, c.setMsg(client.Conn, channel, msg))
}

func (c *Processor) sync(client *Client, channel string, msg []byte) {
	if c.store.DryUpdate(channel, msg) == nil {
		c.cluster.NodeSync(channel, msg)
		c.broadcast(channel, msg, client.Conn)
	}
}

func (c *Processor) errResponse(client *Client, op string, errNotice string) {
	resp := structure.NewErrorResponse()
	resp.Info = errNotice
	resp.Op = op
	client.Outgoing <- resp
}

func (c *Processor) execCommand(client *Client, cmd *structure.Command) {
	switch cmd.Op {
	case "stats":
		c.stats(client)
	case "cluster_stats":
		c.clusterStats(client)
	case "get":
		c.get(client)
	case "subscribe":
		if cmd.Channel == nil {
			c.errResponse(client, "subscribe", "You must specific valid 'Channel' field for the channels you want to subscribe")
			return
		}
		c.subscribe(client, cmd.Channel)
	case "set":
		if cmd.Channel == nil || len(cmd.Channel) != 1 {
			c.errResponse(client, "set", "You must specific valid 'Channel' field for the channels you want to update")
			return
		}
		if cmd.Message == nil {
			c.errResponse(client, "set", "You must specific valid 'Message' for those channels")
			return
		}
		c.set(client, cmd.Channel[0], cmd.Message)
	case "sync":
		if cmd.Channel == nil || len(cmd.Channel) != 1 {
			errors.New("Invalid sync command received, no valid 'Channel'")
			return
		}
		if cmd.Message == nil {
			errors.New("Invalid sync command received, no valid 'Message'")
			return
		}
		c.sync(client, cmd.Channel[0], cmd.Message)
	case "":
		c.errResponse(client, "UNKNOWN", "Directive 'Op' not found, I don't know what to do.")
	default:
		c.errResponse(client, cmd.Op, "Directive 'Op' incorrect. I don't know what does '"+cmd.Op+"' mean")
	}
}

func (c *Processor) ClientSender(client *Client) {

}

func (c *Processor) ClientReader(client *Client) {
	defer client.Remove()
	var err error
	decoder := msgpack.NewDecoder(client.Conn)
	for {
		cmd := &structure.Command{}
		client.Conn.SetReadDeadline(time.Now().Add(c.Config.ConnExpires))
		err = decoder.Decode(cmd)
		if err != nil {
			return
		}
		// log.Printf("%+v", cmd)
		c.execCommand(client, cmd)
	}
}

func (c *Processor) ClientHandler(conn net.Conn, clientList *list.List) {
	newClient := &Client{
		Name:       conn.RemoteAddr().String(),
		Incoming:   make(chan structure.Command),
		Outgoing:   make(chan interface{}),
		Conn:       conn,
		Quit:       make(chan bool),
		ClientList: clientList,
	}
	clientList.PushBack(*newClient)
	go c.ClientSender(newClient)
	go c.ClientReader(newClient)
}

func (c *Processor) serveTcp() {
	clientList := list.New()

	ln, err := net.Listen("tcp", c.Config.TcpListen)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go c.ClientHandler(conn, clientList)
	}
}
