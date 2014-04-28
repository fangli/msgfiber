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
	"errors"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fangli/msgfiber/memstats"
	"github.com/fangli/msgfiber/nodepool"
	"github.com/fangli/msgfiber/parsecfg"
	"github.com/fangli/msgfiber/storemidware"
	"github.com/fangli/msgfiber/structure"
	"github.com/vmihailenco/msgpack"
)

type WritePayload struct {
	Payload      interface{}
	PayloadBytes []byte
	isBytes      int
}

type Client struct {
	Name        string
	Channels    []string
	ChannelLock sync.Mutex
	Outgoing    chan WritePayload
	Conn        net.Conn
	ExitChan    chan byte
}

type ProcessorCounters struct {
	startTime           int64
	TotalConnections    int64
	RejectedConnections int64
	Commands            int64
	CommandSync         int64
}

type Processor struct {
	Config            parsecfg.Config
	store             storemidware.StoreMidware
	nodes             nodepool.Pool
	syncBroadcastChan chan structure.SyncResponse
	pingResponseBytes []byte
	Stats             ProcessorCounters
	ClientList        map[string]*Client
	ClientListLock    sync.Mutex
	currentStats      structure.NodeStatus
	currentStatsLock  sync.Mutex
}

func (p *Processor) ContainChannel(client *Client, channel string) bool {
	client.ChannelLock.Lock()
	defer client.ChannelLock.Unlock()
	for _, t := range client.Channels {
		if t == channel {
			return true
		}
	}
	return false
}

func (p *Processor) Write(client *Client, payload interface{}, rawPayload []byte) error {
	if len(client.Outgoing) >= 4 {
		return errors.New("Client write buffer full")
	}

	if payload != nil {
		client.Outgoing <- WritePayload{
			Payload: payload,
		}
	} else {
		client.Outgoing <- WritePayload{
			PayloadBytes: rawPayload,
			isBytes:      1,
		}
	}
	return nil
}

func (p *Processor) NodesOk() error {
	if p.nodes.AllConnected() {
		return nil
	}
	return errors.New("One or more nodes in this cluster are disconnected. In order to prevent from data inconsistent we rejected your request")
}

func (p *Processor) SyncBroadcastHandler() {
	for {
		msg := <-p.syncBroadcastChan
		atomic.AddInt64(&p.Stats.CommandSync, 1)
		p.ClientListLock.Lock()
		for _, c := range p.ClientList {
			if p.ContainChannel(c, msg.Channel) {
				p.Write(c, msg, []byte{})
			}
		}
		p.ClientListLock.Unlock()
	}
}

func (p *Processor) syncClients(channel string, msg []byte) {
	payload := structure.NewSyncResponse()
	payload.Channel = channel
	payload.Message = msg
	p.syncBroadcastChan <- payload
}

func (p *Processor) stats(reqTime int64) interface{} {
	return p.genStatsInfo(reqTime)
}

func (p *Processor) nodesStats() interface{} {
	return p.genClusterStatsInfo()
}

func (p *Processor) nodesMemStats() interface{} {
	return p.getMemStats()
}

func (p *Processor) get(client *Client) interface{} {
	msgMap := make(map[string][]byte)
	client.ChannelLock.Lock()
	for _, channel := range client.Channels {
		msgMap[channel] = p.store.Get(channel)
	}
	client.ChannelLock.Unlock()
	resp := structure.NewGetResponse()
	resp.Channel = msgMap
	return resp
}

func (p *Processor) subscribe(client *Client, channels []string) interface{} {
	client.ChannelLock.Lock()
	client.Channels = channels
	client.ChannelLock.Unlock()
	resp := structure.NewSubscribeResponse()
	resp.Channel = channels
	return resp
}

func (p *Processor) set(channel string, msg []byte) interface{} {
	resp := structure.NewSetResponse()
	err := p.NodesOk()
	if err != nil {
		resp.Status = 0
		resp.Info = err.Error()
		return resp
	}
	err = p.store.Update(channel, msg)
	if err != nil {
		resp.Status = 0
		resp.Info = err.Error()
		return resp
	}
	p.nodes.NodeSync(channel, msg)
	p.syncClients(channel, msg)

	resp.Status = 1
	resp.Info = "OK"
	return resp
}

