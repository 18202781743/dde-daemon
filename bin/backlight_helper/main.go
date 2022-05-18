/*
 * Copyright (C) 2017 ~ 2018 Deepin Technology Co., Ltd.
 *
 * Author:     jouyouyun <jouyouwen717@gmail.com>
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	dbus "github.com/godbus/dbus"
	"github.com/linuxdeepin/go-lib/dbusutil"
	"github.com/linuxdeepin/go-lib/log"
)

//go:generate dbusutil-gen em -type Manager

const (
	dbusServiceNameV20 = "com.deepin.daemon.helper.Backlight"
	dbusPathV20        = "/com/deepin/daemon/helper/Backlight"
	dbusInterfaceV20   = "com.deepin.daemon.helper.Backlight"

	dbusServiceNameV23 = "org.deepin.daemon.helper.Backlight1"
	dbusPathV23        = "/org/deepin/daemon/helper/Backlight1"
	dbusInterfaceV23   = "org.deepin.daemon.helper.Backlight1"
)

const (
	DisplayBacklight byte = iota + 1
	KeyboardBacklight
)

type Manager struct {
	service *dbusutil.Service
}

var logger = log.NewLogger("backlight_helper")

func (m *Manager) SetBrightness(type0 byte, name string, value int32) *dbus.Error {
	m.service.DelayAutoQuit()
	filename, err := getBrightnessFilename(type0, name)
	if err != nil {
		return dbusutil.ToError(err)
	}

	fh, err := os.OpenFile(filename, os.O_WRONLY, 0666)
	if err != nil {
		return dbusutil.ToError(err)
	}
	defer fh.Close()

	_, err = fh.WriteString(strconv.Itoa(int(value)))
	if err != nil {
		return dbusutil.ToError(err)
	}

	return nil
}

func getBrightnessFilename(type0 byte, name string) (string, error) {
	// check type0
	var subsystem string
	switch type0 {
	case DisplayBacklight:
		subsystem = "backlight"
	case KeyboardBacklight:
		subsystem = "leds"
	default:
		return "", fmt.Errorf("invalid type %d", type0)
	}

	// check name
	if strings.ContainsRune(name, '/') || name == "" ||
		name == "." || name == ".." {
		return "", fmt.Errorf("invalid name %q", name)
	}

	return filepath.Join("/sys/class", subsystem, name, "brightness"), nil
}

func main() {
	m := &Manager{}
	service, err := dbusutil.NewSystemService()
	if err != nil {
		logger.Fatal("failed to new system service:", err)
	}
	m.service = service

	// v20
	err = service.ExportExt(dbusPathV20, dbusInterfaceV20, m)
	if err != nil {
		logger.Fatal("failed to export:", err)
	}
	err = service.RequestName(dbusServiceNameV20)
	if err != nil {
		logger.Fatal("failed to request name:", err)
	}

	// v23
	err = service.ExportExt(dbusPathV23, dbusInterfaceV23, m)
	if err != nil {
		logger.Fatal("failed to export:", err)
	}
	err = service.RequestName(dbusServiceNameV23)
	if err != nil {
		logger.Fatal("failed to request name:", err)
	}

	service.SetAutoQuitHandler(time.Second*30, nil)
	service.Wait()
}
