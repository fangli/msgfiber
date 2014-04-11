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

package structure

import (
	"time"
)

type Command struct {
	Op      string
	Channel []string
	Message []byte
	Reqtime int64
}

type NodeStatus struct {
	Channels_count int
	Storage_trend  [100]int
	Connections    int
	Uptime         int64
	Reqtime        int64
}

type ClusterStatus struct {
	Connected         bool
	Delay             string
	SuccessfulRetries int64
	FailedRetries     int64
	Node_stats        interface{}
}

/////////////////////////////////////////////////
//  The response struct form server to client
/////////////////////////////////////////////////

type SubscribeResponse struct {
	Status  int
	Op      string
	Channel []string
}

func NewSubscribeResponse() SubscribeResponse {
	return SubscribeResponse{
		Status: 1,
		Op:     "subscribe",
	}
}

type SetResponse struct {
	Status int
	Op     string
	Info   string
}

func NewSetResponse() SetResponse {
	return SetResponse{
		Status: 1,
		Op:     "set",
	}
}

type GetResponse struct {
	Status  int
	Op      string
	Channel map[string][]byte
}

func NewGetResponse() GetResponse {
	return GetResponse{
		Status: 1,
		Op:     "get",
	}
}

type ErrorResponse struct {
	Status int
	Op     string
	Info   string
}

func NewErrorResponse() ErrorResponse {
	return ErrorResponse{
		Status: 0,
		Op:     "UNKNOWN",
	}
}

type SyncResponse struct {
	Status  int
	Op      string
	Channel string
	Message []byte
}

func NewSyncResponse() SyncResponse {
	return SyncResponse{
		Status: 1,
		Op:     "sync",
	}
}

type PingResponse struct {
	Status int
	Op     string
}

func NewPingResponse() PingResponse {
	return PingResponse{
		Status: 1,
		Op:     "ping",
	}
}

/////////////////////////////////////////////////

/////////////////////////////////////////////////
//  The command struct form client to server
/////////////////////////////////////////////////

type StatsRequest struct {
	Op      string
	Reqtime int64
}

func NewStatsRequest() StatsRequest {
	return StatsRequest{
		Op:      "stats",
		Reqtime: time.Now().UnixNano(),
	}
}