func (p *Processor) sync(channel string, msg []byte) interface{} {
	if p.store.UpdateWithoutDb(channel, msg) == nil {
		p.nodes.NodeSync(channel, msg)
		p.syncClients(channel, msg)
	}
	return nil
}

func (p *Processor) errResponse(client *Client, op string, errNotice string) interface{} {
	resp := structure.NewErrorResponse()
	resp.Info = errNotice
	resp.Op = op
	return resp
}

func (p *Processor) execCommand(client *Client, cmd *structure.Command) (interface{}, error) {

	if cmd.Op == "p" {
		p.Write(client, nil, p.pingResponseBytes)
		return nil, nil
	}

	err := p.CheckPsk(cmd.Psk)
	if err != nil {
		return nil, err
	}

	switch cmd.Op {

	case "stats":
		if !(cmd.Reqtime > 0) {
			return p.errResponse(client, "stats", "No Reqtime!"), nil
		}
		return p.stats(cmd.Reqtime), nil

	case "cluster_stats":
		return p.nodesStats(), nil

	case "mem_stats":
		return p.nodesMemStats(), nil

	case "get":
		return p.get(client), nil

	case "subscribe":
		if cmd.Channel == nil {
			return p.errResponse(client, "subscribe", "You must specific valid 'Channel' field for the channels you want to subscribe"), nil
		}
		return p.subscribe(client, cmd.Channel), nil

	case "set":
		if cmd.Channel == nil || len(cmd.Channel) != 1 {
			return p.errResponse(client, "set", "You must specific valid 'Channel' field for the channels you want to update"), nil
		}
		if cmd.Message == nil {
			return p.errResponse(client, "set", "You must specific valid 'Message' for those channels"), nil
		}
		return p.set(cmd.Channel[0], cmd.Message), nil

	case "sync":
		if cmd.Channel == nil || len(cmd.Channel) != 1 {
			return errors.New("Invalid sync command received, no valid 'Channel'"), nil
		}
		if cmd.Message == nil {
			return errors.New("Invalid sync command received, no valid 'Message'"), nil
		}
		return p.sync(cmd.Channel[0], cmd.Message), nil

	default:
		return p.errResponse(client, "Unknown", "Unrecognized Command"), nil
	}
}

func (p *Processor) CheckPsk(psk []byte) error {
	if bytes.Equal(psk, p.Config.Psk) || len(p.Config.Psk) == 0 {
		return nil
	}
	return errors.New("Invalid Access Key!!!")
}

func (p *Processor) SendExitSig(client *Client, sig byte) {
	client.ExitChan <- sig
}

func (p *Processor) RemoveMonitor(client *Client) {

	flag := <-client.ExitChan

	p.ClientListLock.Lock()
	delete(p.ClientList, client.Name)
	p.ClientListLock.Unlock()

	if flag == 'w' {
		client.Conn.Close()
		<-client.ExitChan
		close(client.Outgoing)
	} else {
		close(client.Outgoing)
		<-client.ExitChan
		client.Conn.Close()
	}

	close(client.ExitChan)
}

func (p *Processor) ClientWriter(client *Client) {
	var rawbytes []byte
	var payload WritePayload
	var err error

	defer p.SendExitSig(client, 'w')

	for payload = range client.Outgoing {
		client.Conn.SetWriteDeadline(time.Now().Add(time.Second))
		if payload.isBytes == 0 {
			rawbytes, _ = msgpack.Marshal(payload.Payload)
			_, err = client.Conn.Write(rawbytes)
		} else {
			_, err = client.Conn.Write(payload.PayloadBytes)
		}
		if err != nil {
			return
		}
	}
}

