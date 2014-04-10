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
// "github.com/fangli/msgfiber/clientpool"
// "github.com/fangli/msgfiber/nodepool"
// "github.com/fangli/msgfiber/parsecfg"
// "github.com/fangli/msgfiber/storemidware"
// "github.com/fangli/msgfiber/structure"
// "time"
)

// func (p *Client) setMsg(conn net.Conn, channel string, msg []byte) structure.SetResponse {
// 	resp := structure.NewSetResponse()

// 	if !p.cluster.AllConnected() {
// 		resp.Status = 0
// 		resp.Info = "One or more nodes in this cluster are disconnected. In order to prevent from data inconsistent we rejected your request"
// 		return resp
// 	}

// 	err := p.store.Update(channel, msg)
// 	if err != nil {
// 		resp.Status = 0
// 		resp.Info = err.Error()
// 	} else {
// 		resp.Status = 1
// 		resp.Info = "Channel updated succeed"
// 	}
// 	if err == nil {
// 		// p.cluster.NodeSync(channel, msg)
// 		p.(channel, msg, conn)
// 	}
// 	return resp
// }

// func (p *Processor) genStatsInfo() structure.NodeStatus {
// 	var status structure.NodeStatus
// 	status.Uptime = time.Now().Unix() - p.startTime
// 	status.Status = 1
// 	status.Op = "stats"
// 	status.Subscriber_count = p.clients.ClientCount()
// 	status.Pending_msg_count = p.clients.PendingMsgCount()
// 	status.Channels_count = p.store.MsgCount()
// 	status.Storage_trend = p.store.DbStatus()
// 	return status
// }

// func (p *Processor) genClusterStatsInfo() []structure.ClusterNodeStatus {
// 	return p.cluster.Stats()
// }

// func (p *Processor) ServeForever() {

// 	p.init()

// 	// p.serveHttp()
// 	p.serveTcp()
// }

// func (p *Processor) init() {
// 	p.startTime = time.Now().Unix()

// 	p.clients = clientpool.Pool{}
// 	p.clients.Init()

// 	p.store = storemidware.StoreMidware{
// 		Dsn:                p.Config.Dsn,
// 		SyncInterval:       p.Config.SyncInterval,
// 		ChangeNotification: p.broadcastChan,
// 	}
// 	p.store.Init()

// 	p.cluster = nodepool.Pool{
// 		Nodes: p.Config.Nodes,
// 	}
// 	p.cluster.Init()

// }
