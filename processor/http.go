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
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

func (c *Processor) writeHttpPayload(w http.ResponseWriter, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Powered-By", "Msgfiber")
	w.Header().Set("Cache-Control", "max-age=0")
	data, _ := json.Marshal(payload)
	w.Write(data)
}

func (c *Processor) httpStatsHandler(w http.ResponseWriter, r *http.Request) {
	status := c.genStatsInfo()
	c.writeHttpPayload(w, status)
}

func (c *Processor) httpClusterStatsHandler(w http.ResponseWriter, r *http.Request) {
	status := c.genClusterStatsInfo()
	c.writeHttpPayload(w, status)
}

func (c *Processor) httpSetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		body := make([]byte, r.ContentLength)
		_, err := r.Body.Read(body)
		if err != nil {
			http.Error(w, "Unable to read request body", 500)
		}

		channel := strings.Trim(r.Header.Get("CHANNEL"), " ")
		if channel != "" && len(body) > 0 {
			resp := c.setMsg(nil, channel, body)
			c.writeHttpPayload(w, resp)
		} else {
			http.Error(w, "Missing HTTP header 'CHANNEL' or body data", 500)
		}

	} else {
		http.Error(w, "Not Found", 404)
	}
}

func (c *Processor) serveHttp() {
	go func() {
		http.HandleFunc("/cluster_stats", c.httpClusterStatsHandler)
		http.HandleFunc("/stats", c.httpStatsHandler)
		http.HandleFunc("/set", c.httpSetHandler)
		err := http.ListenAndServe(c.Config.HttpListen, nil)
		log.Fatal("Can't create HTTP server:", err.Error())
	}()
}
