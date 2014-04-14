#!/usr/bin/env bash -e
# -*- coding:utf-8 -*-

# *************************************************************************
#  This file is a part of msgfiber, A decentralized and distributed message
#  synchronization system

#  Copyright (C) 2014  Fang Li <surivlee@gmail.com> and Funplus, Inc.
# 
#  This program is free software; you can redistribute it and/or modify
#  it under the terms of the GNU General Public License as published by
#  the Free Software Foundation; either version 2 of the License, or
#  (at your option) any later version.
# 
#  This program is distributed in the hope that it will be useful,
#  but WITHOUT ANY WARRANTY; without even the implied warranty of
#  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
#  GNU General Public License for more details.
# 
#  You should have received a copy of the GNU General Public License along
#  with this program; if not, see http://www.gnu.org/licenses/gpl-2.0.html
# ************************************************************************

cd "$( dirname "${BASH_SOURCE[0]}" )"

version=`cat VERSION`
date=`date`
git_ver=`git rev-parse --short HEAD`

rm -rf dist/
mkdir -p dist/linux_i386
mkdir -p dist/linux_x64
mkdir -p dist/osx_x64

echo "Compiling linux/i386"
export GOOS=linux
export GOARCH=386
go build -o "dist/linux_i386/msgfiber" -ldflags "-X github.com/fangli/msgfiber/parsecfg.SYS_BUILD_DATE '$date' -X github.com/fangli/msgfiber/parsecfg.SYS_VER '$version' -X github.com/fangli/msgfiber/parsecfg.SYS_BUILD_VER '$git_ver'" main.go

echo "Compiling linux/amd64"
export GOOS=linux
export GOARCH=amd64
go build -o "dist/linux_x64/msgfiber" -ldflags "-X github.com/fangli/msgfiber/parsecfg.SYS_BUILD_DATE '$date' -X github.com/fangli/msgfiber/parsecfg.SYS_VER '$version' -X github.com/fangli/msgfiber/parsecfg.SYS_BUILD_VER '$git_ver'" main.go

echo "Compiling OSX/amd64"
export GOOS=darwin
export GOARCH=amd64
go build -o "dist/osx_x64/msgfiber" -ldflags "-X github.com/fangli/msgfiber/parsecfg.SYS_BUILD_DATE '$date' -X github.com/fangli/msgfiber/parsecfg.SYS_VER '$version' -X github.com/fangli/msgfiber/parsecfg.SYS_BUILD_VER '$git_ver'" main.go

echo "Compiling OSX/i386"
export GOOS=darwin
export GOARCH=386
go build -o "dist/osx_x64/msgfiber" -ldflags "-X github.com/fangli/msgfiber/parsecfg.SYS_BUILD_DATE '$date' -X github.com/fangli/msgfiber/parsecfg.SYS_VER '$version' -X github.com/fangli/msgfiber/parsecfg.SYS_BUILD_VER '$git_ver'" main.go
