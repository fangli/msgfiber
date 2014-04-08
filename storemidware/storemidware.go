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

package storemidware

import (
	"bytes"
	"database/sql"
	"errors"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"net"
	"sync"
	"time"
)

type StoreMidware struct {
	Dsn               string
	SyncInterval      time.Duration
	Callback          func(string, []byte, net.Conn)
	db                *sql.DB
	msgPool           map[string][]byte
	storeLock         sync.Mutex
	statusCounterLock sync.Mutex
	dbStatus          [100]int
}

func (s *StoreMidware) Close() error {
	return s.db.Close()
}

func (s *StoreMidware) loadData() {
	var (
		channel string
		msg     []byte
	)
	rows, err := s.db.Query("SELECT channel, msg FROM msgstore")
	if err != nil {
		log.Fatal("Couldn't restore messages from storage: ", err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&channel, &msg)
		if err != nil {
			log.Fatal("Couldn't load message: ", err.Error())
		}
		s.storeLock.Lock()
		s.msgPool[channel] = msg
		s.storeLock.Unlock()
	}
	err = rows.Err()
	if err != nil {
		log.Fatal("Couldn't load message: ", err.Error())
	}
}

func (s *StoreMidware) sync() error {
	var (
		channel string
		msg     []byte
	)
	rows, err := s.db.Query("SELECT channel, msg FROM msgstore")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		err = rows.Scan(&channel, &msg)
		if err != nil {
			return err
		}

		if !bytes.Equal(s.msgPool[channel], msg) {
			s.storeLock.Lock()
			s.msgPool[channel] = msg
			s.storeLock.Unlock()
			s.Callback(channel, msg, nil)
		}
	}
	err = rows.Err()
	if err != nil {
		return err
	}
	return nil
}

func (s *StoreMidware) shiftDbStatus(status int) {
	s.statusCounterLock.Lock()
	defer s.statusCounterLock.Unlock()
	for i := 0; i < len(s.dbStatus)-1; i++ {
		s.dbStatus[i] = s.dbStatus[i+1]
	}
	s.dbStatus[len(s.dbStatus)-1] = status
}

func (s *StoreMidware) periodicSync() {
	for {
		time.Sleep(s.SyncInterval)
		if s.sync() == nil {
			s.shiftDbStatus(1)
		} else {
			s.shiftDbStatus(0)
		}
	}
}

func (s *StoreMidware) Init() bool {
	var err error
	s.msgPool = make(map[string][]byte)

	s.db, err = sql.Open("mysql", s.Dsn)
	if err != nil {
		log.Fatal("Database DSN parameters error: ", err.Error())
	}

	err = s.db.Ping()
	if err != nil {
		log.Fatal("Couldn't connect to database: ", err.Error())
	}

	s.loadData()
	go s.periodicSync()
	return true
}

func (s *StoreMidware) Update(channel string, msg []byte) error {
	if bytes.Equal(msg, s.msgPool[channel]) {
		return errors.New("No changes found")
	}

	_, err := s.db.Exec("INSERT INTO msgstore(channel, msg) VALUES(?, ?) ON DUPLICATE KEY UPDATE msg=?", channel, msg, msg)
	if err != nil {
		return err
	} else {
		s.storeLock.Lock()
		s.msgPool[channel] = msg
		s.storeLock.Unlock()
		return err
	}
}

func (s *StoreMidware) DryUpdate(channel string, msg []byte) error {
	s.storeLock.Lock()
	defer s.storeLock.Unlock()
	if bytes.Equal(msg, s.msgPool[channel]) {
		return errors.New("No changes found")
	} else {
		s.storeLock.Lock()
		s.msgPool[channel] = msg
		s.storeLock.Unlock()
		return nil
	}
}

func (s *StoreMidware) MsgCount() int {
	s.storeLock.Lock()
	defer s.storeLock.Unlock()
	return len(s.msgPool)
}

func (s *StoreMidware) DbStatus() [100]int {
	s.statusCounterLock.Lock()
	defer s.statusCounterLock.Unlock()
	return s.dbStatus
}

func (s *StoreMidware) Get(channel string) []byte {
	s.storeLock.Lock()
	defer s.storeLock.Unlock()
	return s.msgPool[channel]
}

// func (s *StoreMidware) Get(channel string) ([]byte, error) {
// 	var msg []byte
// 	err := s.db.QueryRow("SELECT msg FROM msgstore WHERE channel=?", channel).Scan(&msg)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return msg, nil
// }