func (p *Processor) ClientReader(client *Client) {
	var err error
	var output interface{}

	defer p.SendExitSig(client, 'r')

	decoder := msgpack.NewDecoder(client.Conn)
	cmd := &structure.Command{}
	for {
		client.Conn.SetReadDeadline(time.Now().Add(p.Config.ConnExpires))
		err = decoder.Decode(cmd)
		if err != nil {
			return
		}

		atomic.AddInt64(&p.Stats.Commands, 1)

		output, err = p.execCommand(client, cmd)

		if err != nil {
			atomic.AddInt64(&p.Stats.RejectedConnections, 1)
			return
		}

		if output != nil {
			if p.Write(client, output, []byte{}) != nil {
				log.Println("Message dropped due to full of buffer, closing connection to", client.Name)
				return
			}
		}
	}
}

func (p *Processor) ClientHandler(conn net.Conn) {
	name := conn.RemoteAddr().String()
	newClient := &Client{
		Name:        name,
		Outgoing:    make(chan WritePayload, 5),
		Channels:    []string{},
		ChannelLock: sync.Mutex{},
		Conn:        conn,
		ExitChan:    make(chan byte),
	}
	p.ClientListLock.Lock()
	p.ClientList[name] = newClient
	p.ClientListLock.Unlock()

	atomic.AddInt64(&p.Stats.TotalConnections, 1)

	go p.RemoveMonitor(newClient)
	go p.ClientWriter(newClient)
	go p.ClientReader(newClient)
}

func (p *Processor) serveTcp() {
	log.Println("Starting TCP socket at", p.Config.TcpListen)
	ln, err := net.Listen("tcp", p.Config.TcpListen)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer ln.Close()

	log.Println("TCP service started")
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		p.ClientHandler(conn)
	}
}

func (p *Processor) statsInfoHandler() {
	for {
		p.currentStatsLock.Lock()

		p.currentStats.Uptime = time.Now().Unix() - p.Stats.startTime

		p.ClientListLock.Lock()
		p.currentStats.Connections_current = len(p.ClientList)
		p.ClientListLock.Unlock()

		p.currentStats.Connections_total = atomic.LoadInt64(&p.Stats.TotalConnections)

		p.currentStats.Connections_rejected = atomic.LoadInt64(&p.Stats.RejectedConnections)

		p.currentStats.Commands = atomic.LoadInt64(&p.Stats.Commands)
		p.currentStats.CommandSync = atomic.LoadInt64(&p.Stats.CommandSync)

		p.currentStats.Channels_count = p.store.MsgCount()
		p.currentStats.Storage_trend = p.store.DbStatus()
		p.currentStats.Sync_lasts = p.store.GetSyncPeriod()
		p.currentStats.Last_sync = p.store.GetLastSync()

		p.currentStatsLock.Unlock()

		time.Sleep(time.Millisecond * 1100)
	}
}

func (p *Processor) genStatsInfo(reqTime int64) structure.NodeStatus {
	var stats structure.NodeStatus
	p.currentStatsLock.Lock()
	statsStr, _ := msgpack.Marshal(p.currentStats)
	p.currentStatsLock.Unlock()
	msgpack.Unmarshal(statsStr, &stats)
	stats.Reqtime = reqTime
	statsStr = nil
	return stats
}

func (p *Processor) genClusterStatsInfo() interface{} {
	return p.nodes.Stats()
}

func (p *Processor) getMemStats() interface{} {
	return memstats.Stats()
}

func (p *Processor) ServeForever() {
	p.init()
	p.serveHttp()
	p.serveTcp()
}

func (p *Processor) init() {

	if len(p.Config.Psk) == 0 {
		log.Println("No PSK specificed, you are strongly recommended to set a secret key!!!")
	}

	p.syncBroadcastChan = make(chan structure.SyncResponse)
	p.ClientList = make(map[string]*Client)
	p.pingResponseBytes, _ = msgpack.Marshal(structure.NewPingResponse())
	p.currentStats = structure.NodeStatus{}

	go p.SyncBroadcastHandler()

	p.Stats.startTime = time.Now().Unix()

	p.store = storemidware.StoreMidware{
		Dsn:                p.Config.Dsn,
		SyncInterval:       p.Config.SyncInterval,
		ChangeNotification: p.syncBroadcastChan,
	}
	p.store.Init()

	p.nodes = nodepool.Pool{
		NodeAddrs: p.Config.Nodes,
		Psk:       p.Config.Psk,
	}
	p.nodes.Init()

	go p.statsInfoHandler()
}
