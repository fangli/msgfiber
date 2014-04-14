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

package main

import (
	"github.com/fangli/msgfiber/parsecfg"
	"github.com/fangli/msgfiber/processor"
	"log"
)

const BANNER = `
 __  __             ______ _ _               
|  \/  |           |  ____(_) |              
| \  / |___  __ _  | |__   _| |__   ___ _ __ 
| |\/| / __|/ _  | |  __| | | '_ \ / _ \ '__|
| |  | \__ \ (_| | | |    | | |_) |  __/ |   
|_|  |_|___/\__, | |_|    |_|_.__/ \___|_|   
             __/ |                           
            |___/                            
`

func main() {
	cfg := parsecfg.Parse()

	log.Println("Starting MsgFiber...")
	log.Println(BANNER)

	app := processor.Processor{
		Config: cfg,
	}
	app.ServeForever()
}
